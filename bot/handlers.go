package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func registerHandlers() {
	Bot.Session.AddHandler(handleVoiceUpdate)
}

func handleVoiceUpdate(_ *discordgo.Session, update *discordgo.VoiceServerUpdate) {
	fmt.Println(Bot.Session.State.SessionID)
	err := Bot.LavalinkConnection.UpdateVoice(update.GuildID, Bot.Session.State.SessionID, update.Token, update.Endpoint)
	if err != nil {
		log.Error().Err(err).Str("GuildID", update.GuildID).Msg("Failed to update voice connection")
	} else {
		log.Debug().Str("GuildID", update.GuildID).Msg("Updated voice connection")
	}
}
