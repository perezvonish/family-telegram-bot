package telegram_bot

import (
	"context"
	"log"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/domain/user"
	"slices"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	mealsOptions = []string{"Завтрак", "Обед", "Ужин"}
	medsOptions  = []string{"Венлаксор | Утро", "Венлаксор | Вечер", "Триттико"}
)

type ChatBot struct {
	ctx context.Context

	telegramBotApi  *tgbotapi.BotAPI
	sessionStore    *SessionStore
	userRepo        user.Repository
	dailyReportRepo daily_report.Repository
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

	if text == "/start" || text == "/diary" {
		c.enterDiaryScene(chatID)
		return
	}

	session := c.sessionStore.Get(chatID)
	if session == nil || session.Step < 0 {
		msg := tgbotapi.NewMessage(chatID, "Используй /diary чтобы начать дневник здоровья")
		c.telegramBotApi.Send(msg)
		return
	}

	c.handleTextStep(chatID, session, text)
}

func (c *ChatBot) enterDiaryScene(chatID int64) {
	session := c.sessionStore.Reset(chatID)
	session.Step = 0

	msg := tgbotapi.NewMessage(chatID, "Во сколько вчера легла?")
	msg.ReplyMarkup = sleepTimeKeyboard()
	c.telegramBotApi.Send(msg)
}

func (c *ChatBot) handleTextStep(chatID int64, session *Session, text string) {
	switch session.Step {
	case 0:
		session.Answers.SleepTime = text
		session.Step = 1
		c.sendWithKeyboard(chatID, "Во сколько сегодня проснулась?", wakeTimeKeyboard())

	case 1:
		session.Answers.WakeTime = text
		session.Step = 2
		c.sendWithKeyboard(chatID, "Работала сегодня?", yesNoKeyboard())

	case 2:
		session.Answers.WorkedToday = text
		session.Step = 3
		c.sendWithKeyboard(chatID, "Была менструация?", yesNoKeyboard())

	case 3:
		session.Answers.Menstruation = text
		session.Step = 4
		c.sendWithKeyboard(chatID, "Было ли голодание в течение дня?", fastingKeyboard())

	case 4:
		session.Answers.Fasting = text
		session.Step = 5
		c.sendWithKeyboard(chatID, "Была ли физическая активность?", activityKeyboard())

	case 5:
		session.Answers.Activity = text
		session.Step = 6
		c.sendWithInlineKeyboard(chatID, "Что пропускала?", multiSelectKeyboard(mealsOptions, session.Answers.MealsSkipped))

	case 10:
		raw := strings.TrimSpace(strings.ReplaceAll(text, ",", "."))
		dose, err := strconv.ParseFloat(raw, 64)

		if err != nil || dose <= 0 {
			c.sendMessage(chatID, "Нужно ввести число в миллиграммах. Например: 200")
			return
		}

		session.Answers.MigraineDose = dose
		session.Step = 11
		c.sendWithInlineKeyboard(chatID, "Либидо:", scaleKeyboard("libido", true))
	}
}

func (c *ChatBot) handleCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID
	messageID := callback.Message.MessageID
	data := callback.Data

	c.telegramBotApi.Request(tgbotapi.NewCallback(callback.ID, ""))

	if !c.isAuthorized(userID) {
		log.Printf("Unauthorized user: %d", userID)
		return
	}

	session := c.sessionStore.Get(chatID)
	if session == nil {
		return
	}

	log.Printf("callback from %s: %s (step %d)", callback.From.UserName, data, session.Step)

	switch session.Step {
	case 6:
		c.handleMultiSelect(chatID, messageID, session, "mealsSkipped", mealsOptions, data, 7, func() {
			c.sendWithInlineKeyboard(chatID, "Какие таблетки пропустила?", multiSelectKeyboard(medsOptions, session.Answers.MedsIssues))
		})

	case 7:
		c.handleMultiSelect(chatID, messageID, session, "medsIssues", medsOptions, data, 8, func() {
			c.sendWithInlineKeyboard(chatID, "Оцени настроение:", scaleKeyboard("mood", true))
		})

	case 8:
		value := c.parseScaleValue(data)
		session.Answers.Mood = value
		session.Step = 9
		c.sendWithInlineKeyboard(chatID, "Оцени мигрень:", scaleKeyboard("migraine", false))

	case 9:
		value := c.parseScaleValue(data)
		session.Answers.Migraine = value

		if value <= 2 {
			session.Step = 11
			c.sendWithInlineKeyboard(chatID, "Либидо:", scaleKeyboard("libido", true))
		} else {
			session.Step = 10
			msg := tgbotapi.NewMessage(chatID, "Введите дозировку в мг (например 400):")
			msg.ReplyMarkup = removeKeyboard()
			c.telegramBotApi.Send(msg)
		}

	case 11:
		value := c.parseScaleValue(data)
		session.Answers.Libido = value
		session.Step = 12

		log.Printf("FINAL: %+v", session.Answers)

		if err := c.saveDailyReport(userID, session); err != nil {
			log.Printf("Failed to save daily report: %v", err)
			c.sendMessage(chatID, "Ошибка сохранения. Попробуй ещё раз.")
			return
		}

		c.sendMessage(chatID, "Готово ✅")
		c.sessionStore.Delete(chatID)
	}
}

func (c *ChatBot) handleMultiSelect(chatID int64, messageID int, session *Session, field string, options []string, data string, nextStep int, nextQuestion func()) {
	if data == "m:done" {
		session.Step = nextStep
		c.editMessageRemoveKeyboard(chatID, messageID)
		nextQuestion()
		return
	}

	value := strings.TrimPrefix(data, "m:")
	c.toggleValue(session, field, value)

	var selected []string
	if field == "mealsSkipped" {
		selected = session.Answers.MealsSkipped
	} else if field == "medsIssues" {
		selected = session.Answers.MedsIssues
	}

	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, multiSelectKeyboard(options, selected))
	c.telegramBotApi.Send(edit)
}

func (c *ChatBot) toggleValue(session *Session, field, value string) {
	var arr *[]string

	if field == "mealsSkipped" {
		arr = &session.Answers.MealsSkipped
	} else if field == "medsIssues" {
		arr = &session.Answers.MedsIssues
	} else {
		return
	}

	idx := slices.Index(*arr, value)
	if idx >= 0 {
		*arr = slices.Delete(*arr, idx, idx+1)
	} else {
		*arr = append(*arr, value)
	}
}

func (c *ChatBot) parseScaleValue(data string) int {
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return 0
	}
	value, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return value
}

func (c *ChatBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	c.telegramBotApi.Send(msg)
}

func (c *ChatBot) sendWithKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	c.telegramBotApi.Send(msg)
}

func (c *ChatBot) sendWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	c.telegramBotApi.Send(msg)
}

func (c *ChatBot) editMessageRemoveKeyboard(chatID int64, messageID int) {
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
	c.telegramBotApi.Send(edit)
}

func (c *ChatBot) isAuthorized(telegramUserID int64) bool {
	_, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	return err == nil
}

func (c *ChatBot) saveDailyReport(telegramID int64, session *Session) error {
	user, err := c.userRepo.FindByTelegramID(c.ctx, telegramID)
	if err != nil {
		return err
	}

	report := daily_report.NewDailyReport(user.ID)
	report.SleepTime = session.Answers.SleepTime
	report.WakeTime = session.Answers.WakeTime
	report.WorkedToday = session.Answers.WorkedToday
	report.Menstruation = session.Answers.Menstruation
	report.Fasting = session.Answers.Fasting
	report.Activity = session.Answers.Activity
	report.MealsSkipped = session.Answers.MealsSkipped
	report.MedsIssues = session.Answers.MedsIssues
	report.Mood = session.Answers.Mood
	report.Migraine = session.Answers.Migraine
	report.MigraineDose = session.Answers.MigraineDose
	report.Libido = session.Answers.Libido

	return c.dailyReportRepo.Create(c.ctx, report)
}

func NewChatBot(ctx context.Context, bot *tgbotapi.BotAPI, userRepo user.Repository, dailyReportRepo daily_report.Repository) Bot {
	return &ChatBot{
		ctx:             ctx,
		telegramBotApi:  bot,
		sessionStore:    NewSessionStore(),
		userRepo:        userRepo,
		dailyReportRepo: dailyReportRepo,
	}
}
