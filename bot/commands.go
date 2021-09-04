package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hako/durafmt"
)

func registerCommands() {
	Bot.CommandParser.NewCommand("join", "Join a voice channel", joinChannelCommand)
	Bot.CommandParser.NewCommand("play", "Play a track", playCommand)
	Bot.CommandParser.NewCommand("skip", "Skip the current track", skipCommand)
	Bot.CommandParser.NewCommand("queue", "Show the current queue", queueCommand)
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
	Query string
}

func playCommand(message *discordgo.MessageCreate, args PlayCommandArgs) {
	guildState, found := Bot.State[message.GuildID]
	if !found {
		Bot.Session.ChannelMessageSend(message.ChannelID, ":no_entry_sign:")
		return
	}

	resp, err := Bot.LavalinkRequester.LoadTracks(args.Query)
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
		Query: args.Query,
	})
	guildState.TrackQueued <- struct{}{}

	Bot.Session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Queued track: %s", track.Info.Title))
}

func skipCommand(message *discordgo.MessageCreate, args struct{}) {
	_, found := Bot.State[message.GuildID]
	if !found {
		Bot.Session.ChannelMessageSend(message.ChannelID, ":no_entry_sign:")
		return
	}

	Bot.LavalinkConnection.Stop(message.GuildID)
}

func queueCommand(message *discordgo.MessageCreate, args struct{}) {
	_, found := Bot.State[message.GuildID]
	if !found {
		Bot.Session.ChannelMessageSend(message.ChannelID, ":no_entry_sign:")
		return
	}

	trackFields := []*discordgo.MessageEmbedField{}

	totalDuration := uint(0)

	for index, track := range Bot.State[message.GuildID].Queue {
		trackFields = append(trackFields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("%d. %s", index+1, track.Track.Info.Title),
			Value: fmt.Sprintf("Requested by: %s\nInitial query: %s", track.User.Mention(), track.Query),
		})
		totalDuration += track.Track.Info.Length
	}

	trackCountSuffix := ""

	if len(Bot.State[message.GuildID].Queue) != 1 {
		trackCountSuffix = "s"
	}

	queueEmbed := &discordgo.MessageEmbed{
		Title:  "Queue",
		Fields: trackFields,
		Color:  (120 << 16) + (195 << 8) + (255),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf(
				"%d track%s, totalling %s",
				len(Bot.State[message.GuildID].Queue),
				trackCountSuffix,
				durafmt.Parse(time.Duration(totalDuration)*time.Millisecond)),
		},
	}

	Bot.Session.ChannelMessageSendEmbed(message.ChannelID, queueEmbed)
}
