package ecsdecorator

import (
	"github.com/aws/amazon-cloudwatch-agent/translator/config"
	"github.com/aws/amazon-cloudwatch-agent/translator/util/ec2util"
	"os"
)

const (
	SectionKeyHostIP = "host_ip"
)

type HostIP struct{}

func (h *HostIP) ApplyRule(input interface{}) (string, interface{}) {
	if hostIP := os.Getenv(config.HOST_IP); hostIP != "" {
		return SectionKeyHostIP, hostIP
	}
	return SectionKeyHostIP, ec2util.GetEC2UtilSingleton().PrivateIP
}

func init() {
	RegisterRule(SectionKeyHostIP, new(HostIP))
}
