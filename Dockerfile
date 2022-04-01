FROM golang:latest as BUILDER

MAINTAINER xwzqmxx<986740642@qq.com>

# build binary
WORKDIR /go/src/github.com/opensourceways/robot-gitee-owners-monitor
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 go build -a -o robot-gitee-owners-monitor .

# copy binary config and utils
FROM alpine:3.14
COPY  --from=BUILDER /go/src/github.com/opensourceways/robot-gitee-owners-monitor/robot-gitee-owners-monitor /opt/app/robot-gitee-owners-monitor

ENTRYPOINT ["/opt/app/robot-gitee-owners-monitor"]