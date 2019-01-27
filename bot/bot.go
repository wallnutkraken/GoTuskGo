// Package bot contains the Telegram Bot, markov brain, as well as the database calls
package bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/wallnutkraken/gotuskgo/tuskbrain"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/dbwrap"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/serial"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/settings"
	"strings"
	"sync"
	"time"
)

// Bot is the object containing everything to operate the GoTuskGo bot
type Bot struct {
	appSettings settings.Application
	brain       tuskbrain.Brain
	telegram    *tgbotapi.BotAPI
	discord     *discordgo.Session
	db          Database
	lock        *sync.Mutex
	logLine     chan serial.LogLine
}

var (
	// ErrServiceInit is an error for one of the messaging APIs failing to initialize.
	// This error should be only logged, as the service must be able to run
	// with services not working, so that the API Key can be remotely
	// set via gRPC
	ErrServiceInit = errors.New("A messaging service failed to initialize")
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
func New(config settings.Application, db Database, logLine chan serial.LogLine) (*Bot, error) {
	tusk := &Bot{
		appSettings: config,
		brain:       tuskbrain.New(config.Brain),
		db:          db,
		lock:        &sync.Mutex{},
		logLine:     logLine,
	}
	// Connect to Telegram
	tg, err := tgbotapi.NewBotAPI(config.APIs.Telegram)
	if err != nil {
		return tusk, ErrServiceInit
	}
	tusk.telegram = tg

	// Connect to Discord
	if err := tusk.InitDiscord(config.APIs.Discord); err != nil {
		// Return ErrServiceInit above to let the application run
		// without this service, but log the actual error
		logLine <- serial.LogLine{
			Message: err.Error(),
			UNIX:    time.Now().Unix(),
		}
		return tusk, ErrServiceInit
	}

	// Initialize it by feeding it from the database
	if err := tusk.FillBrainFromDatabase(); err != nil {
		return nil, err
	}
	return tusk, nil
}

func (b *Bot) log(message string) {
	b.logLine <- serial.LogLine{
		Message: message,
		UNIX:    time.Now().Unix(),
	}
}

func (b *Bot) logf(message string, args ...interface{}) {
	b.log(fmt.Sprintf(message, args...))
}

// UpdateSettings changes the settings for the bot and re-initializes the Telegram client,
// As well as the markov length (if different)
func (b *Bot) UpdateSettings(config settings.Application) error {
	var err error
	b.lock.Lock()
	defer b.lock.Unlock()
	if config.APIs.Telegram != b.appSettings.APIs.Telegram {
		// Telegram re-init is needed, re-init with new key
		b.telegram, err = tgbotapi.NewBotAPI(config.APIs.Telegram)
		if err != nil {
			b.logf("Error while re-initializing Telegram after settings update: %s", err.Error())
		}
	}
	// Also, for discord
	if config.APIs.Discord != b.appSettings.APIs.Discord {
		// Discord re-init is needed, re-init with new key
		if err := b.InitDiscord(config.APIs.Discord); err != nil {
			b.logf("Error while re-initializing Discord after settings update: %s", err.Error())
		}
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

// GetMessagesTelegram gets the latest messages from Telegram
func (b *Bot) GetMessagesTelegram() error {
	if b.telegram == nil {
		return ErrServiceInit
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
			commander, exists := telegramCmd[cmd]
			if !exists {
				// No such command, ignore it. Might be for a different bot.
				continue
			}
			if err := commander(update, b); err != nil {
				return errors.Wrapf(err, "commander[%s]", cmd)
			}
			// And continue the loop, don't add this message to db/brain
			continue
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

// InitDiscord creates a discord bot, and initializes it with flavour such as "Playing GoTuskGo"
func (b *Bot) InitDiscord(apiKey string) error {
	discord, err := discordgo.New("Bot " + apiKey)
	if err != nil {
		return ErrServiceInit
	}
	b.discord = discord
	discord.AddHandler(b.onDiscordMessage)

	// Open a websocket connection to Discord and begin listening.
	err = discord.Open()
	if err != nil {
		return err
	}

	// Update the status
	if err := discord.UpdateStatus(0, "GoTuskGo"); err != nil {
		return err
	}

	return nil
}

// onDiscordMessage is a function that will be called every time a new message is sent from Discord
func (b *Bot) onDiscordMessage(discord *discordgo.Session, message *discordgo.MessageCreate) {
	// Ignore messages from myself
	if message.Author.ID == discord.State.User.ID {
		return
	}
	// Check if it starts with !, if so, it might be a command.
	if strings.HasPrefix(message.Content, "!") {
		// Find the command
		cmd, exists := discordCmd[message.Content]
		if !exists {
			// No command, just return. Better to just ignore messages starting with !, might be commands to other bots
			return
		}
		if err := cmd(message, b); err != nil {
			b.logf("Discord Error handling command [%s]: %s", message.Content, err.Error())
		}
		// Return upon finishing handling commands, do not let a command message be saved
		return
	}
	// Just a regular message, add it to the bot
	if err := b.db.AddMessage(message.Content); err != nil {
		b.logf("Error saving discord message [%s] to database: %s", message.Content, err.Error())
	}
	b.brain.Feed(message.Content)
}

// AddMessages adds the given array of messages to the database and the markov chain
func (b *Bot) AddMessages(msgs []string) error {
	// Add it to the database first, so if it fai.conls, there's no inconsistency between the database
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

// SendTUSK sends out a TUSK message to all subscribed channels on Telegram.
// It will only reply to discord messages, and never initiate a message.
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
