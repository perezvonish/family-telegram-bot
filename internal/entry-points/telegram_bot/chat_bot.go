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
	mealsOptions = []string{"Завтрак", "Перекус", "Обед", "Ужин"}
	medsOptions  = []string{"Мальтофер", "Витамин Д", "Метилфолат", "Витамин B12", "Эвика"}
	libidoLabels = []string{"Отсутствует", "Слабое", "Среднее", "Повышенное", "Высокое"}
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

	c.handleTextStep(chatID, userID, session, text)
}

func (c *ChatBot) enterDiaryScene(chatID int64) {
	session := c.sessionStore.Reset(chatID)
	session.Step = 0
	c.sendWithKeyboard(chatID, "Во сколько вчера легла?", sleepTimeKeyboard())
}

func (c *ChatBot) handleTextStep(chatID int64, userID int64, session *Session, text string) {
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
		c.sendWithKeyboard(chatID, "Сегодня была менструация?", yesNoKeyboard())

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
		c.sendWithInlineKeyboard(chatID, "Все приемы пищи были?", multiSelectKeyboard(mealsOptions, session.Answers.MealsSkipped))

	case 11:
		dose := strings.TrimSpace(text)
		if dose == "" {
			c.sendMessage(chatID, "Введи препарат и дозировку. Например: Ибуклин 600")
			return
		}

		session.Answers.MigraineDose = dose
		session.Step = 12
		c.sendWithInlineKeyboard(chatID, "Оцени уровень либидо:", labeledScaleKeyboard("libido", libidoLabels))
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
			c.sendWithInlineKeyboard(chatID, "Все таблетки выпила?", multiSelectKeyboard(medsOptions, session.Answers.MedsIssues))
		})

	case 7:
		c.handleMultiSelect(chatID, messageID, session, "medsIssues", medsOptions, data, 8, func() {
			c.sendWithInlineKeyboard(chatID, "Какое у тебя было настроение:", scaleRangeKeyboard("mood", 1, 10, true))
		})

	case 8:
		value := c.parseScaleValue(data)
		session.Answers.Mood = value
		session.Step = 9
		c.sendWithInlineKeyboard(chatID, "Оцени мигрень (0 - не было | 5 - вызываем скорую):", scaleRangeKeyboard("migraine", 0, 5, false))

	case 9:
		value := c.parseScaleValue(data)
		session.Answers.Migraine = value

		if value >= 1 {
			session.Step = 10
			c.sendWithInlineKeyboard(chatID, "Где локализована боль?", migraineSideKeyboard())
			return
		}

		session.Step = 12
		c.sendWithInlineKeyboard(chatID, "Оцени уровень либидо:", labeledScaleKeyboard("libido", libidoLabels))

	case 10:
		session.Answers.MigraineSide = strings.TrimPrefix(data, "mside:")

		if session.Answers.Migraine >= 2 {
			session.Step = 11
			msg := tgbotapi.NewMessage(chatID, "Какой препарат принимала? (например: Ибуклин 600)")
			msg.ReplyMarkup = removeKeyboard()
			c.telegramBotApi.Send(msg)
			return
		}

		session.Step = 12
		c.sendWithInlineKeyboard(chatID, "Оцени уровень либидо:", labeledScaleKeyboard("libido", libidoLabels))

	case 12:
		value := c.parseScaleValue(data)
		session.Answers.Libido = value
		c.finishSurvey(chatID, userID, session)
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

func (c *ChatBot) sendPhotoWithInlineKeyboard(chatID int64, assetPath string, caption string, keyboard tgbotapi.InlineKeyboardMarkup) {
	data, err := scaleAssets.ReadFile(assetPath)
	if err != nil {
		log.Printf("Failed to read asset %s: %v", assetPath, err)
		c.sendWithInlineKeyboard(chatID, caption, keyboard)
		return
	}
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: assetPath, Bytes: data})
	photo.Caption = caption
	photo.ReplyMarkup = keyboard
	c.telegramBotApi.Send(photo)
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
	report.MigraineSide = session.Answers.MigraineSide
	report.MigraineDose = session.Answers.MigraineDose
	report.Libido = session.Answers.Libido
	report.Extras = session.Answers.Extras
	report.Anxiety = session.Answers.Anxiety
	report.Energy = session.Answers.Energy
	report.SleepQuality = session.Answers.SleepQuality
	report.MoodStability = session.Answers.MoodStability
	report.Relationship = session.Answers.Relationship
	report.Closeness = session.Answers.Closeness
	report.DayComment = session.Answers.DayComment

	return c.dailyReportRepo.Create(c.ctx, report)
}

func (c *ChatBot) finishSurvey(chatID int64, userID int64, session *Session) {
	session.Step = 13

	log.Printf("FINAL: %+v", session.Answers)

	if err := c.saveDailyReport(userID, session); err != nil {
		log.Printf("Failed to save daily report: %v", err)
		c.sendMessage(chatID, "Ошибка сохранения. Попробуй ещё раз.")
		return
	}

	c.sendMessage(chatID, "Готово ✅")
	c.sessionStore.Delete(chatID)
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
