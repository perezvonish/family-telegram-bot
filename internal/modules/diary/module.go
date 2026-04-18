package diary

import (
	"log"
	"slices"
	"strconv"
	"strings"

	"perezvonish/health-tracker/internal/bot"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/domain/user"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MedsOptions — список препаратов для мультиселекта и стрик-алертов.
var MedsOptions = []string{
	"Мальтофер",
	"Витамин Д",
	"Метилфолат",
	"Витамин B12",
	"Эвика",
	"Венлаксор - утро",
	"Венлаксор - вечер",
	"Дюфастон - утро",
	"Дюфастон - вечер",
	"Триттико",
}

var (
	mealsOptions       = []string{"Завтрак", "Перекус", "Обед", "Ужин"}
	extrasOptions      = []string{"Кофе", "Матча", "Алкоголь", "Сигареты", "Энергос", "Вкусняшки (с высоким ГИ)"}
	sleepLabels        = []string{"Почти не спала", "Плохой", "Средний", "Хороший", "Отличный"}
	anxietyLabels      = []string{"Нет", "Слабая", "Умеренная", "Сильная", "Парализующая"}
	energyLabels       = []string{"Опустошение", "Низкая", "Средняя", "Хорошая", "Заряжен"}
	libidoLabels       = []string{"Отсутствует", "Слабое", "Среднее", "Повышенное", "Высокое"}
	relationshipLabels = []string{"Очень недовольна", "Скорее недовольна", "Нейтрально", "В целом хорошо", "Очень довольна"}
	closenessLabels    = []string{"Изолированность", "Дистанция", "Нейтрально", "Близко", "Очень близко"}
)

// Session data keys
const (
	keySleepTime     = "sleepTime"
	keyWakeTime      = "wakeTime"
	keyWorkedToday   = "workedToday"
	keyMenstruation  = "menstruation"
	keyFasting       = "fasting"
	keyActivity      = "activity"
	keyMealsSkipped  = "mealsSkipped"
	keyMedsIssues    = "medsIssues"
	keyExtras        = "extras"
	keySleepQuality  = "sleepQuality"
	keyMood          = "mood"
	keyMoodStability = "moodStability"
	keyAnxiety       = "anxiety"
	keyEnergy        = "energy"
	keyLibido        = "libido"
	keyRelationship  = "relationship"
	keyCloseness     = "closeness"
	keyMigraine      = "migraine"
	keyMigraineSide  = "migraineSide"
	keyMigraineDose  = "migraineDose"
	keyDayComment    = "dayComment"
)

// Module — дневник здоровья.
type Module struct {
	dailyReportRepo daily_report.Repository
	userRepo        user.Repository
	// onComplete вызывается после успешного сохранения отчёта (например, запуск pills-алертов).
	onComplete func(ctx bot.BotContext)
}

func New(dailyReportRepo daily_report.Repository, userRepo user.Repository, onComplete func(ctx bot.BotContext)) *Module {
	return &Module{
		dailyReportRepo: dailyReportRepo,
		userRepo:        userRepo,
		onComplete:      onComplete,
	}
}

func (m *Module) Name() string { return "diary" }
func (m *Module) Commands() []string {
	return []string{"diary", "start"}
}
func (m *Module) CallbackPrefixes() []string {
	return []string{
		"m:", "sleep:", "mood:", "stability:",
		"anxiety:", "energy:", "libido:", "rel:", "close:",
		"migraine:", "mside:", "back:",
	}
}

// HandleCommand — /diary или /start: начать новую запись.
func (m *Module) HandleCommand(ctx bot.BotContext, _, _ string) error {
	session := bot.NewSession("diary")
	session.Step = 0
	ctx.Sessions.Set(ctx.ChatID, session)
	ctx.SendWithKeyboard("Во сколько вчера легла?", sleepTimeKeyboard())
	return nil
}

// HandleTextStep — обработка текстовых шагов (0–5, 19, 20).
func (m *Module) HandleTextStep(ctx bot.BotContext, session *bot.Session, text string) error {
	if text == "← Назад" {
		m.goBackText(ctx, session)
		return nil
	}

	switch session.Step {
	case 0:
		session.Set(keySleepTime, text)
		session.Step = 1
		ctx.SendWithKeyboard("Во сколько сегодня проснулась?", wakeTimeKeyboard())

	case 1:
		session.Set(keyWakeTime, text)
		session.Step = 2
		ctx.SendWithKeyboard("Работала сегодня?", yesNoKeyboard())

	case 2:
		session.Set(keyWorkedToday, text)
		session.Step = 3
		ctx.SendWithKeyboard("Сегодня была менструация?", yesNoKeyboard())

	case 3:
		session.Set(keyMenstruation, text)
		session.Step = 4
		ctx.SendWithKeyboard("Было ли голодание в течение дня?", fastingKeyboard())

	case 4:
		session.Set(keyFasting, text)
		session.Step = 5
		ctx.SendWithKeyboard("Была ли физическая активность?", activityKeyboard())

	case 5:
		session.Set(keyActivity, text)
		session.Step = 6
		ctx.SendWithInlineKeyboard("Все приемы пищи были?",
			multiSelectKeyboard(mealsOptions, session.GetStringSlice(keyMealsSkipped)))

	case 19:
		dose := strings.TrimSpace(text)
		if dose == "" {
			ctx.Send("Введи препарат и дозировку. Например: Ибуклин 600")
			return nil
		}
		session.Set(keyMigraineDose, dose)
		session.Step = 20
		ctx.SendWithKeyboard("Есть что-то, что хочется отметить про день?", skipKeyboard())

	case 20:
		if text != "Пропустить" && text != "/skip" {
			session.Set(keyDayComment, text)
		}
		return m.finishSurvey(ctx, session)
	}
	return nil
}

// HandleCallback — обработка inline-кнопок шагов 6–18.
func (m *Module) HandleCallback(ctx bot.BotContext, msgID int, data string) error {
	session := ctx.Sessions.Get(ctx.ChatID)
	if session == nil {
		return nil
	}

	log.Printf("[diary] callback step=%d data=%s", session.Step, data)

	switch {
	case strings.HasPrefix(data, "m:"):
		return m.handleMultiSelect(ctx, session, msgID, data)

	case strings.HasPrefix(data, "sleep:"):
		session.Set(keySleepQuality, parseScaleValue(data))
		session.Step = 10
		m.sendPhoto(ctx, "assets/scales/mood.jpg", "Оцени настроение за день:",
			scaleRangeKeyboard("mood", 1, 10))

	case strings.HasPrefix(data, "mood:"):
		session.Set(keyMood, parseScaleValue(data))
		session.Step = 11
		ctx.SendWithInlineKeyboard("Как менялось настроение в течение дня?", stabilityKeyboard())

	case strings.HasPrefix(data, "stability:"):
		session.Set(keyMoodStability, strings.TrimPrefix(data, "stability:"))
		session.Step = 12
		m.sendPhoto(ctx, "assets/scales/anxiety.jpg", "Оцени тревогу:",
			labeledScaleKeyboard("anxiety", anxietyLabels))

	case strings.HasPrefix(data, "anxiety:"):
		session.Set(keyAnxiety, parseScaleValue(data))
		session.Step = 13
		m.sendPhoto(ctx, "assets/scales/energy.jpg", "Оцени уровень энергии:",
			labeledScaleKeyboard("energy", energyLabels))

	case strings.HasPrefix(data, "energy:"):
		session.Set(keyEnergy, parseScaleValue(data))
		session.Step = 14
		m.sendPhoto(ctx, "assets/scales/libido.jpg", "Оцени уровень либидо:",
			labeledScaleKeyboard("libido", libidoLabels))

	case strings.HasPrefix(data, "libido:"):
		session.Set(keyLibido, parseScaleValue(data))
		session.Step = 15
		m.sendPhoto(ctx, "assets/scales/relationship.jpg", "Оцени удовлетворённость отношениями:",
			labeledScaleKeyboard("rel", relationshipLabels))

	case strings.HasPrefix(data, "rel:"):
		session.Set(keyRelationship, parseScaleValue(data))
		session.Step = 16
		m.sendPhoto(ctx, "assets/scales/closeness.jpg", "Оцени близость с партнёром:",
			labeledScaleKeyboard("close", closenessLabels))

	case strings.HasPrefix(data, "close:"):
		session.Set(keyCloseness, parseScaleValue(data))
		session.Step = 17
		m.sendPhoto(ctx, "assets/scales/migraine.jpg",
			"Оцени мигрень (0 - не было | 5 - вызываем скорую):",
			scaleRangeKeyboard("migraine", 0, 5))

	case strings.HasPrefix(data, "migraine:"):
		migraine := parseScaleValue(data)
		session.Set(keyMigraine, migraine)
		if migraine >= 1 {
			session.Step = 18
			ctx.SendWithInlineKeyboard("Где локализована боль?", migraineSideKeyboard())
		} else {
			session.Step = 20
			ctx.SendWithKeyboard("Есть что-то, что хочется отметить про день?", skipKeyboard())
		}

	case strings.HasPrefix(data, "mside:"):
		session.Set(keyMigraineSide, strings.TrimPrefix(data, "mside:"))
		migraine := session.GetInt(keyMigraine)
		if migraine >= 2 {
			session.Step = 19
			msg := tgbotapi.NewMessage(ctx.ChatID, "Какой препарат принимала? (например: Ибуклин 600)")
			msg.ReplyMarkup = removeKeyboard()
			ctx.API.Send(msg) //nolint:errcheck
		} else {
			session.Step = 20
			ctx.SendWithKeyboard("Есть что-то, что хочется отметить про день?", skipKeyboard())
		}

	case strings.HasPrefix(data, "back:"):
		m.goBack(ctx, session)
	}
	return nil
}

func (m *Module) goBack(ctx bot.BotContext, session *bot.Session) {
	switch session.Step {
	case 6:
		session.Step = 5
		ctx.SendWithKeyboard("Была ли физическая активность?", activityKeyboard())
	case 7:
		session.Step = 6
		ctx.SendWithInlineKeyboard("Все приемы пищи были?",
			multiSelectKeyboard(mealsOptions, session.GetStringSlice(keyMealsSkipped)))
	case 8:
		session.Step = 7
		ctx.SendWithInlineKeyboard("Все таблетки выпила?",
			multiSelectKeyboard(MedsOptions, session.GetStringSlice(keyMedsIssues)))
	case 9:
		session.Step = 8
		ctx.SendWithInlineKeyboard("Что употребляла сегодня?",
			multiSelectKeyboard(extrasOptions, session.GetStringSlice(keyExtras)))
	case 10:
		session.Step = 9
		m.sendPhoto(ctx, "assets/scales/sleep_quality.jpg", "Оцени качество сна:",
			labeledScaleKeyboard("sleep", sleepLabels))
	case 11:
		session.Step = 10
		m.sendPhoto(ctx, "assets/scales/mood.jpg", "Оцени настроение за день:",
			scaleRangeKeyboard("mood", 1, 10))
	case 12:
		session.Step = 11
		ctx.SendWithInlineKeyboard("Как менялось настроение в течение дня?", stabilityKeyboard())
	case 13:
		session.Step = 12
		m.sendPhoto(ctx, "assets/scales/anxiety.jpg", "Оцени тревогу:",
			labeledScaleKeyboard("anxiety", anxietyLabels))
	case 14:
		session.Step = 13
		m.sendPhoto(ctx, "assets/scales/energy.jpg", "Оцени уровень энергии:",
			labeledScaleKeyboard("energy", energyLabels))
	case 15:
		session.Step = 14
		m.sendPhoto(ctx, "assets/scales/libido.jpg", "Оцени уровень либидо:",
			labeledScaleKeyboard("libido", libidoLabels))
	case 16:
		session.Step = 15
		m.sendPhoto(ctx, "assets/scales/relationship.jpg", "Оцени удовлетворённость отношениями:",
			labeledScaleKeyboard("rel", relationshipLabels))
	case 17:
		session.Step = 16
		m.sendPhoto(ctx, "assets/scales/closeness.jpg", "Оцени близость с партнёром:",
			labeledScaleKeyboard("close", closenessLabels))
	case 18:
		session.Step = 17
		m.sendPhoto(ctx, "assets/scales/migraine.jpg",
			"Оцени мигрень (0 - не было | 5 - вызываем скорую):",
			scaleRangeKeyboard("migraine", 0, 5))
	}
}

func (m *Module) goBackText(ctx bot.BotContext, session *bot.Session) {
	switch session.Step {
	case 0:
		ctx.SendWithKeyboard("Во сколько вчера легла?", sleepTimeKeyboard())
	case 1:
		session.Step = 0
		ctx.SendWithKeyboard("Во сколько вчера легла?", sleepTimeKeyboard())
	case 2:
		session.Step = 1
		ctx.SendWithKeyboard("Во сколько сегодня проснулась?", wakeTimeKeyboard())
	case 3:
		session.Step = 2
		ctx.SendWithKeyboard("Работала сегодня?", yesNoKeyboard())
	case 4:
		session.Step = 3
		ctx.SendWithKeyboard("Сегодня была менструация?", yesNoKeyboard())
	case 5:
		session.Step = 4
		ctx.SendWithKeyboard("Было ли голодание в течение дня?", fastingKeyboard())
	case 19:
		session.Step = 18
		ctx.SendWithInlineKeyboard("Где локализована боль?", migraineSideKeyboard())
	case 20:
		migraine := session.GetInt(keyMigraine)
		if migraine >= 2 {
			session.Step = 19
			msg := tgbotapi.NewMessage(ctx.ChatID, "Какой препарат принимала? (например: Ибуклин 600)")
			msg.ReplyMarkup = removeKeyboard()
			ctx.API.Send(msg) //nolint:errcheck
		} else if migraine >= 1 {
			session.Step = 18
			ctx.SendWithInlineKeyboard("Где локализована боль?", migraineSideKeyboard())
		} else {
			session.Step = 17
			m.sendPhoto(ctx, "assets/scales/migraine.jpg",
				"Оцени мигрень (0 - не было | 5 - вызываем скорую):",
				scaleRangeKeyboard("migraine", 0, 5))
		}
	}
}

func (m *Module) handleMultiSelect(ctx bot.BotContext, session *bot.Session, msgID int, data string) error {
	if data == "m:done" {
		ctx.EditMessageRemoveKeyboard(msgID)
		switch session.Step {
		case 6:
			session.Step = 7
			ctx.SendWithInlineKeyboard("Все таблетки выпила?",
				multiSelectKeyboard(MedsOptions, session.GetStringSlice(keyMedsIssues)))
		case 7:
			session.Step = 8
			ctx.SendWithInlineKeyboard("Что употребляла сегодня?",
				multiSelectKeyboard(extrasOptions, session.GetStringSlice(keyExtras)))
		case 8:
			session.Step = 9
			m.sendPhoto(ctx, "assets/scales/sleep_quality.jpg", "Оцени качество сна:",
				labeledScaleKeyboard("sleep", sleepLabels))
		}
		return nil
	}

	value := strings.TrimPrefix(data, "m:")
	var (
		fieldKey string
		options  []string
	)
	switch session.Step {
	case 6:
		fieldKey, options = keyMealsSkipped, mealsOptions
	case 7:
		fieldKey, options = keyMedsIssues, MedsOptions
	case 8:
		fieldKey, options = keyExtras, extrasOptions
	default:
		return nil
	}

	toggleStringSlice(session, fieldKey, value)
	edit := tgbotapi.NewEditMessageReplyMarkup(ctx.ChatID, msgID,
		multiSelectKeyboard(options, session.GetStringSlice(fieldKey)))
	ctx.API.Send(edit) //nolint:errcheck
	return nil
}

func (m *Module) finishSurvey(ctx bot.BotContext, session *bot.Session) error {
	log.Printf("[diary] finishing survey for user=%d", ctx.UserID)

	u, err := ctx.Users.FindByTelegramID(ctx.Ctx, ctx.UserID)
	if err != nil {
		ctx.Send("Ошибка загрузки пользователя.")
		return err
	}

	report := daily_report.NewDailyReport(u.PrimaryStorageID())
	report.SleepTime = session.GetString(keySleepTime)
	report.WakeTime = session.GetString(keyWakeTime)
	report.WorkedToday = session.GetString(keyWorkedToday)
	report.Menstruation = session.GetString(keyMenstruation)
	report.Fasting = session.GetString(keyFasting)
	report.Activity = session.GetString(keyActivity)
	report.MealsSkipped = session.GetStringSlice(keyMealsSkipped)
	report.MedsIssues = session.GetStringSlice(keyMedsIssues)
	report.Extras = session.GetStringSlice(keyExtras)
	report.SleepQuality = session.GetInt(keySleepQuality)
	report.Mood = session.GetInt(keyMood)
	report.MoodStability = session.GetString(keyMoodStability)
	report.Anxiety = session.GetInt(keyAnxiety)
	report.Energy = session.GetInt(keyEnergy)
	report.Libido = session.GetInt(keyLibido)
	report.Relationship = session.GetInt(keyRelationship)
	report.Closeness = session.GetInt(keyCloseness)
	report.Migraine = session.GetInt(keyMigraine)
	report.MigraineSide = session.GetString(keyMigraineSide)
	report.MigraineDose = session.GetString(keyMigraineDose)
	report.DayComment = session.GetString(keyDayComment)

	if err := m.dailyReportRepo.Create(ctx.Ctx, report); err != nil {
		log.Printf("[diary] failed to save report: %v", err)
		ctx.Send("Ошибка сохранения. Попробуй ещё раз.")
		return err
	}

	ctx.Send("Готово ✅")
	ctx.Sessions.Delete(ctx.ChatID)

	if m.onComplete != nil {
		go m.onComplete(ctx)
	}
	return nil
}

func (m *Module) sendPhoto(ctx bot.BotContext, assetPath, caption string, keyboard tgbotapi.InlineKeyboardMarkup) {
	data, err := scaleAssets.ReadFile(assetPath)
	if err != nil {
		log.Printf("[diary] failed to read asset %s: %v", assetPath, err)
		ctx.SendWithInlineKeyboard(caption, keyboard)
		return
	}
	photo := tgbotapi.NewPhoto(ctx.ChatID, tgbotapi.FileBytes{Name: assetPath, Bytes: data})
	photo.Caption = caption
	photo.ReplyMarkup = keyboard
	ctx.API.Send(photo) //nolint:errcheck
}

func toggleStringSlice(session *bot.Session, key, value string) {
	sl := session.GetStringSlice(key)
	idx := slices.Index(sl, value)
	if idx >= 0 {
		sl = slices.Delete(sl, idx, idx+1)
	} else {
		sl = append(sl, value)
	}
	session.Set(key, sl)
}

func parseScaleValue(data string) int {
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return 0
	}
	v, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return v
}
