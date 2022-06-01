package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/opensourceways/community-robot-lib/config"
	framework "github.com/opensourceways/community-robot-lib/robot-gitee-framework"
	sdk "github.com/opensourceways/go-gitee/gitee"
	"github.com/opensourceways/sync-file-server/grpc/client"
	"github.com/opensourceways/sync-file-server/protocol"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
)

const botName = "owners-monitor"

type iClient interface {
	GetPullRequestChanges(org, repo string, number int32) ([]sdk.PullRequestFiles, error)
}

func newRobot(cli iClient, sfc client.Client) *robot {
	return &robot{cli: cli, sfc: sfc}
}

type robot struct {
	cli iClient
	sfc client.Client
}

func (bot *robot) NewConfig() config.Config {
	return &configuration{}
}

func (bot *robot) getConfig(cfg config.Config) (*configuration, error) {
	if c, ok := cfg.(*configuration); ok {
		return c, nil
	}

	return nil, fmt.Errorf("can't convert to configuration")
}

func (bot *robot) RegisterEventHandler(f framework.HandlerRegitster) {
	f.RegisterPushEventHandler(bot.handlePushEvent)
}

func (bot *robot) handlePushEvent(e *sdk.PushEvent, c config.Config, log *logrus.Entry) error {
	if !e.GetCreated() {
		return nil
	}

	cfg, err := bot.getConfig(c)
	if err != nil {
		return err
	}

	org, repo := e.GetOrgRepo()

	bcfg := cfg.configFor(org, repo)
	if bcfg == nil {
		return fmt.Errorf("no config for this repo: %s/%s", org, repo)
	}

	sha := e.GetAfter()
	ref := e.GetRef()
	index := strings.LastIndex(ref, "/") + 1
	ref = ref[index:]

	return bot.handle(org, repo, ref, sha, bcfg.GetFileNames())
}

func (bot *robot) handle(org, repo, branch, sha string, fileNames []string) error {
	_, err := bot.sfc.SyncRepoFile(
		context.Background(),
		&protocol.SyncRepoFileRequest{
			Branch: &protocol.Branch{
				Org:       org,
				Repo:      repo,
				Branch:    branch,
				BranchSha: sha,
			},
			FileNames: fileNames,
		},
	)

	return err
}

func (bot *robot) ownersFilesChanged(org, repo string, number int32, cfg *botConfig, log *logrus.Entry) ([]string, bool) {
	cs, err := bot.cli.GetPullRequestChanges(org, repo, number)
	if err != nil {
		log.Error(err)

		return nil, false
	}

	cfs := sets.NewString()

	for _, fn := range cfg.FileNames {
		for _, v := range cs {
			if strings.HasSuffix(v.Filename, fn) {
				cfs.Insert(fn)

				break
			}
		}
	}

	return cfs.UnsortedList(), cfs.Len() > 0
}
