// Package server contains the GoTuskGo backend, as well as the control API
package server

import (
	"math/rand"
	"sync"
	"time"

	"github.com/wallnutkraken/gotuskgo/bot"
	"github.com/wallnutkraken/gotuskgo/memlog"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/dbwrap"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/settings"
)

// Server is the main object for the GoTuskGo server, containing the database, gRPC API,
// and the GoTuskGo bot
type Server struct {
	tusk          *bot.Bot
	config        settings.Application
	nextMessageAt int64
	settingsLock  *sync.Mutex
	tuskLogs      *memlog.Logger
	serverLogger  *memlog.Child
}

// AllLogs returns every log stored in ther server's memory
func (s *Server) AllLogs() []memlog.LogLine {
	return s.tuskLogs.GetAllLogs()
}

// LogChild is a wrapper for memlog's logger NewChild function
func (s *Server) LogChild(packageName string) *memlog.Child {
	return s.tuskLogs.NewChild(packageName)
}

// SetSettings sets the settings for all the underlying objects
func (s *Server) SetSettings(cfg settings.Application) error {
	// Update the settings inside this object
	s.settingsLock.Lock()
	s.config = cfg
	s.settingsLock.Unlock()

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
	tuskLogs := memlog.New()
	serv := &Server{
		config:       config,
		settingsLock: &sync.Mutex{},
		tuskLogs:     tuskLogs,
		serverLogger: tuskLogs.NewChild("server"),
	}
	tusk, err := bot.New(config, db, tuskLogs.NewChild("bot"))
	serv.tusk = tusk

	return serv, err
}

// Start the GoTuskGo bot instance
//
// This is a blocking call
func (s *Server) Start() {
	for {
		if err := s.tusk.GetMessagesTelegram(); err != nil {
			// Add it to the application errors for remote logging
			s.serverLogger.ErrorMessage(err, "GetMessagesTelegram")
		}
		time.Sleep(time.Millisecond * 500)
	}
}