package telegram_bot

import (
	"context"
	"log"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/domain/pill_tracker"
	"perezvonish/health-tracker/internal/domain/user"
	"slices"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	mealsOptions       = []string{"Завтрак", "Перекус", "Обед", "Ужин"}
	medsOptions        = []string{"Мальтофер", "Витамин Д", "Метилфолат", "Витамин B12", "Эвика"}
	extrasOptions      = []string{"Кофе", "Матча", "Алкоголь", "Сигареты"}
	sleepLabels        = []string{"Почти не спала", "Плохой", "Средний", "Хороший", "Отличный"}
	anxietyLabels      = []string{"Нет", "Слабая", "Умеренная", "Сильная", "Парализующая"}
	energyLabels       = []string{"Опустошение", "Низкая", "Средняя", "Хорошая", "Заряжен"}
	libidoLabels       = []string{"Отсутствует", "Слабое", "Среднее", "Повышенное", "Высокое"}
	relationshipLabels = []string{"Очень недовольна", "Скорее недовольна", "Нейтрально", "В целом хорошо", "Очень довольна"}
	closenessLabels    = []string{"Изолированность", "Дистанция", "Нейтрально", "Близко", "Очень близко"}
)

type ChatBot struct {
	ctx context.Context

	telegramBotApi  *tgbotapi.BotAPI
	sessionStore    *SessionStore
	userRepo        user.Repository
	dailyReportRepo daily_report.Repository
	pillRepo        pill_tracker.Repository
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
	// Удаляем webhook если он был установлен — long polling и webhook несовместимы
	c.telegramBotApi.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false})

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"} // явно, не null
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

	if text == "/pills" {
		c.handlePillsCommand(chatID, userID)
		return
	}

	if text == "/help" {
		c.handleHelpCommand(chatID)
		return
	}

	switch {
	case text == "/today":
		c.handleTodayCommand(chatID, userID)
		return
	case text == "/week":
		c.handleWeekCommand(chatID, userID)
		return
	case text == "/migraine":
		c.handleMigraineCommand(chatID, userID)
		return
	case strings.HasPrefix(text, "/stats"):
		days := 30
		parts := strings.Fields(text)
		if len(parts) > 1 {
			days, _ = strconv.Atoi(parts[1])
		}
		c.handleStatsCommand(chatID, userID, days)
		return
	}

	session := c.sessionStore.Get(chatID)
	if session == nil || session.Step < 0 {
		msg := tgbotapi.NewMessage(chatID, "Используй /diary чтобы начать дневник здоровья")
		c.telegramBotApi.Send(msg)
		return
	}

	if session.Scene == ScenePills {
		c.handlePillsTextStep(chatID, userID, session, text)
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

	case 19:
		dose := strings.TrimSpace(text)
		if dose == "" {
			c.sendMessage(chatID, "Введи препарат и дозировку. Например: Ибуклин 600")
			return
		}
		session.Answers.MigraineDose = dose
		session.Step = 20
		c.sendWithKeyboard(chatID, "Есть что-то, что хочется отметить про день?", skipKeyboard())

	case 20:
		if text != "Пропустить" && text != "/skip" {
			session.Answers.DayComment = text
		}
		c.finishSurvey(chatID, userID, session)
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

	if strings.HasPrefix(data, "pills:") {
		c.handlePillsCallback(chatID, userID, messageID, data)
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
			c.sendWithInlineKeyboard(chatID, "Что употребляла сегодня?", multiSelectKeyboard(extrasOptions, session.Answers.Extras))
		})

	case 8:
		c.handleMultiSelect(chatID, messageID, session, "extras", extrasOptions, data, 9, func() {
			c.sendPhotoWithInlineKeyboard(chatID, "assets/scales/sleep_quality.jpg", "Оцени качество сна:", labeledScaleKeyboard("sleep", sleepLabels))
		})

	case 9:
		session.Answers.SleepQuality = c.parseScaleValue(data)
		session.Step = 10
		c.sendPhotoWithInlineKeyboard(chatID, "assets/scales/mood.jpg", "Оцени настроение за день:", scaleRangeKeyboard("mood", 1, 10, true))

	case 10:
		session.Answers.Mood = c.parseScaleValue(data)
		session.Step = 11
		c.sendWithInlineKeyboard(chatID, "Как менялось настроение в течение дня?", stabilityKeyboard())

	case 11:
		session.Answers.MoodStability = strings.TrimPrefix(data, "stability:")
		session.Step = 12
		c.sendPhotoWithInlineKeyboard(chatID, "assets/scales/anxiety.jpg", "Оцени тревогу:", labeledScaleKeyboard("anxiety", anxietyLabels))

	case 12:
		session.Answers.Anxiety = c.parseScaleValue(data)
		session.Step = 13
		c.sendPhotoWithInlineKeyboard(chatID, "assets/scales/energy.jpg", "Оцени уровень энергии:", labeledScaleKeyboard("energy", energyLabels))

	case 13:
		session.Answers.Energy = c.parseScaleValue(data)
		session.Step = 14
		c.sendPhotoWithInlineKeyboard(chatID, "assets/scales/libido.jpg", "Оцени уровень либидо:", labeledScaleKeyboard("libido", libidoLabels))

	case 14:
		session.Answers.Libido = c.parseScaleValue(data)
		session.Step = 15
		c.sendPhotoWithInlineKeyboard(chatID, "assets/scales/relationship.jpg", "Оцени удовлетворённость отношениями:", labeledScaleKeyboard("rel", relationshipLabels))

	case 15:
		session.Answers.Relationship = c.parseScaleValue(data)
		session.Step = 16
		c.sendPhotoWithInlineKeyboard(chatID, "assets/scales/closeness.jpg", "Оцени близость с партнёром:", labeledScaleKeyboard("close", closenessLabels))

	case 16:
		session.Answers.Closeness = c.parseScaleValue(data)
		session.Step = 17
		c.sendPhotoWithInlineKeyboard(chatID, "assets/scales/migraine.jpg", "Оцени мигрень (0 - не было | 5 - вызываем скорую):", scaleRangeKeyboard("migraine", 0, 5, false))

	case 17:
		session.Answers.Migraine = c.parseScaleValue(data)
		if session.Answers.Migraine >= 1 {
			session.Step = 18
			c.sendWithInlineKeyboard(chatID, "Где локализована боль?", migraineSideKeyboard())
			return
		}
		session.Step = 20
		c.sendWithKeyboard(chatID, "Есть что-то, что хочется отметить про день?", skipKeyboard())

	case 18:
		session.Answers.MigraineSide = strings.TrimPrefix(data, "mside:")
		if session.Answers.Migraine >= 2 {
			session.Step = 19
			msg := tgbotapi.NewMessage(chatID, "Какой препарат принимала? (например: Ибуклин 600)")
			msg.ReplyMarkup = removeKeyboard()
			c.telegramBotApi.Send(msg)
			return
		}
		session.Step = 20
		c.sendWithKeyboard(chatID, "Есть что-то, что хочется отметить про день?", skipKeyboard())
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
	} else if field == "extras" {
		selected = session.Answers.Extras
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
	} else if field == "extras" {
		arr = &session.Answers.Extras
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

func (c *ChatBot) sendRemovingKeyboard(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = removeKeyboard()
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
	session.Step = 21

	log.Printf("FINAL: %+v", session.Answers)

	if err := c.saveDailyReport(userID, session); err != nil {
		log.Printf("Failed to save daily report: %v", err)
		c.sendMessage(chatID, "Ошибка сохранения. Попробуй ещё раз.")
		return
	}

	c.sendMessage(chatID, "Готово ✅")
	c.sessionStore.Delete(chatID)

	go c.checkPillsForUser(chatID, userID)
}

func NewChatBot(ctx context.Context, bot *tgbotapi.BotAPI, userRepo user.Repository, dailyReportRepo daily_report.Repository, pillRepo pill_tracker.Repository) Bot {
	return &ChatBot{
		ctx:             ctx,
		telegramBotApi:  bot,
		sessionStore:    NewSessionStore(),
		userRepo:        userRepo,
		dailyReportRepo: dailyReportRepo,
		pillRepo:        pillRepo,
	}
}
