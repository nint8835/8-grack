package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func registerCommands() {
	Bot.CommandParser.NewCommand("join", "Join a voice channel", joinChannelCommand)
	Bot.CommandParser.NewCommand("play", "Play a track", playCommand)
	Bot.CommandParser.NewCommand("skip", "Skip the current track", skipCommand)
}

func joinVoiceChannel(guildId string, channelId string, sourceTextChannelId string) error {
	// TODO: Don't allow re-joining the current voice channel
	// TODO: Add flag to kill connection handler
	err := Bot.Session.ChannelVoiceJoinManual(guildId, channelId, false, false)
	if err != nil {
		return fmt.Errorf("error joining voice channel: %w", err)
	}

	Bot.State[guildId] = &GuildState{
		Queue:         []QueueItem{},
		TrackQueued:   make(chan struct{}, 50),
		TrackEnded:    make(chan struct{}, 1),
		TextChannelID: sourceTextChannelId,
	}

	// Immediately write to the channel so the bot knows it's in a ready state - future writes will come from actual tracks ending
	Bot.State[guildId].TrackEnded <- struct{}{}

	go voiceConnectionHandler(guildId)

	return nil
}

func joinChannelCommand(message *discordgo.MessageCreate, args struct{}) {
	guild, err := Bot.Session.State.Guild(message.GuildID)
	if err != nil {
		Bot.Session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Failed to join channel: error retrieving guild: %s", err.Error()))
	}

	for _, state := range guild.VoiceStates {
		if strings.EqualFold(message.Author.ID, state.UserID) {
			err = joinVoiceChannel(message.GuildID, state.ChannelID, message.ChannelID)
			if err != nil {
				Bot.Session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Failed to join channel: %s", err.Error()))
			}
			return
		}
	}

	Bot.Session.ChannelMessageSend(message.ChannelID, "Join can only be used when you're in a voice channel - join a channel and try again.")
}

type PlayCommandArgs struct {
	URL string
}

func playCommand(message *discordgo.MessageCreate, args PlayCommandArgs) {
	guildState, found := Bot.State[message.GuildID]
	if !found {
		Bot.Session.ChannelMessageSend(message.ChannelID, ":no_entry_sign:")
		return
	}

	resp, err := Bot.LavalinkRequester.LoadTracks(args.URL)
	if err != nil {
		Bot.Session.ChannelMessageSend(message.ChannelID, "Failed to load track: "+err.Error())
		return
	}
	if len(resp.Tracks) == 0 {
		Bot.Session.ChannelMessageSend(message.ChannelID, "Failed to load track: no tracks found")
		return
	}
	track := resp.Tracks[0]

	guildState.Queue = append(guildState.Queue, QueueItem{
		User:  message.Author,
		Track: track,
	})
	guildState.TrackQueued <- struct{}{}
}

func skipCommand(message *discordgo.MessageCreate, args struct{}) {
	_, found := Bot.State[message.GuildID]
	if !found {
		Bot.Session.ChannelMessageSend(message.ChannelID, ":no_entry_sign:")
		return
	}

	Bot.LavalinkConnection.Stop(message.GuildID)
}
