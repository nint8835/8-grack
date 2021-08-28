package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func registerCommands() {
	Bot.CommandParser.NewCommand("join", "Join a voice channel", joinChannelCommand)
	Bot.CommandParser.NewCommand("play", "Play a track", playCommand)
}

func joinChannelCommand(message *discordgo.MessageCreate, args struct {
	ChannelID string
}) {
	err := Bot.Session.ChannelVoiceJoinManual(message.GuildID, args.ChannelID, false, false)
	if err != nil {
		Bot.Session.ChannelMessageSend(message.ChannelID, "Failed to join channel: "+err.Error())
	}
}

func playCommand(message *discordgo.MessageCreate, args struct {
	URL string
}) {
	resp, err := Bot.LavalinkRequester.LoadTracks(args.URL)
	if err != nil {
		Bot.Session.ChannelMessageSend(message.ChannelID, "Failed to load track: "+err.Error())
		return
	}
	track := resp.Tracks[0]

	if err := Bot.LavalinkConnection.Play(message.GuildID, track.ID); err != nil {
		Bot.Session.ChannelMessageSend(message.ChannelID, "Failed to play track: "+err.Error())
		return
	}
	Bot.Session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Now playing: %s", track.Info.Title))
}
