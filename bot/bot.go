// Package bot contains the Telegram Bot, markov brain, as well as the database calls
package bot

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/wallnutkraken/gotuskgo/tuskbrain"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/dbwrap"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/settings"
	"strings"
	"sync"
)

// Bot is the object containing everything to operate the GoTuskGo bot
type Bot struct {
	appSettings settings.Application
	brain       tuskbrain.Brain
	telegram    *tgbotapi.BotAPI
	db          Database
	lock        *sync.Mutex
}

var (
	// ErrTelegramInit is an error for the Telegram API failing to initialize.
	// This error should be only logged, as the service must be able to run
	// with Telegram not working, so that the Telegram API Key can be remotely
	// set via gRPC
	ErrTelegramInit = errors.New("Telegram failed to initialize")
)

// Database is the database interface for dbwrap
type Database interface {
	GetOffset() int
	SetOffset(value int) error
	AddMessage(msg string) error
	GetSubscription(chatID int64) (dbwrap.Subscription, error)
	AddSubscription(chatID int64) error
	Unsubscribe(sub dbwrap.Subscription) error
	GetSubscriptions() ([]dbwrap.Subscription, error)
	AddSubscribeError(chatID int64, message string) error
	GetAllMessages() ([]dbwrap.Message, error)
}

// New creates a new instance of the bot
func New(config settings.Application, db Database) (*Bot, error) {
	// Connect to Telegram
	tg, err := tgbotapi.NewBotAPI(config.Telegram.APIKey)
	tusk := &Bot{
		appSettings: config,
		brain:       tuskbrain.New(config.Brain),
		db:          db,
		lock:        &sync.Mutex{},
	}
	if err != nil {
		return tusk, ErrTelegramInit
	}
	tusk.telegram = tg

	// Initialize it by feeding it from the database
	if err := tusk.FillBrainFromDatabase(); err != nil {
		return nil, err
	}
	return tusk, nil
}

// UpdateSettings changes the settings for the bot and re-initializes the Telegram client,
// As well as the markov length (if different)
func (b *Bot) UpdateSettings(config settings.Application) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	// Check if a telegram reinit is necessary
	if config.Telegram.APIKey != b.appSettings.Telegram.APIKey {
		// Telegram re-init is needed, re-init with new key
		b.telegram, _ = tgbotapi.NewBotAPI(config.Telegram.APIKey)
	}
	// Check if the markov chain length changed
	if config.Brain.ChainLength != b.appSettings.Brain.ChainLength {
		// Re-init the brain with the new length
		b.brain = tuskbrain.New(config.Brain)
		if err := b.FillBrainFromDatabase(); err != nil {
			return err
		}
	}

	// Finally, just replace the settings object
	b.appSettings = config
	return nil
}

// FillBrainFromDatabase fills the markov brain from the messages stored in the database
func (b *Bot) FillBrainFromDatabase() error {
	msgs, err := b.db.GetAllMessages()
	if err != nil {
		return errors.WithMessage(err, "[TUSK]GetAllMessages")
	}
	// Go through and feed all the messages to the chain
	for _, message := range msgs {
		b.brain.Feed(message.Content)
	}
	return nil
}

// GetMessages gets the latest messages from Telegram
func (b *Bot) GetMessages() error {
	if b.telegram == nil {
		return ErrTelegramInit
	}
	// Lock this while it runs, as this interacts with the markov chain
	b.lock.Lock()
	defer b.lock.Unlock()
	// Get the current offset
	offset := b.db.GetOffset()

	// Get the messages from Telegram
	updates, err := b.telegram.GetUpdates(tgbotapi.UpdateConfig{
		Offset: offset,
	})
	if err != nil {
		return errors.Wrap(err, "telegram.GetUpdates")
	}

	// Iterate through every message, update offset every time so that
	// at the end, the offset will be of the last message
	for _, update := range updates {
		offset = update.UpdateID
		if update.Message == nil {
			// Ignore non-messages
			continue
		}
		if strings.HasPrefix(update.Message.Text, "/") {
			// This is a command, trim it and give it to the appropriate Commander
			cmd := trimCommand(update.Message.Text)
			commander, exists := commands[cmd]
			if !exists {
				// No such command, ignore it. Might be for a different bot.
				continue
			}
			if err := commander(update, b); err != nil {
				return errors.Wrapf(err, "commander[%s]", cmd)
			}
		}

		// Save the update content to the database
		if err := b.db.AddMessage(update.Message.Text); err != nil {
			return errors.WithMessagef(err, "AddMessage [%d]", offset)
		}
		// Add it to the markov brain
		b.brain.Feed(update.Message.Text)
	}
	// Update the offset
	if err := b.db.SetOffset(offset + 1); err != nil {
		return errors.WithMessage(err, "SetOffset")
	}
	return nil
}

// AddMessages adds the given array of messages to the database and the markov chain
func (b *Bot) AddMessages(msgs []string) error {
	// Add it to the database first, so if it fails, there's no inconsistency between the database
	// and the chain
	for _, msg := range msgs {
		if err := b.db.AddMessage(msg); err != nil {
			return errors.WithMessage(err, "AddMessage to DB")
		}
	}
	// And add it to the chain
	b.lock.Lock()
	b.brain.Feed(msgs...)
	b.lock.Unlock()
	return nil
}

// sendMessage attempts to send a message to the given chat
func (b *Bot) sendMessage(chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	_, err := b.telegram.Send(msg)
	return err
}

// SendTUSK sends out a TUSK message to all subscribed channels
func (b *Bot) SendTUSK() error {
	// Lock this while it runs, as this interacts with the markov chain
	b.lock.Lock()
	defer b.lock.Unlock()
	subscriptions, err := b.db.GetSubscriptions()
	if err != nil {
		return errors.WithMessage(err, "GetSubscriptions")
	}
	// Generate a message
	message := b.brain.Generate()
	for _, sub := range subscriptions {
		err := b.sendMessage(sub.ChatID, message)
		if err != nil {
			// An error occurred, unsubscribe the chat. Ignore errors.
			b.db.AddSubscribeError(sub.ChatID, err.Error())
		}
	}
	return nil
}
