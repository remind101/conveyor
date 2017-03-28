package stream

import (
	"os"
	"time"
)

func ExampleHeartbeat() {
	w := os.Stdout
	defer close(Heartbeat(w, time.Second)) // close to cleanup resources
}
