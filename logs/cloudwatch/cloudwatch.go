package cloudwatch

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
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
	group := aws.String(l.Group)
	stream := aws.String(name)

	_, err := l.client.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  group,
		LogStreamName: stream,
	})
	if err != nil {
		return nil, err
	}

	return &writer{
		group:  group,
		stream: stream,
		client: l.client,
	}, nil
}

func (l *Logger) Open(name string) (io.Reader, error) {
	group := aws.String(l.Group)
	stream := aws.String(name)

	return &reader{
		group:  group,
		stream: stream,
		client: l.client,
	}, nil
}

type client interface {
	PutLogEvents(*cloudwatchlogs.PutLogEventsInput) (*cloudwatchlogs.PutLogEventsOutput, error)
	CreateLogStream(*cloudwatchlogs.CreateLogStreamInput) (*cloudwatchlogs.CreateLogStreamOutput, error)
	GetLogEvents(*cloudwatchlogs.GetLogEventsInput) (*cloudwatchlogs.GetLogEventsOutput, error)
}

type RejectedLogEventsInfoError struct {
	Info *cloudwatchlogs.RejectedLogEventsInfo
}

func (e *RejectedLogEventsInfoError) Error() string {
	return fmt.Sprintf("log messages were rejected")
}

// writer is an io.Writer implementation that writes lines to a cloudwatch logs
// stream.
type writer struct {
	group, stream, sequenceToken *string

	client client
}

func (w *writer) Write(b []byte) (int, error) {
	r := bufio.NewReader(bytes.NewReader(b))

	var (
		n      int
		events []*cloudwatchlogs.InputLogEvent
		eof    bool
	)

	for !eof {
		b, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				eof = true
			} else {
				break
			}
		}

		events = append(events, &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(string(b)),
			Timestamp: aws.Int64(now().UnixNano() / 1000000),
		})

		n += len(b)
	}

	resp, err := w.client.PutLogEvents(&cloudwatchlogs.PutLogEventsInput{
		LogEvents:     events,
		LogGroupName:  w.group,
		LogStreamName: w.stream,
		SequenceToken: w.sequenceToken,
	})
	if err != nil {
		return n, err
	}

	if resp.RejectedLogEventsInfo != nil {
		return n, &RejectedLogEventsInfoError{Info: resp.RejectedLogEventsInfo}
	}

	w.sequenceToken = resp.NextSequenceToken

	return n, nil
}

// reader is an io.Reader implementation that streams log lines from cloudwatch
// logs.
type reader struct {
	b bytes.Buffer

	group, stream, nextToken *string

	client client
}

func (r *reader) Read(b []byte) (int, error) {
	if r.b.Len() > 0 {
		return r.b.Read(b)
	}

	resp, err := r.client.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  r.group,
		LogStreamName: r.stream,
		NextToken:     r.nextToken,
	})
	if err != nil {
		return 0, err
	}

	r.nextToken = resp.NextForwardToken

	for _, event := range resp.Events {
		r.b.WriteString(*event.Message)
	}

	if r.nextToken == nil {
		n, err := r.b.Read(b)
		if err != nil {
			return n, err
		}
		return n, io.EOF
	}

	return r.b.Read(b)
}
