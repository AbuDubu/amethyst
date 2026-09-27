package mail

import (
	"context"
	"sync"
)

// Recorder is a Mailer for tests: it keeps messages instead of sending them,
// and can be told to fail.
type Recorder struct {
	mu   sync.Mutex
	sent []Message
	// Err, if set, is returned by Send instead of recording.
	Err error
}

func (r *Recorder) Send(_ context.Context, m Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	r.sent = append(r.sent, m)
	return nil
}

// Sent returns the messages recorded so far.
func (r *Recorder) Sent() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Message(nil), r.sent...)
}
