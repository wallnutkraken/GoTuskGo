// Package panel contains the logic for implementing the gRPC server
package panel

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/wallnutkraken/gotuskgo/controlpanel"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/serial"
	"google.golang.org/grpc"
	"net"

	"github.com/wallnutkraken/gotuskgo/server"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/dbwrap"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/settings"
)

var (
	// ErrBadAuthCode is the error returned to the gRPC client when the authentication code
	// provided is not the one defined in the settings
	ErrBadAuthCode = errors.New("Bad authentication code")
)

const (
	// ChunkSize is the size of a gzipped database chunk
	ChunkSize = 512 * 1024 // 512 KiB
)

// Panel is the gRPC control panel endpoint provider
type Panel struct {
	config settings.GRPC
	srv    *server.Server
	db     Database
}

// New creates a new instance of the Control Panel gRPC API
func New(cfg settings.GRPC, srv *server.Server, db Database) *Panel {
	return &Panel{
		config: cfg,
		srv:    srv,
		db:     db,
	}
}

// ListenAndServe starts the gRPC server, listening on the port provided in the configuration.
func (p *Panel) ListenAndServe() error {
	port := p.config.GetPort()
	// Start listening on the set port
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return errors.Wrap(err, "net")
	}
	serv := grpc.NewServer()
	controlpanel.RegisterControllerServer(serv, p)
	// And serve the gRPC server
	return serv.Serve(lis)
}

// Database is the database interface for dbwrap containing only
// the relevant functions for Panel
type Database interface {
	GetSubscribeErrors() ([]dbwrap.SubscribeError, error)
	PurgeSubscribeErrors() error
	GetAllMessages() ([]dbwrap.Message, error)
}

// GetApplicationErrors is the gRPC endpoint for retrieving a log of application errors that have
// appeared as the application has ran
func (p *Panel) GetApplicationErrors(ctx context.Context, auth *controlpanel.AuthCode) (*controlpanel.AppErrors, error) {
	if auth.Code != p.config.AuthCode { // TODO: Change from error to log
		return nil, ErrBadAuthCode
	}
	logs := p.srv.AllLogs()
	// Create a list of errors for us to return the errors we have in a compatible way
	errorListGRPC := []*controlpanel.ApplicationError{}
	for _, appError := range logs {
		errorListGRPC = append(errorListGRPC, &controlpanel.ApplicationError{
			Error: appError.Message,
			Unix:  appError.UNIX,
		})
	}
	return &controlpanel.AppErrors{
		Error: errorListGRPC,
	}, nil
}

// GetConfig is the gRPC endpoint for getting the JSON-encoded configuration file
func (p *Panel) GetConfig(ctx context.Context, auth *controlpanel.AuthCode) (*controlpanel.SerializedData, error) {
	if auth.Code != p.config.AuthCode {
		return nil, ErrBadAuthCode
	}

	// Marshal the settings JSON
	data, err := json.Marshal(p.srv.GetGlobalSettings())
	if err != nil {
		return nil, errors.Wrap(err, "json")
	}

	return &controlpanel.SerializedData{
		Content: data,
	}, nil
}

// SetConfig provides a gRPC endpoint for updating the configuration file
func (p *Panel) SetConfig(ctx context.Context, params *controlpanel.SetConfigParams) (*controlpanel.Empty, error) {
	if params.Auth.Code != p.config.AuthCode {
		return nil, ErrBadAuthCode
	}

	// Unmarshall the data into the settings object
	config := settings.Application{}
	if err := json.Unmarshal(params.Data.Content, &config); err != nil {
		return nil, errors.Wrap(err, "json")
	}

	// Save it to file
	if err := settings.Save(config); err != nil {
		return nil, errors.Wrap(err, "save")
	}

	// And propogate the changes
	p.config = config.GRPC
	// Run the setting change propogations
	err := p.srv.SetSettings(config)
	return &controlpanel.Empty{}, err
}

// AddToDatabase provides a gRPC endpoint for adding a payload of messages to the database
func (p *Panel) AddToDatabase(ctx context.Context, messages *controlpanel.MessageList) (*controlpanel.Empty, error) {
	if messages.Auth.Code != p.config.AuthCode {
		return nil, ErrBadAuthCode
	}

	err := p.srv.AddMessages(messages.Message)
	return &controlpanel.Empty{}, err
}

// GetDatabase is the gRPC endpoint for getting a gzipped backup of the database messages (not chat IDs)
func (p *Panel) GetDatabase(auth *controlpanel.AuthCode, respStream controlpanel.Controller_GetDatabaseServer) error {
	if auth.Code != p.config.AuthCode {
		return ErrBadAuthCode
	}

	// Get all messages from the database
	messages, err := p.db.GetAllMessages()
	if err != nil {
		return errors.WithMessage(err, "Database Error")
	}

	// Encode the messages
	rawData, err := serial.Marshal(messages)
	if err != nil {
		return errors.Wrap(err, "serial")
	}
	// Check if rawData is smaller or equal to ChunkSize, if so, just send it and return
	if len(rawData) <= ChunkSize {
		err := respStream.Send(&controlpanel.SerializedData{
			Content: rawData,
		})
		// If there's an error, just return it. It's likely the connection is severed.
		return err
	}
	// Start sending it by chunks.
	for len(rawData) > ChunkSize {
		err := respStream.Send(&controlpanel.SerializedData{
			Content: rawData[:ChunkSize],
		})
		// If there's an error, just return it.
		if err != nil {
			return err
		}
		// Move rawData to the right by one chunk
		rawData = rawData[ChunkSize:]
	}
	// And now, just send the final chunk
	return respStream.Send(&controlpanel.SerializedData{
		Content: rawData,
	})
}

// TriggerSendout triggers a GoTuskGo sendout to all available channels
func (p *Panel) TriggerSendout(ctx context.Context, auth *controlpanel.AuthCode) (*controlpanel.Empty, error) {
	if auth.Code != p.config.AuthCode {
		return &controlpanel.Empty{}, ErrBadAuthCode
	}

	return &controlpanel.Empty{}, p.srv.SendOutMessages()
}
