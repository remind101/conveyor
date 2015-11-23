package cloudwatch

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	now = func() time.Time {
		return time.Unix(1, 0)
	}
}

func TestWriter(t *testing.T) {
	c := new(mockClient)
	w := &writer{
		group:  aws.String("group"),
		stream: aws.String("1234"),
		client: c,
	}

	c.On("PutLogEvents", &cloudwatchlogs.PutLogEventsInput{
		LogEvents: []*cloudwatchlogs.InputLogEvent{
			{Message: aws.String("Hello\n"), Timestamp: aws.Int64(1000)},
			{Message: aws.String("World"), Timestamp: aws.Int64(1000)},
		},
		LogGroupName:  aws.String("group"),
		LogStreamName: aws.String("1234"),
	}).Return(&cloudwatchlogs.PutLogEventsOutput{}, nil)

	n, err := io.WriteString(w, "Hello\nWorld")
	assert.NoError(t, err)
	assert.Equal(t, 11, n)
}

func TestWriter_Rejected(t *testing.T) {
	c := new(mockClient)
	w := &writer{
		group:  aws.String("group"),
		stream: aws.String("1234"),
		client: c,
	}

	c.On("PutLogEvents", &cloudwatchlogs.PutLogEventsInput{
		LogEvents: []*cloudwatchlogs.InputLogEvent{
			{Message: aws.String("Hello\n"), Timestamp: aws.Int64(1000)},
			{Message: aws.String("World"), Timestamp: aws.Int64(1000)},
		},
		LogGroupName:  aws.String("group"),
		LogStreamName: aws.String("1234"),
	}).Return(&cloudwatchlogs.PutLogEventsOutput{
		RejectedLogEventsInfo: &cloudwatchlogs.RejectedLogEventsInfo{
			TooOldLogEventEndIndex: aws.Int64(2),
		},
	}, nil)

	_, err := io.WriteString(w, "Hello\nWorld")
	assert.Error(t, err)
	assert.IsType(t, &RejectedLogEventsInfoError{}, err)
}

func TestReader(t *testing.T) {
	c := new(mockClient)
	r := &reader{
		group:  aws.String("group"),
		stream: aws.String("1234"),
		client: c,
	}

	c.On("GetLogEvents", &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("group"),
		LogStreamName: aws.String("1234"),
	}).Return(&cloudwatchlogs.GetLogEventsOutput{
		Events: []*cloudwatchlogs.OutputLogEvent{
			{Message: aws.String("Hello\n"), Timestamp: aws.Int64(1000)},
			{Message: aws.String("World"), Timestamp: aws.Int64(1000)},
		},
	}, nil)

	b := new(bytes.Buffer)
	n, err := io.Copy(b, r)
	assert.NoError(t, err)
	assert.Equal(t, int64(11), n)
}

type mockClient struct {
	mock.Mock
}

func (c *mockClient) PutLogEvents(input *cloudwatchlogs.PutLogEventsInput) (*cloudwatchlogs.PutLogEventsOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*cloudwatchlogs.PutLogEventsOutput), args.Error(1)
}

func (c *mockClient) CreateLogStream(input *cloudwatchlogs.CreateLogStreamInput) (*cloudwatchlogs.CreateLogStreamOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*cloudwatchlogs.CreateLogStreamOutput), args.Error(1)
}

func (c *mockClient) GetLogEvents(input *cloudwatchlogs.GetLogEventsInput) (*cloudwatchlogs.GetLogEventsOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*cloudwatchlogs.GetLogEventsOutput), args.Error(1)
}
