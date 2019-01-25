// Package server contains the GoTuskGo backend, as well as the control API
package server

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/wallnutkraken/gotuskgo/bot"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/dbwrap"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/serial"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/settings"
)

// Server is the main object for the GoTuskGo server, containing the database, gRPC API,
// and the GoTuskGo bot
type Server struct {
	tusk          *bot.Bot
	config        settings.Application
	logs          []serial.LogLine
	nextMessageAt int64
	lock          *sync.Mutex
}

// AllLogs returns every log stored in ther server's memory
func (s *Server) AllLogs() []serial.LogLine {
	return s.logs
}

// SetSettings sets the settings for all the underlying objects
func (s *Server) SetSettings(cfg settings.Application) error {
	// Update the settings inside this object
	s.lock.Lock()
	s.config = cfg
	s.lock.Unlock()

	// Propogate it downwards to the bot
	return s.tusk.UpdateSettings(cfg)
}

// AddMessages adds the given array of messages to the database and the markov chain
func (s *Server) AddMessages(msgs []string) error {
	return s.tusk.AddMessages(msgs)
}

// GetGlobalSettings returns the global application settings
func (s *Server) GetGlobalSettings() settings.Application {
	return s.config
}

// New creates a new instance of the Server
func New(config settings.Application, db dbwrap.Wrapper) (*Server, error) {
	rand.Seed(time.Now().UnixNano())
	tusk, err := bot.New(config, db)
	return &Server{
		tusk:   tusk,
		config: config,
		logs:   []serial.LogLine{},
		lock:   &sync.Mutex{},
	}, err
}

func (s *Server) setNextMessageTime() {
	// Get the min-max minutes between messages
	s.lock.Lock()
	now := time.Now()
	min := s.config.Messaging.NormalMinMinutes
	max := s.config.Messaging.NormalMaxMinutes
	if now.Hour() > 22 || now.Hour() < 7 {
		min = s.config.Messaging.SleepMinMinutes
		max = s.config.Messaging.SleepMaxMinutes
	}
	s.lock.Unlock()

	// And calculate the amount of minutes until the next one
	distanceFromMin := max - min
	// Check if the distance is not negative
	if distanceFromMin < 0 {
		panic(fmt.Sprintf("Distance From Min is negative, got %d from MIN[%d] MAX[%d]", distanceFromMin, min, max))
	}
	randDistance := rand.Intn(distanceFromMin)
	minutesUntilNext := time.Minute * time.Duration(randDistance+min)
	s.nextMessageAt = time.Now().Add(minutesUntilNext).Unix()
}

// Start the GoTuskGo bot instance
//
// This is a blocking call
func (s *Server) Start() {
	s.setNextMessageTime()
	for {
		if err := s.tusk.GetMessages(); err != nil {
			// Add it to the application errors for remote logging
			s.LogError(err)
		}
		if s.nextMessageAt <= time.Now().Unix() {
			// Time to send out messages
			if err := s.tusk.SendTUSK(); err != nil {
				// Add it to the application errors for remote logging
				s.LogError(err)
			}
			s.setNextMessageTime()
		}

		time.Sleep(time.Second * 5)
	}
}

// SendOutMessages triggers a message sendout in the GoTuskGo bot
func (s *Server) SendOutMessages() error {
	if err := s.tusk.SendTUSK(); err != nil {
		return err
	}
	s.setNextMessageTime()
	return nil
}

// Log adds a message with the current UNIX timestamp to the application
// in-memory logs
func (s *Server) Log(message string) {
	s.logs = append(s.logs, serial.LogLine{
		Message: message,
		UNIX:    time.Now().Unix(),
	})
}

// Logf adds a message with the current UNIX timestamp to the application
// in-memory logs, with formatting
func (s *Server) Logf(message string, args ...interface{}) {
	s.Log(fmt.Sprintf(message, args...))
}

// LogError adds a message with the current UNIX timestamp with the error
// message to the in-memory logs
func (s *Server) LogError(err error) {
	s.Logf("[ERROR] %s", err.Error())
}
