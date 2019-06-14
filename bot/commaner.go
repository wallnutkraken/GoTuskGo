package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/go-telegram-bot-api/telegram-bot-api"
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
	"/say": Say,
}

var discordCmd = DiscordCommander{
	"!tusk": tuskDiscord,
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
