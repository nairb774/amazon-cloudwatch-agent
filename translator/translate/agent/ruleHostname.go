package agent

import (
	"github.com/aws/amazon-cloudwatch-agent/translator"
	"github.com/aws/amazon-cloudwatch-agent/translator/config"
	"github.com/aws/amazon-cloudwatch-agent/translator/context"
	"os"
)

type Hostname struct {
}

func (h *Hostname) ApplyRule(input interface{}) (returnKey string, returnVal interface{}) {
	defaultValue := ""
	if context.CurrentContext().RunInContainer() {
		defaultValue = os.Getenv(config.HOST_NAME)
	}
	returnKey, returnVal = translator.DefaultCase("hostname", defaultValue, input)
	return
}

func init() {
	h := new(Hostname)
	RegisterRule("hostname", h)
}
