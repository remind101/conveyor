package conveyor

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/remind101/conveyor/builder"
	"golang.org/x/net/context"
)

// BuildQueue represents a queue that can push build requests onto a queue, and
// also pop requests from the queue.
type BuildQueue interface {
	// Push pushes the build request onto the queue.
	Push(context.Context, builder.BuildOptions) error

	// Subscribe returns a channel that can be received on to fetch
	// BuildRequests.
	Subscribe() chan BuildRequest
}

// BuildRequest adds a context.Context to build options.
type BuildRequest struct {
	builder.BuildOptions
	Ctx context.Context
}

// buildQueue is an implementation of the BuildQueue interface that is in memory
// using a channel.
type buildQueue struct {
	queue chan BuildRequest
}

func newBuildQueue(buffer int) *buildQueue {
	return &buildQueue{
		queue: make(chan BuildRequest, buffer),
	}
}

// NewBuildQueue returns a new in memory BuildQueue.
func NewBuildQueue(buffer int) BuildQueue {
	return newBuildQueue(buffer)
}

func (q *buildQueue) Push(ctx context.Context, options builder.BuildOptions) error {
	q.queue <- BuildRequest{
		Ctx:          ctx,
		BuildOptions: options,
	}
	return nil
}

func (q *buildQueue) Subscribe() chan BuildRequest {
	return q.queue
}

type sqsClient interface {
	SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
	ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessageBatch(input *sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error)
}

// SQSBuildQueue is an implementation of the BuildQueue interface backed by
// Amazon SQS.
type SQSBuildQueue struct {
	// QueueURL is the URL for the SQS queue.
	QueueURL string

	// Context is used to generate a context.Context when receiving a
	// message. The zero value is context.Background.
	Context func() context.Context

	sqs sqsClient
}

// NewSQSBuildQueue returns a new SQSBuildQueue instance backed by a
// pre-configured sqs client.
func NewSQSBuildQueue(config *aws.Config) *SQSBuildQueue {
	return &SQSBuildQueue{
		sqs: sqs.New(config),
	}
}

func (q *SQSBuildQueue) Push(ctx context.Context, options builder.BuildOptions) error {
	raw, err := json.Marshal(options)
	if err != nil {
		return err
	}

	input := &sqs.SendMessageInput{
		MessageBody: aws.String(string(raw)),
		QueueUrl:    aws.String(q.QueueURL),
	}

	_, err = q.sqs.SendMessage(input)
	return err
}

// Subscribe enters into a loop and sends BuildRequests to ch. This method
// blocks.
func (q *SQSBuildQueue) Subscribe(ch chan BuildRequest) error {
	for {
		if err := q.receiveMessage(ch); err != nil {
			return err
		}
	}
}

// receiveMessage calls ReceiveMessage and sends the build requests of ch.
func (q *SQSBuildQueue) receiveMessage(ch chan BuildRequest) (err error) {
	var resp *sqs.ReceiveMessageOutput
	resp, err = q.sqs.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: aws.String(q.QueueURL),
	})
	if err != nil {
		return
	}

	var entries []*sqs.DeleteMessageBatchRequestEntry
	defer func() {
		_, err = q.sqs.DeleteMessageBatch(&sqs.DeleteMessageBatchInput{
			QueueUrl: aws.String(q.QueueURL),
			Entries:  entries,
		})
		return
	}()

	for i, m := range resp.Messages {
		var options builder.BuildOptions
		if err = json.Unmarshal([]byte(*m.Body), &options); err != nil {
			return
		}

		ch <- BuildRequest{
			Ctx:          q.context(),
			BuildOptions: options,
		}

		entries = append(entries, &sqs.DeleteMessageBatchRequestEntry{
			Id:            aws.String(fmt.Sprintf("%d", i)),
			ReceiptHandle: m.ReceiptHandle,
		})
	}

	return
}

func (q *SQSBuildQueue) context() context.Context {
	if q.Context == nil {
		return context.Background()
	}

	return q.Context()
}
