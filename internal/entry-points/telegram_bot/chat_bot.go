package telegram_bot

import (
	"context"
	"log"
	"perezvonish/health-tracker/internal/shared/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ChatBot struct {
	ctx context.Context

	telegramBotApi *tgbotapi.BotAPI
	isDebug        bool
}

func (c *ChatBot) Start() {
	go c.startGettingUpdates()

	<-c.ctx.Done()
	log.Println("ChatBot is shutting down")
	c.Stop()
}

func (c *ChatBot) Stop() {
	c.telegramBotApi.StopReceivingUpdates()
	log.Println("Telegram bot stopped")
}

func (c *ChatBot) startGettingUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := c.telegramBotApi.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			c.telegramBotApi.Send(msg)
		}
	}
}

func NewChatBot(ctx context.Context, config config.TelegramConfig) Bot {
	chatBot := &ChatBot{
		ctx: ctx,
	}

	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	chatBot.telegramBotApi = bot
	chatBot.telegramBotApi.Debug = true

	return chatBot
}
