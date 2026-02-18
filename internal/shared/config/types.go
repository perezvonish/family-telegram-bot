package config

type Config struct {
	Server   ServerConfig
	Mongo    MongoConfig
	Telegram TelegramConfig
}

type ServerConfig struct {
	Host string `env:"SERVER_HOST" envDefault:"localhost"`
	Port int64  `env:"SERVER_PORT" envDefault:"8080"`
}

type MongoConfig struct {
	Host              string `env:"MONGO_HOST" envDefault:"localhost" required:"true"`
	Port              int    `env:"MONGO_PORT" envDefault:"5432" required:"true"`
	Username          string `env:"MONGO_USERNAME" envDefault:"postgres" required:"true"`
	Password          string `env:"MONGO_PASSWORD" envDefault:"postgres" required:"true"`
	DatabaseName      string `env:"MONGO_DATABASE_NAME" envDefault:"postgres" required:"true"`
	ConnectRetryCount int    `env:"MONGO_DATABASE_CONNECT_RETRY_COUNT" envDefault:"10" required:"true"`
}

type TelegramConfig struct {
	BotToken string `env:"TELEGRAM_BOT_TOKEN" envDefault:"" required:"true"`
}
