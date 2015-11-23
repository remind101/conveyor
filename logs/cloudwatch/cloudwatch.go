package cloudwatch

import (
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/ejholmes/cloudwatch"
)

var now = time.Now

type Logger struct {
	Group string

	client *cloudwatchlogs.CloudWatchLogs
}

func NewLogger(group string) *Logger {
	c := cloudwatchlogs.New(defaults.DefaultConfig)
	return &Logger{
		Group:  group,
		client: c,
	}
}

func (l *Logger) Create(name string) (io.Writer, error) {
	_, err := l.client.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(l.Group),
		LogStreamName: aws.String(name),
	})
	if err != nil {
		return nil, err
	}

	return cloudwatch.NewWriter(l.Group, name, l.client), nil
}

func (l *Logger) Open(name string) (io.Reader, error) {
	return cloudwatch.NewReader(l.Group, name, l.client), nil
}
