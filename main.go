package main

import (
	"errors"
	"flag"
	"os"

	"github.com/opensourceways/community-robot-lib/giteeclient"
	"github.com/opensourceways/community-robot-lib/logrusutil"
	liboptions "github.com/opensourceways/community-robot-lib/options"
	framework "github.com/opensourceways/community-robot-lib/robot-gitee-framework"
	"github.com/opensourceways/community-robot-lib/secret"
	"github.com/opensourceways/sync-file-server/grpc/client"
	"github.com/sirupsen/logrus"
)

type options struct {
	service            liboptions.ServiceOptions
	gitee              liboptions.GiteeOptions
	SyncFileServerAddr string
}

func (o *options) Validate() error {
	if err := o.service.Validate(); err != nil {
		return err
	}

	if o.SyncFileServerAddr == "" {
		return errors.New("sync-file-addr cannot be empty, please configure it")
	}

	return o.gitee.Validate()
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options

	o.gitee.AddFlags(fs)
	o.service.AddFlags(fs)

	fs.StringVar(&o.SyncFileServerAddr, "sync-file-addr", "", "the address of sync file server.")
	fs.Parse(args)

	return o
}

func main() {
	logrusutil.ComponentInit(botName)

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	secretAgent := new(secret.Agent)
	if err := secretAgent.Start([]string{o.gitee.TokenPath}); err != nil {
		logrus.WithError(err).Fatal("Error starting secret agent.")
	}

	defer secretAgent.Stop()

	syc, err := client.NewClient(o.SyncFileServerAddr)
	if err != nil {
		logrus.WithError(err).Fatal("init sync file server failed")
	}

	defer syc.Disconnect()

	c := giteeclient.NewClient(secretAgent.GetTokenGenerator(o.gitee.TokenPath))
	r := newRobot(c, syc)

	framework.Run(r, o.service)
}
