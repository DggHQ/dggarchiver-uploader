package config

import (
	"database/sql"
	"os"
	"strings"
	"time"

	log "github.com/DggHQ/dggarchiver-logger"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	_ "modernc.org/sqlite"
)

type Flags struct {
	Verbose bool
}

type NATSConfig struct {
	Host           string
	Topic          string
	NatsConnection *nats.Conn
}

type SQLiteConfig struct {
	URI             string
	DB              *sql.DB
	InsertStatement *sql.Stmt
}

type LBRYConfig struct {
	URI         string
	Author      string
	ChannelName string
}

type PluginConfig struct {
	On           bool
	PathToScript string
}

type Config struct {
	Flags        Flags
	NATSConfig   NATSConfig
	SQLiteConfig SQLiteConfig
	LBRYConfig   LBRYConfig
	PluginConfig PluginConfig
}

func (cfg *Config) loadDotEnv() {
	log.Debugf("Loading environment variables")
	godotenv.Load()

	// Flags
	verbose := strings.ToLower(os.Getenv("VERBOSE"))
	if verbose == "1" || verbose == "true" {
		cfg.Flags.Verbose = true
	}

	// NATS Host Name or IP
	cfg.NATSConfig.Host = os.Getenv("NATS_HOST")
	if cfg.NATSConfig.Host == "" {
		log.Fatalf("Please set the NATS_HOST environment variable and restart the app")
	}

	// NATS Topic Name
	cfg.NATSConfig.Topic = os.Getenv("NATS_TOPIC")
	if cfg.NATSConfig.Topic == "" {
		log.Fatalf("Please set the NATS_TOPIC environment variable and restart the app")
	}

	// SQLite
	cfg.SQLiteConfig.URI = os.Getenv("SQLITE_DB")
	if cfg.SQLiteConfig.URI == "" {
		log.Fatalf("Please set the SQLITE_DB environment variable and restart the app")
	}

	// LBRY
	cfg.LBRYConfig.URI = os.Getenv("LBRY_URI")
	if cfg.LBRYConfig.URI == "" {
		log.Fatalf("Please set the LBRY_URI environment variable and restart the app")
	}
	cfg.LBRYConfig.Author = os.Getenv("LBRY_AUTHOR")
	if cfg.LBRYConfig.Author == "" {
		log.Fatalf("Please set the LBRY_AUTHOR environment variable and restart the app")
	}
	cfg.LBRYConfig.ChannelName = os.Getenv("LBRY_CHANNEL_NAME")
	if cfg.LBRYConfig.ChannelName == "" {
		log.Fatalf("Please set the LBRY_CHANNEL_NAME environment variable and restart the app")
	}

	// Lua Plugins
	plugins := strings.ToLower(os.Getenv("PLUGINS"))
	if plugins == "1" || plugins == "true" {
		cfg.PluginConfig.On = true
		cfg.PluginConfig.PathToScript = os.Getenv("LUA_PATH_TO_SCRIPT")
		if cfg.PluginConfig.PathToScript == "" {
			log.Fatalf("Please set the LUA_PATH_TO_SCRIPT environment variable and restart the app")
		}
	}

	log.Debugf("Environment variables loaded successfully")
}

func (cfg *Config) loadNats() {
	// Connect to NATS server
	nc, err := nats.Connect(cfg.NATSConfig.Host, nil, nats.PingInterval(20*time.Second), nats.MaxPingsOutstanding(5))
	if err != nil {
		log.Fatalf("Wasn't able to declare the AMQP queue: %s", err)
		log.Fatalf("Could not connect to NATS server: %s", err)
	}
	log.Infof("Successfully connected to NATS server: %s", cfg.NATSConfig.Host)
	cfg.NATSConfig.NatsConnection = nc
}

func (cfg *Config) loadSQLite() {
	var err error

	cfg.SQLiteConfig.DB, err = sql.Open("sqlite", cfg.SQLiteConfig.URI)
	if err != nil {
		log.Fatalf("Wasn't able to open the SQLite DB: %s", err)
	}

	_, err = cfg.SQLiteConfig.DB.Exec("CREATE TABLE IF NOT EXISTS uploaded_vods (id text, pubtime text, title text, starttime text, endtime text, ogthumbnail text, thumbnail text, thumbnailpath text, path text, duration integer, claim text, lbry_name text, lbry_normalized_name text, lbry_permanent_url text);")
	if err != nil {
		log.Fatalf("Wasn't able to create the SQLite table: %s", err)
	}

	cfg.SQLiteConfig.InsertStatement, err = cfg.SQLiteConfig.DB.Prepare("INSERT INTO uploaded_vods (id, pubtime, title, starttime, endtime, ogthumbnail, thumbnail, thumbnailpath, path, duration, claim, lbry_name, lbry_normalized_name, lbry_permanent_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);")
	if err != nil {
		log.Fatalf("Wasn't able to prepare the insert statement: %s", err)
	}
}

func (cfg *Config) Initialize() {
	cfg.loadDotEnv()
	cfg.loadNats()
	cfg.loadSQLite()
}
