package app

import (
	"context"
	"errors"
	"log"
	"perezvonish/health-tracker/internal/bot"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/domain/pill_tracker"
	"perezvonish/health-tracker/internal/domain/user"
	httpentry "perezvonish/health-tracker/internal/entry-points/http"
	"perezvonish/health-tracker/internal/entry-points/telegram_bot"
	"perezvonish/health-tracker/internal/infrastructure/database"
	"perezvonish/health-tracker/internal/infrastructure/repository"
	"perezvonish/health-tracker/internal/modules/diary"
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
	HTTPServer  *httpentry.Server
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
	c.initHTTPServer(ctx)

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

	// Diary регистрируется после Pills, чтобы onComplete мог вызвать pills.RunAlerts
	diaryModule := diary.New(c.DailyReportRepo, c.UserRepo, func(ctx bot.BotContext) {
		c.PillsModule.RunAlerts(ctx)
	})
	c.Router.Register(diaryModule)

	log.Println("Bot router initialized (diary + pills)")
}

func (c *Container) initRepositories() {
	userRepo := repository.NewUserRepository(c.MongoDB)
	c.UserRepo = repository.NewCachedUserRepository(userRepo)

	c.DailyReportRepo = repository.NewDailyReportRepository(c.MongoDB)
	c.PillTrackerRepo = repository.NewPillTrackerRepository(c.MongoDB)
}

func (c *Container) initTelegramBot(ctx context.Context) {
	botAPI, err := tgbotapi.NewBotAPI(c.Config.Telegram.BotToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", botAPI.Self.UserName)
	botAPI.Debug = true

	c.TelegramBot = telegram_bot.NewChatBot(ctx, botAPI, c.UserRepo, c.DailyReportRepo, c.Router, c.PillsModule)
}

func (c *Container) initHTTPServer(ctx context.Context) {
	c.HTTPServer = httpentry.NewServer(ctx, c.Config.Server, c.UserRepo, c.DailyReportRepo)
}

func (c *Container) Close(ctx context.Context) error {
	var err error
	if c.HTTPServer != nil {
		err = errors.Join(err, c.HTTPServer.Shutdown(ctx))
	}
	if c.TelegramBot != nil {
		c.TelegramBot.Stop()
	}
	if c.MongoDB != nil {
		err = errors.Join(err, c.MongoDB.Close(ctx))
	}
	return err
}
