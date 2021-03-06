package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/wallnutkraken/gotuskgo/stringer"
)

// TgCommander contains functions for dealing with a specific command in Telegram
type TgCommander map[string]TgCommand

// TgCommand is a specific function for dealing with a specific command in Telegram
type TgCommand func(update tgbotapi.Update, bot *Bot) error

// DiscordCommander contains functions for dealing with commands from Discord
type DiscordCommander map[string]DiscordCommand

// DiscordCommand is a specific function for dealing with a command from Discord
type DiscordCommand func(message *discordgo.MessageCreate, bot *Bot) error

var telegramCmd = TgCommander{
	"/subscribe":   Subscribe,
	"/unsubscribe": Unsubscribe,
	"/say":         Say,
}

var discordCmd = DiscordCommander{
	"!tusk": tuskDiscord,
}

// Subscribe deals with commands regarding subscriptions
func Subscribe(update tgbotapi.Update, bot *Bot) error {
	// Check for an existing subscription
	_, err := bot.db.GetSubscription(update.Message.Chat.ID)
	if err != nil && err != gorm.ErrRecordNotFound {
		// General error
		return errors.WithMessage(err, "GetSubscription")
	}
	if err == nil {
		// No error, just send them a message saying you're already subscribed
		return bot.sendMessage(update.Message.Chat.ID, "You're already subscribed here, away with ye!")
	}
	// No subscription found, subscibe them
	if err := bot.db.AddSubscription(update.Message.Chat.ID); err != nil {
		return errors.WithMessage(err, "AddSubscription")
	}
	// And tell them about it
	return bot.sendMessage(update.Message.Chat.ID, "Welcome to GoTuskGo! You've been subscribed!")
}

// Unsubscribe deals with commands regarding unsubsribing
func Unsubscribe(update tgbotapi.Update, bot *Bot) error {
	// Find the subscription
	sub, err := bot.db.GetSubscription(update.Message.Chat.ID)
	if err != nil && err != gorm.ErrRecordNotFound {
		// General error
		return errors.WithMessage(err, "GetSubscription")
	}
	if err == gorm.ErrRecordNotFound {
		// It's not in subscriptions, just ignore it
		return nil
	}
	// The chat is subscribed, unsubscribe it
	return bot.db.Unsubscribe(sub)
}

// Say sends a new message to the specific chat
func Say(update tgbotapi.Update, bot *Bot) error {
	return bot.sendMessage(update.Message.Chat.ID, bot.brain.Generate())
}

func tuskDiscord(message *discordgo.MessageCreate, bot *Bot) error {
	_, err := bot.discord.ChannelMessageSend(message.ChannelID, bot.brain.Generate())
	return err
}

// trimCommand removes anything past the first word in a command string
func trimCommand(cmd string) string {
	cmdParts := stringer.SplitMultiple(cmd, "@ \n\t")
	if len(cmdParts) == 0 {
		return ""
	}
	return cmdParts[0]
}
