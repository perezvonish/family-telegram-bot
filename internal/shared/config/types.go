package config

type Config struct {
	Server   ServerConfig
	Mongo    MongoConfig
	Telegram TelegramConfig
}

type ServerConfig struct {
	Host         string `env:"SERVER_HOST" envDefault:"localhost"`
	Port         int64  `env:"SERVER_PORT" envDefault:"8080"`
	BearerToken  string `env:"HTTP_BEARER_TOKEN" envDefault:""`
	DashboardURL string `env:"DASHBOARD_PUBLIC_URL" envDefault:""`
}

type MongoConfig struct {
	URI               string `env:"MONGO_URI" envDefault:""`
	Host              string `env:"MONGO_HOST" envDefault:"localhost"`
	Port              int    `env:"MONGO_PORT" envDefault:"27017"`
	Username          string `env:"MONGO_USERNAME" envDefault:""`
	Password          string `env:"MONGO_PASSWORD" envDefault:""`
	DatabaseName      string `env:"MONGO_DATABASE_NAME" envDefault:"health_tracker" required:"true"`
	ConnectRetryCount int    `env:"MONGO_CONNECT_RETRY_COUNT" envDefault:"5"`
}

type TelegramConfig struct {
	BotToken string `env:"TELEGRAM_BOT_TOKEN" envDefault:"" required:"true"`
}
