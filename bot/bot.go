// Package bot contains the Telegram Bot, markov brain, as well as the database calls
package bot

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/wallnutkraken/gotuskgo/memlog"
	"github.com/wallnutkraken/gotuskgo/stringer"
	"github.com/wallnutkraken/gotuskgo/tuskbrain"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/dbwrap"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/rnn"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/settings"
)

// Bot is the object containing everything to operate the GoTuskGo bot
type Bot struct {
	appSettings        settings.Application
	brain              *tuskbrain.Brain
	telegram           *tgbotapi.BotAPI
	discord            *discordgo.Session
	db                 Database
	logLine            *memlog.Child
	neuralnet          *rnn.Network
	cancelNextTraining chan interface{}
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
	GetAllMessages() ([]dbwrap.Message, error)
}

// New creates a new instance of the bot
func New(config settings.Application, db Database, logLine *memlog.Child) (*Bot, error) {
	tusk := &Bot{
		appSettings:        config,
		brain:              tuskbrain.New(config.Brain),
		db:                 db,
		logLine:            logLine,
		neuralnet:          rnn.New(config.RNN, logLine),
		cancelNextTraining: make(chan interface{}, 8),
	}
	if config.Brain.UseRNN {
		go tusk.NeuralNetworkSevice()
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
		tusk.logLine.ErrorMessage(err, "Failed to initialize Discord")
		return tusk, ErrServiceInit
	}

	// Initialize it by feeding it from the database
	if err := tusk.FillBrainFromDatabase(); err != nil {
		return nil, err
	}
	return tusk, nil
}

// NeuralNetworkSevice runs a service, which trains the neural network
// every n minutes.
//
// n is defined in settings.RNN.TrainMinsPeriod
func (b *Bot) NeuralNetworkSevice() {
	for {
		// Create a time.After channel for the next training
		// Listen to that and cancelNextTraining
		select {
		case <-time.After(time.Duration(b.appSettings.RNN.TrainMinsPeriod) * time.Minute):
			b.trainNetwork()
		case <-b.cancelNextTraining:
			return
		}
	}
}

// trainNetwork trains the RNN with the current database data
func (b *Bot) trainNetwork() {
	msgs, err := b.db.GetAllMessages()
	if err != nil {
		b.logLine.ErrorMessage(err, "ailed getting messages from the database")
	}
	// msgs -> string
	msgStr := make([]string, len(msgs))
	for index, msg := range msgs {
		msgStr[index] = msg.Content
	}

	b.logLine.Logf("Starting training with %d lines", len(msgStr))
	start := time.Now()
	if err := b.neuralnet.Train(msgStr); err != nil {
		b.logLine.ErrorMessage(err, "Failed training the RNN")
		return
	}
	b.logLine.Logf("Finished training in %s", time.Since(start).String())
}

// UpdateSettings changes the settings for the bot and re-initializes the Telegram client,
// As well as the markov length (if different)
func (b *Bot) UpdateSettings(config settings.Application) error {
	var err error
	if config.APIs.Telegram != b.appSettings.APIs.Telegram {
		// Telegram re-init is needed, re-init with new key
		b.telegram, err = tgbotapi.NewBotAPI(config.APIs.Telegram)
		if err != nil {
			b.logLine.ErrorMessage(err, "Error while re-initializing Telegram after settings update")
		}
	}
	// Also, for discord
	if config.APIs.Discord != b.appSettings.APIs.Discord {
		// Discord re-init is needed, re-init with new key
		if err := b.InitDiscord(config.APIs.Discord); err != nil {
			b.logLine.ErrorMessage(err, "Error while re-initializing Discord after settings update")
		}
	}
	// Check if the markov chain length changed
	b.brain.UpdateSettings(config.Brain)
	// Set the settings for the RNN
	b.neuralnet.UpdateSettings(config.RNN)

	// Check if use_neuralnet is different
	if b.appSettings.Brain.UseRNN != config.Brain.UseRNN {
		// Change in settings, start or stop the service depending on what the new setting is
		if config.Brain.UseRNN {
			// It wasn't running, start it
			go b.NeuralNetworkSevice()
		} else {
			// It's running, stop it.
			// NOTE: this will not stop any training currently in progress, it will only
			// stop it from training after this point
			b.cancelNextTraining <- true
		}
	}

	// Finally, just replace the settings object
	b.appSettings = config
	return nil
}

// GenerateN generates count of messages. Preferring the RNN if possible.
// If the RNN returns an error, it is logged, and the default markov
// backend is used for the remainder of the messages
func (b *Bot) GenerateN(count int) []string {
	// First, generate neuralnet responses
	messages, err := b.neuralnet.GenerateN(count)
	if err == nil {
		// No error, just return messages
		return messages
	}
	// An error has occurred, log it, then default to generateMarkovN
	b.logLine.ErrorMessage(err, "Failed generating messages via neural network")
	return b.generateMarkovN(count)
}

// generateMarkovN generates count amount of messages via the markov backend
func (b *Bot) generateMarkovN(count int) []string {
	messages := make([]string, count)
	for index := range messages {
		messages[index] = b.brain.Generate()
	}
	return messages
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

// HandleInline processes and inline request
func (b *Bot) HandleInline(update tgbotapi.Update) error {
	// First, get the messages to send
	messageList := b.GenerateN(5) // TODO: make this 5 a selectable option
	// Create a 5 possible choices
	responses := make([]interface{}, 5)
	for index, msg := range messageList {
		responses[index] = tgbotapi.InlineQueryResultArticle{
			Type:  "article",
			ID:    strconv.Itoa(rand.Int()),
			Title: msg,
			InputMessageContent: tgbotapi.InputTextMessageContent{
				Text: msg,
			},
		}
	}

	_, err := b.telegram.AnswerInlineQuery(tgbotapi.InlineConfig{
		InlineQueryID: update.InlineQuery.ID,
		CacheTime:     0,
		Results:       responses,
	})
	return err
}

// GetMessagesTelegram gets the latest messages from Telegram
func (b *Bot) GetMessagesTelegram() error {
	if b.telegram == nil {
		return ErrServiceInit
	}
	// Lock this while it runs, as this interacts with the markov chain
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
		if update.InlineQuery != nil {
			if err := b.HandleInline(update); err != nil {
				return errors.WithMessage(err, "HandleInline")
			}
		}
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
		// Split the message to find the command only
		commandPieces := stringer.SplitMultiple(strings.ToLower(message.Content), " \n") // TODO: config
		if len(commandPieces) == 0 {
			// Weird, just return
			return
		}
		// Find the command
		cmd, exists := discordCmd[commandPieces[0]]
		if !exists {
			// No command, just return. Better to just ignore messages starting with !, might be commands to other bots
			return
		}
		if err := cmd(message, b); err != nil {
			b.logLine.ErrorMessagef(err, "Discord Error handling command [%s]", message.Content)
		}
		// Return upon finishing handling commands, do not let a command message be saved
		return
	}
	// Just a regular message, add it to the bot
	if err := b.db.AddMessage(message.Content); err != nil {
		b.logLine.ErrorMessagef(err, "Error saving discord message [%s] to database", message.Content)
	}
	b.brain.Feed(message.Content)
}

// AddMessages adds the given array of messages to the database and the markov chain
func (b *Bot) AddMessages(msgs []string) error {
	// Add it to the database first, so if it fai.conls, there's no inconsistency between the database
	// and the chain
	total := len(msgs)
	for index, msg := range msgs {
		if index%100 == 0 {
			// Divisible by 100, log how many are added
			b.logLine.Logf("Added plaintext messages %d/%d", index, total)
			// And also, take a short, 30ms break every 100 entries
			time.Sleep(time.Microsecond * 30)
		}
		// Ignore empty messages
		if msg == "" {
			continue
		}
		if err := b.db.AddMessage(msg); err != nil {
			return errors.WithMessagef(err, "AddMessage to DB [%d]", index)
		}
	}
	b.logLine.Logf("Added plaintext messages %d/%d", total, total)
	// And add it to the chain
	b.brain.Feed(msgs...)
	return nil
}

// sendMessage attempts to send a message to the given chat
func (b *Bot) sendMessage(chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	_, err := b.telegram.Send(msg)
	return err
}