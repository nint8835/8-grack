package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/lukasl-dev/waterlink/entity/event"
	"github.com/lukasl-dev/waterlink/entity/player"
	"github.com/rs/zerolog/log"
)

func registerHandlers() {
	Bot.Session.AddHandler(handleVoiceUpdate)
}

func handleVoiceUpdate(_ *discordgo.Session, update *discordgo.VoiceServerUpdate) {
	err := Bot.LavalinkConnection.UpdateVoice(update.GuildID, Bot.Session.State.SessionID, update.Token, update.Endpoint)
	if err != nil {
		log.Error().Err(err).Str("GuildID", update.GuildID).Msg("Failed to update voice connection")
	} else {
		log.Debug().Str("GuildID", update.GuildID).Msg("Updated voice connection")
	}
}

func lavalinkEventHandler() {
	log.Debug().Msg("Lavalink event handler started")
	for evt := range Bot.LavalinkConnection.Events() {
		switch evt.Type() {
		case event.TrackEnd:
			evt := evt.(player.TrackEnd)
			log.Debug().Str("GuildID", evt.GuildID).Msg("Track ended")
			Bot.State[evt.GuildID].TrackEnded <- struct{}{}
		}
	}
}

func voiceConnectionHandler(guildId string) {
	log.Debug().Str("GuildID", guildId).Msg("Voice connection handler started")

	guildState, _ := Bot.State[guildId]

	for {
		select {
		case <-guildState.TrackEnded:
			break
		}

		select {
		case <-guildState.TrackQueued:
			track := guildState.Queue[0]
			guildState.Queue = guildState.Queue[1:]

			err := Bot.LavalinkConnection.Play(guildId, track.Track.ID)
			if err != nil {
				Bot.Session.ChannelMessageSend(guildState.TextChannelID, "Failed to play track: "+err.Error())
				return
			}
			Bot.Session.ChannelMessageSend(guildState.TextChannelID, fmt.Sprintf("Now playing: %s", track.Track.Info.Title))
		}
	}
}
