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
// 2.1 - Create the message channel
// 2.2 - Create the error channel
// 2.8 - Protect against concurrent writes to log
func New(w io.Writer) *Alog {
	if w == nil {
		w = os.Stdout
	}
	return &Alog{
		dest:    w,
		m:       &sync.Mutex{},     // 2.8.1 - Initialize the `m` field of `Alog` with a new sync.Mutex
		msgCh:   make(chan string), // 2.1.1 - Initialize the message channel
		errorCh: make(chan error),  // 2.2.1 - Create the error channel
	}
}

// Start begins the message loop for the asynchronous logger. It should be initiated as a goroutine to prevent
// the caller from being blocked.
// 2.7 - Update Start() method to call `write()` when message comes in
func (al Alog) Start() {
	// 2.7.1 - Create an infinite loop that receives messages from `al.msgCh`
	for {
		// 2.7.2 - When a message arrives, a goroutine should pass the message along to the `write()` method
		// 2.7.2 - However, for Stage #2, just pass `nil` to replace the `wg` argument -- we won't use goroutines yet
		msg := <-al.msgCh
		go al.write(msg, nil) // 2.7.2 - Start a goroutine that passes the message to `write()`
	}
}

func (al Alog) formatMessage(msg string) string {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	return fmt.Sprintf("[%v] - %v", time.Now().Format("2006-01-02 15:04:05"), msg)
}

// 2.5 - Implement write() method
// 2.6 - Add error handling to write() method
// 2.8 - Protect against concurrent writes to log
// 2.9 - Refactor error reporting
func (al Alog) write(msg string, wg *sync.WaitGroup) {
	al.m.Lock()         // 2.8.2 - Lock the mutex before the `Write()` method call
	defer al.m.Unlock() // 2.8.3 - Unlock the mutex after the `Write()` method call

	// 2.5.1 - Format the `msg` argument using `formatMessage()`
	_, err := al.dest.Write([]byte(al.formatMessage(msg)))
	if err != nil { // 2.6.1 - Capture the error `err` and if it is not `nil` send it to `al.errorCh`
		// 2.9.1 - Use a goroutine to send the error to `al.errorCh`
		go func(err error) {
			al.errorCh <- err
		}(err)
	}
}

func (al Alog) shutdown() {
}

// MessageChannel returns a channel that accepts messages that should be written to the log.
// 2.3 - Implement the MessageChannel() accessor method
func (al Alog) MessageChannel() chan<- string { // 2.3.2 Return a send-only channel
	return al.msgCh // 2.3.1 - Return the message channel
}

// ErrorChannel returns a channel that will be populated when an error is raised during a write operation.
// This channel should always be monitored in some way to prevent deadlock goroutines from being generated
// when errors occur.
// 2.4 - Implement the ErrorChannel() accessor method
func (al Alog) ErrorChannel() <-chan error { // 2.4.2 Return a receive-only channel
	return al.errorCh // 2.4.1 - Return the error channel
}

// Stop shuts down the logger. It will wait for all pending messages to be written and then return.
// The logger will no longer function after this method has been called.
func (al Alog) Stop() {
}

// Write synchronously sends the message to the log output
func (al Alog) Write(msg string) (int, error) {
	return al.dest.Write([]byte(al.formatMessage(msg)))
}
