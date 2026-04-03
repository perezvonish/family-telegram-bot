package app

import (
	"context"
	"log"
	"perezvonish/health-tracker/internal/bot"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/domain/pill_tracker"
	"perezvonish/health-tracker/internal/domain/user"
	"perezvonish/health-tracker/internal/entry-points/telegram_bot"
	"perezvonish/health-tracker/internal/infrastructure/database"
	"perezvonish/health-tracker/internal/infrastructure/repository"
	"perezvonish/health-tracker/internal/modules/pills"
	"perezvonish/health-tracker/internal/shared/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Container struct {
	Config  *config.Config
	MongoDB *database.MongoDB

	UserRepo        user.Repository
	DailyReportRepo daily_report.Repository
	PillTrackerRepo pill_tracker.Repository

	FeatureFlags bot.FeatureFlagStore
	Router       *bot.Router
	PillsModule  *pills.Module

	TelegramBot telegram_bot.Bot
}

func NewContainer(ctx context.Context, cfg *config.Config, mongoDB *database.MongoDB) *Container {
	c := &Container{
		Config:  cfg,
		MongoDB: mongoDB,
	}

	c.initRepositories()
	c.initFeatureFlags()
	c.initRouter()
	c.initTelegramBot(ctx)

	return c
}

func (c *Container) initFeatureFlags() {
	c.FeatureFlags = bot.NewEnvFeatureFlags()
	log.Println("Feature flags initialized (env)")
}

func (c *Container) initRouter() {
	sessions := bot.NewSessionStore()
	c.Router = bot.NewRouter(sessions, c.FeatureFlags)

	c.PillsModule = pills.New(c.PillTrackerRepo, c.UserRepo)
	c.Router.Register(c.PillsModule)

	log.Println("Bot router initialized with pills module")
}

func (c *Container) initRepositories() {
	userRepo := repository.NewUserRepository(c.MongoDB)
	c.UserRepo = repository.NewCachedUserRepository(userRepo)

	c.DailyReportRepo = repository.NewDailyReportRepository(c.MongoDB)
	c.PillTrackerRepo = repository.NewPillTrackerRepository(c.MongoDB)
}

func (c *Container) initTelegramBot(ctx context.Context) {
	bot, err := tgbotapi.NewBotAPI(c.Config.Telegram.BotToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)
	bot.Debug = true

	c.TelegramBot = telegram_bot.NewChatBot(ctx, bot, c.UserRepo, c.DailyReportRepo, c.PillTrackerRepo, c.Router, c.PillsModule)
}

func (c *Container) Close(ctx context.Context) error {
	return c.MongoDB.Close(ctx)
}
