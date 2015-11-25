package cloudwatch

import (
	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

// Reader is an io.Reader implementation that streams log lines from cloudwatch
// logs.
type Reader struct {
	b bytes.Buffer

	group, stream, nextToken *string

	client client
}

func NewReader(group, stream string, client *cloudwatchlogs.CloudWatchLogs) *Reader {
	return &Reader{
		group:  aws.String(group),
		stream: aws.String(stream),
		client: client,
	}
}

func (r *Reader) Read(b []byte) (int, error) {
	// Drain existing buffered data.
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

	// We want to re-use the existing token in the event that
	// NextForwardToken is nil, which means there's no new messages to
	// consume.
	if resp.NextForwardToken != nil {
		r.nextToken = resp.NextForwardToken
	}

	// If there are no messages, return so that the consumer can read again.
	if len(resp.Events) == 0 {
		return 0, nil
	}

	for _, event := range resp.Events {
		r.b.WriteString(*event.Message)
	}

	return r.b.Read(b)
}
