package cloudwatch

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

// Reader is an io.Reader implementation that streams log lines from cloudwatch
// logs.
type Reader struct {
	group, stream, nextToken *string

	client client

	throttle <-chan time.Time

	b lockingBuffer

	closed bool

	// If an error occurs when getting events from the stream, this will be
	// populated and subsequent calls to Read will return the error.
	err error
}

// http://www.nthelp.com/ascii.htm
const endOfText = '\x03'

func NewReader(group, stream string, client *cloudwatchlogs.CloudWatchLogs) *Reader {
	return newReader(group, stream, client)
}

func newReader(group, stream string, client client) *Reader {
	r := &Reader{
		group:    aws.String(group),
		stream:   aws.String(stream),
		client:   client,
		throttle: time.Tick(readThrottle),
	}
	go r.start()
	return r
}

func (r *Reader) start() {
	for {
		<-r.throttle
		if r.err = r.read(); r.err != nil {
			return
		}
	}
}

func (r *Reader) read() error {

	params := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  r.group,
		LogStreamName: r.stream,
		StartFromHead: aws.Bool(true),
		NextToken:     r.nextToken,
	}

	resp, err := r.client.GetLogEvents(params)

	if err != nil {
		return err
	}

	// We want to re-use the existing token in the event that
	// NextForwardToken is nil, which means there's no new messages to
	// consume.
	if resp.NextForwardToken != nil {
		r.nextToken = resp.NextForwardToken
	}

	// If there are no messages, return so that the consumer can read again.
	if len(resp.Events) == 0 {
		return nil
	}

	for _, event := range resp.Events {
		r.b.WriteString(*event.Message)
	}

	return nil
}

func (r *Reader) Read(b []byte) (int, error) {
	// Return the AWS error if there is one.
	if r.err != nil {
		return 0, r.err
	}

	// If there is not data right now, return. Reading from the buffer would
	// result in io.EOF being returned, which is not what we want.
	if r.b.Len() == 0 {
		return 0, nil
	}

	return r.b.Read(b)
}

func (r *Reader) Close() error {
	_, err := r.b.Write([]byte{endOfText})

	if err != nil {
		return err
	}

	return nil
}

// lockingBuffer is a bytes.Buffer that locks Reads and Writes.
type lockingBuffer struct {
	sync.Mutex
	bytes.Buffer
	closed bool
}

func (r *lockingBuffer) Read(b []byte) (int, error) {
	if r.closed == true {
		return 0, io.EOF
	}

	r.Lock()
	defer r.Unlock()

	n, err := r.Buffer.Read(b)

	if err != nil {
		return n, err
	}

	if n > 0 && b[n-1] == endOfText {
		r.closed = true
		return n, io.EOF
	}

	return n, nil

}

func (r *lockingBuffer) Write(b []byte) (int, error) {
	r.Lock()
	defer r.Unlock()

	return r.Buffer.Write(b)
}
