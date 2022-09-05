package bot

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/lukasl-dev/waterlink"
	"github.com/lukasl-dev/waterlink/entity/track"
	"github.com/nint8835/parsley"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Prefix             string `default:"8g!"`
	DiscordID          string `required:"true" split_words:"true"`
	DiscordToken       string `required:"true" split_words:"true"`
	LogLevel           string `default:"debug" split_words:"true"`
	LavalinkPassphrase string `default:"correct-horse-battery-staple" split_words:"true"`
	LavalinkHost       string `default:"localhost:2333" split_words:"true"`
}

type QueueItem struct {
	User  *discordgo.User
	Track track.Track
	Query string
}

type GuildState struct {
	Queue         []QueueItem
	TextChannelID string
	NowPlaying    *QueueItem

	TrackQueued chan struct{}
	TrackEnded  chan struct{}
}

type Instance struct {
	Session       *discordgo.Session
	Config        Config
	CommandParser *parsley.Parser

	LavalinkConnection waterlink.Connection
	LavalinkRequester  waterlink.Requester

	State map[string]*GuildState
}

var Bot *Instance

func Start() error {
	Bot = &Instance{
		State: make(map[string]*GuildState),
	}

	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Failed to load .env file: %s\n", err.Error())
	}

	var config Config
	err = envconfig.Process("8grack", &config)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	Bot.Config = config

	logLevel, err := zerolog.ParseLevel(Bot.Config.LogLevel)
	if err != nil {
		return fmt.Errorf("error parsing log level: %w", err)
	}
	zerolog.SetGlobalLevel(logLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Debug().Msg("Creating Discord session")
	session, err := discordgo.New("Bot " + Bot.Config.DiscordToken)
	if err != nil {
		return fmt.Errorf("error creating Discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged | discordgo.IntentGuildMessages | discordgo.IntentsMessageContent
	Bot.Session = session

	log.Debug().Msg("Creating command parser")
	parser := parsley.New(Bot.Config.Prefix)
	parser.RegisterHandler(Bot.Session)
	Bot.CommandParser = parser

	log.Debug().Msg("Creating Lavalink connection")
	connOpts := waterlink.NewConnectOptions().WithUserID(Bot.Config.DiscordID).WithPassphrase(Bot.Config.LavalinkPassphrase)
	conn, err := waterlink.Connect(
		context.TODO(),
		url.URL{
			Scheme: "ws",
			Host:   Bot.Config.LavalinkHost,
		},
		connOpts,
	)
	if err != nil {
		return fmt.Errorf("error creating Lavalink connection: %w", err)
	}
	Bot.LavalinkConnection = conn

	log.Debug().Msg("Creating Lavalink requester")
	reqOpts := waterlink.NewRequesterOptions().WithPassphrase(Bot.Config.LavalinkPassphrase)
	req := waterlink.NewRequester(
		url.URL{
			Scheme: "http",
			Host:   Bot.Config.LavalinkHost,
		},
		reqOpts,
	)
	Bot.LavalinkRequester = req

	go lavalinkEventHandler()

	registerHandlers()
	registerCommands()

	log.Debug().Msg("Connecting to Discord")
	err = session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	log.Info().Msg("8-Grack connected to Discord. Use Ctrl-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	log.Info().Msg("8-Grack disconnected from Discord.")

	return nil
}
