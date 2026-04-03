package telegram_bot

import (
	"context"
	"log"
	"strconv"
	"strings"

	"perezvonish/health-tracker/internal/bot"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/domain/user"
	"perezvonish/health-tracker/internal/modules/pills"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ChatBot struct {
	ctx context.Context

	telegramBotApi  *tgbotapi.BotAPI
	userRepo        user.Repository
	dailyReportRepo daily_report.Repository

	router      *bot.Router
	pillsModule *pills.Module
}

func (c *ChatBot) makeBotContext(chatID, userID int64) bot.BotContext {
	return bot.BotContext{
		Ctx:      c.ctx,
		ChatID:   chatID,
		UserID:   userID,
		API:      c.telegramBotApi,
		Sessions: c.router.Sessions(),
		Users:    c.userRepo,
	}
}

func (c *ChatBot) Start() {
	go c.startGettingUpdates()
	go c.startAlertWorker()

	<-c.ctx.Done()
	log.Println("ChatBot is shutting down")
	c.Stop()
}

func (c *ChatBot) Stop() {
	c.telegramBotApi.StopReceivingUpdates()
	log.Println("Telegram bot stopped")
}

func (c *ChatBot) startGettingUpdates() {
	c.telegramBotApi.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false})

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"}
	updates := c.telegramBotApi.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			c.handleMessage(update.Message)
			continue
		}
		if update.CallbackQuery != nil {
			c.handleCallback(update.CallbackQuery)
			continue
		}
	}
}

func (c *ChatBot) handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID
	text := message.Text

	log.Printf("[%s] %s", message.From.UserName, text)

	if !c.isAuthorized(userID) {
		log.Printf("Unauthorized user: %d", userID)
		return
	}

	// /stats может идти с аргументом: "/stats 90" — обрабатываем отдельно до роутера
	cmd := strings.ToLower(strings.TrimPrefix(text, "/"))
	if idx := strings.IndexAny(cmd, " @"); idx != -1 {
		cmd = cmd[:idx]
	}
	if cmd == "stats" {
		days := 30
		parts := strings.Fields(strings.TrimPrefix(text, "/"))
		if len(parts) > 1 {
			days, _ = strconv.Atoi(parts[1])
		}
		c.handleStatsCommand(chatID, userID, days)
		return
	}

	// Команды reports.go (до Этапа 5 остаются здесь)
	switch cmd {
	case "help":
		c.handleHelpCommand(chatID)
		return
	case "today":
		c.handleTodayCommand(chatID, userID)
		return
	case "week":
		c.handleWeekCommand(chatID, userID)
		return
	case "migraine":
		c.handleMigraineCommand(chatID, userID)
		return
	}

	// Всё остальное — diary, pills, unknown — идёт через роутер
	c.router.HandleMessage(c.makeBotContext(chatID, userID), text)
}

func (c *ChatBot) handleCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID
	messageID := callback.Message.MessageID

	c.telegramBotApi.Request(tgbotapi.NewCallback(callback.ID, ""))

	if !c.isAuthorized(userID) {
		log.Printf("Unauthorized user: %d", userID)
		return
	}

	log.Printf("callback from %s: %s", callback.From.UserName, callback.Data)

	c.router.HandleCallback(c.makeBotContext(chatID, userID), messageID, callback.Data)
}

func (c *ChatBot) isAuthorized(telegramUserID int64) bool {
	_, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	return err == nil
}

func (c *ChatBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	c.telegramBotApi.Send(msg) //nolint:errcheck
}

func NewChatBot(
	ctx context.Context,
	botAPI *tgbotapi.BotAPI,
	userRepo user.Repository,
	dailyReportRepo daily_report.Repository,
	router *bot.Router,
	pillsModule *pills.Module,
) Bot {
	return &ChatBot{
		ctx:             ctx,
		telegramBotApi:  botAPI,
		userRepo:        userRepo,
		dailyReportRepo: dailyReportRepo,
		router:          router,
		pillsModule:     pillsModule,
	}
}
