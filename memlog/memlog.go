// Package memlog contains an in-memory logger that can be passed down to other sub-packages
// and includes special naming for these spinoffs
package memlog

import (
	"fmt"
	"sync"
	"time"
)

// Logger contains all messages from all child loggers, and is
// also used to create child loggers
type Logger struct {
	receiveMutex *sync.Mutex
	allLogs      []LogLine
}

// LogLine is a log entry, containing both the message (usually errors)
// as well as the Unix time stamp
type LogLine struct {
	Message string
	UNIX    int64
}

// New creates a new top-level logger
func New() *Logger {
	return &Logger{
		receiveMutex: &sync.Mutex{},
		allLogs:      []LogLine{},
	}
}

// GetAllLogs retreives all messages stored in the Logger
func (l *Logger) GetAllLogs() []LogLine {
	l.receiveMutex.Lock()
	defer l.receiveMutex.Unlock()
	return l.allLogs[:]
}

// watch watches a single channel for messages
func (l *Logger) watch(ch chan string) {
	for {
		message := <-ch
		l.receiveMutex.Lock()
		l.allLogs = append(l.allLogs, LogLine{
			Message: message,
			UNIX:    time.Now().Unix(),
		})
		l.receiveMutex.Unlock()
	}

}

// Child is a child logger of Logger. It contains a package name (defined by user),
// and is able to report back to the main logger
type Child struct {
	packageName string
	parentChan  chan string
}

// NewChild creates a child logger, which reports back to the main logger
func (l *Logger) NewChild(packageName string) *Child {

	// Create the child channel
	channel := make(chan string, 16)

	// Start watching it
	go l.watch(channel)

	// Create a child with that channel
	return &Child{
		packageName: packageName,
		parentChan:  channel,
	}
}

// Log logs a message to the parent channel
func (c *Child) Log(message string) {
	c.parentChan <- fmt.Sprintf("%s: %s", c.packageName, message)
}

// Logf logs a formatted message to the parent channel
func (c *Child) Logf(message string, args ...interface{}) {
	c.Log(fmt.Sprintf(message, args...))
}

// Error logs an error as a log message
func (c *Child) Error(err error) {
	c.Logf("[ERROR] %s", err.Error())
}

// ErrorMessage logs an error with an accompanying message
func (c *Child) ErrorMessage(err error, message string) {
	c.Logf("[ERROR] %s: %s", message, err.Error())
}

// ErrorMessagef logs an error with an accompanying formatted message
func (c *Child) ErrorMessagef(err error, message string, args ...interface{}) {
	c.ErrorMessage(err, fmt.Sprintf(message, args...))
}
