package cloudwatch

import (
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/ejholmes/cloudwatch"
)

func NewLogger(group string) *cloudwatch.Group {
	c := cloudwatchlogs.New(defaults.DefaultConfig)
	return cloudwatch.NewGroup(group, c)
}
