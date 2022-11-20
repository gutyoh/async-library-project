// Package alog provides a simple asynchronous logger that will write to provided io.Writers without blocking calling
// goroutines.
package alog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Alog is a type that defines a logger. It can be used to write log messages synchronously (via the Write method)
// or asynchronously via the channel returned by the MessageChannel accessor.
type Alog struct {
	dest               io.Writer
	m                  *sync.Mutex
	msgCh              chan string
	errorCh            chan error
	shutdownCh         chan struct{}
	shutdownCompleteCh chan struct{}
}

// New creates a new Alog object that writes to the provided io.Writer.
// If nil is provided the output will be directed to os.Stdout.
// 3.1 - Create channels to shutdown logger
func New(w io.Writer) *Alog {
	if w == nil {
		w = os.Stdout
	}
	return &Alog{
		dest:               w,
		m:                  &sync.Mutex{},
		msgCh:              make(chan string),
		errorCh:            make(chan error),
		shutdownCh:         make(chan struct{}), // 3.1.1 - Initialize the shutdown channel
		shutdownCompleteCh: make(chan struct{}), // 3.1.2 - Initialize the shutdown complete channel
	}
}

// Start begins the message loop for the asynchronous logger. It should be initiated as a goroutine to prevent
// the caller from being blocked.
// 3.3 - Update Start() to process messages from shutdownCh
// 3.5 - Wait for all log entries to be written before shutting down
func (al Alog) Start() {
	wg := &sync.WaitGroup{} // 3.5.1 - Create a `sync.WaitGroup` to track the number of pending log entries

	for {
		// 3.3.1 - Add a `select` statement to the loop so messages can be processed from
		// the `al.msgCh` channel and the `al.shutdownCh` channel
		select {
		// 3.3.2 - If a message is received on the `al.msgCh` call `al.write()` to write the message to the log
		case msg := <-al.msgCh:
			wg.Add(1) // 3.5.2 - Increment the `WaitGroup` counter before calling `al.write()`

			// 3.5.3 - Pass the `WaitGroup` to `al.write()` to call `wg.Done()` when the message has been written
			go al.write(msg, wg)

		// 3.3.3 - If a message is received on the `al.shutdownCh` channel, call `al.shutdown()`
		// and break out of the infinite loop
		case <-al.shutdownCh:
			// 3.5.5 - Call the `Wait()` method on the `WaitGroup` when a message is received by `Start()`
			// but before it calls `al.shutdown()`
			wg.Wait()
			al.shutdown()
			return
		}
	}
}

func (al Alog) formatMessage(msg string) string {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	return fmt.Sprintf("[%v] - %v", time.Now().Format("2006-01-02 15:04:05"), msg)
}

// 3.5 - Wait for all log entries to be written before shutting down
func (al Alog) write(msg string, wg *sync.WaitGroup) {
	// 3.5.4 - Defer the call on the `Done()` method on the `WaitGroup` after the message has been written
	defer wg.Done()

	al.m.Lock()
	defer al.m.Unlock()

	_, err := al.dest.Write([]byte(al.formatMessage(msg)))
	if err != nil {
		go func(err error) {
			al.errorCh <- err
		}(err)
	}
}

// 3.2 Implement shutdown() method
// 3.4 Implement the Stop() method
func (al Alog) shutdown() {
	// 3.2.1 - Close the message channel to prevent the logger from receiving new messages:
	close(al.msgCh)
	// 3.4.3 - Send a finished message on the shutdown complete channel to indicate that the shutdown is complete:
	al.shutdownCompleteCh <- struct{}{}
}

// MessageChannel returns a channel that accepts messages that should be written to the log.
func (al Alog) MessageChannel() chan<- string {
	return al.msgCh
}

// ErrorChannel returns a channel that will be populated when an error is raised during a write operation.
// This channel should always be monitored in some way to prevent deadlock goroutines from being generated
// when errors occur.
func (al Alog) ErrorChannel() <-chan error {
	return al.errorCh
}

// Stop shuts down the logger. It will wait for all pending messages to be written and then return.
// The logger will no longer function after this method has been called.
// 3.4 - Implement the Stop() method
func (al Alog) Stop() {
	// 3.4.1 - Send a message on the shutdown channel to initiate the shutdown process
	al.shutdownCh <- struct{}{}
	// 3.4.2 - Wait for the shutdown complete channel to be closed
	<-al.shutdownCompleteCh
}

// Write synchronously sends the message to the log output
func (al Alog) Write(msg string) (int, error) {
	return al.dest.Write([]byte(al.formatMessage(msg)))
}
