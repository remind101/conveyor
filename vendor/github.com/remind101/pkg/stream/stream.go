// Package stream provides types that make it easier to perform streaming io.
package stream

import (
	"fmt"
	"io"
	"time"
)

// Heartbeat sends the null character periodically, to keep the connection alive.
func Heartbeat(outStream io.Writer, interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	t := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-t.C:
				fmt.Fprintf(outStream, "\x00")
				continue
			case <-stop:
				t.Stop()
				return
			}
		}
	}()

	return stop
}
