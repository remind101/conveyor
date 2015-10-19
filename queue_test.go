package conveyor

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBuildQueue(t *testing.T) {
	q := &buildQueue{
		queue: make(chan BuildRequest, 1),
	}

	background := context.Background()
	options := builder.BuildOptions{}
	err := q.Push(background, options)
	assert.NoError(t, err)

	req := <-q.Subscribe()
	assert.Equal(t, req.BuildOptions, options)
	assert.Equal(t, req.Ctx, background)
}

func TestSQSBuildQueue_Push(t *testing.T) {
	c := new(mockSQSClient)
	q := &SQSBuildQueue{
		sqs: c,
	}

	c.On("SendMessage", &sqs.SendMessageInput{
		MessageBody: aws.String(`{"Repository":"remind101/acme-inc","Sha":"abcd","Branch":"master","NoCache":false}`),
		QueueUrl:    aws.String(""),
	}).Return(&sqs.SendMessageOutput{}, nil)

	background := context.Background()
	options := builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "abcd",
	}
	err := q.Push(background, options)
	assert.NoError(t, err)
}

func TestSQSBuildQueue_Subscribe(t *testing.T) {
	c := new(mockSQSClient)
	q := &SQSBuildQueue{
		sqs: c,
	}

	c.On("ReceiveMessage", &sqs.ReceiveMessageInput{
		QueueUrl: aws.String(""),
	}).Return(&sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{
			{
				ReceiptHandle: aws.String("a"),
				Body:          aws.String(`{"Repository":"remind101/acme-inc-1","Sha":"abcd","Branch":"master","NoCache":false}`),
			},
			{
				ReceiptHandle: aws.String("b"),
				Body:          aws.String(`{"Repository":"remind101/acme-inc-2","Sha":"abcd","Branch":"master","NoCache":false}`),
			},
		},
	}, nil)
	c.On("DeleteMessageBatch", &sqs.DeleteMessageBatchInput{
		Entries: []*sqs.DeleteMessageBatchRequestEntry{
			{Id: aws.String("0"), ReceiptHandle: aws.String("a")},
			{Id: aws.String("1"), ReceiptHandle: aws.String("b")},
		},
		QueueUrl: aws.String(""),
	}).Return(&sqs.DeleteMessageBatchOutput{}, nil)

	ch := make(chan BuildRequest, 1)
	go q.Subscribe(ch)

	assert.Equal(t, builder.BuildOptions{
		Repository: "remind101/acme-inc-1",
		Branch:     "master",
		Sha:        "abcd",
	}, (<-ch).BuildOptions)
	assert.Equal(t, builder.BuildOptions{
		Repository: "remind101/acme-inc-2",
		Branch:     "master",
		Sha:        "abcd",
	}, (<-ch).BuildOptions)
}

// mockSQSClient is an implementation of the sqsClient interface for testing.
type mockSQSClient struct {
	mock.Mock
}

func (c *mockSQSClient) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*sqs.SendMessageOutput), args.Error(1)
}

func (c *mockSQSClient) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func (c *mockSQSClient) DeleteMessageBatch(input *sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*sqs.DeleteMessageBatchOutput), args.Error(1)
}
