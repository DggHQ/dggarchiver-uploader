package config

import (
	"context"
	"database/sql"
	"os"
	"strings"

	log "github.com/DggHQ/dggarchiver-logger"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	_ "modernc.org/sqlite"
)

type Flags struct {
	Verbose bool
}

type AMQPConfig struct {
	URI          string
	ExchangeName string
	ExchangeType string
	QueueName    string
	Context      context.Context
	Channel      *amqp.Channel
	connection   *amqp.Connection
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
	AMQPConfig   AMQPConfig
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

	// AMQP
	cfg.AMQPConfig.URI = os.Getenv("AMQP_URI")
	if cfg.AMQPConfig.URI == "" {
		log.Fatalf("Please set the AMQP_URI environment variable and restart the app")
	}
	cfg.AMQPConfig.ExchangeName = ""
	cfg.AMQPConfig.ExchangeType = "direct"
	cfg.AMQPConfig.QueueName = "worker"

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

func (cfg *Config) loadAMQP() {
	var err error

	cfg.AMQPConfig.Context = context.Background()

	cfg.AMQPConfig.connection, err = amqp.Dial(cfg.AMQPConfig.URI)
	if err != nil {
		log.Fatalf("Wasn't able to connect to the AMQP server: %s", err)
	}

	cfg.AMQPConfig.Channel, err = cfg.AMQPConfig.connection.Channel()
	if err != nil {
		log.Fatalf("Wasn't able to create the AMQP channel: %s", err)
	}

	_, err = cfg.AMQPConfig.Channel.QueueDeclare(
		cfg.AMQPConfig.QueueName, // queue name
		true,                     // durable
		false,                    // auto delete
		false,                    // exclusive
		false,                    // no wait
		nil,                      // arguments
	)
	if err != nil {
		log.Fatalf("Wasn't able to declare the AMQP queue: %s", err)
	}
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
	cfg.loadAMQP()
	cfg.loadSQLite()
}
