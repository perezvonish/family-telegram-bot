package pills

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"perezvonish/health-tracker/internal/bot"
	"perezvonish/health-tracker/internal/domain/pill_tracker"
	"perezvonish/health-tracker/internal/domain/user"

	"github.com/google/uuid"
)

const (
	keyEditingID = "editingID"
	keyName      = "name"
	keyTotal     = "total"
	keyDailyDose = "dailyDose"
)

// Module — трекер таблеток.
type Module struct {
	pillRepo pill_tracker.Repository
	userRepo user.Repository
}

func New(pillRepo pill_tracker.Repository, userRepo user.Repository) *Module {
	return &Module{pillRepo: pillRepo, userRepo: userRepo}
}

func (m *Module) Name() string               { return "pills" }
func (m *Module) Commands() []string         { return []string{"pills"} }
func (m *Module) CallbackPrefixes() []string { return []string{"pills:"} }

// HandleCommand — /pills: показать список трекеров.
func (m *Module) HandleCommand(ctx bot.BotContext, _, _ string) error {
	u, err := ctx.Users.FindByTelegramID(ctx.Ctx, ctx.UserID)
	if err != nil {
		ctx.Send("Ошибка загрузки пользователя.")
		return err
	}

	trackers, err := m.pillRepo.FindByUser(ctx.Ctx, u.ID)
	if err != nil {
		ctx.Send("Ошибка загрузки трекеров.")
		return err
	}

	if len(trackers) == 0 {
		ctx.SendWithInlineKeyboard("💊 Трекеров таблеток пока нет.\nДобавь первый препарат:", listKeyboard(trackers))
		return nil
	}

	text := "💊 Твои препараты:\n\n"
	for _, t := range trackers {
		text += fmt.Sprintf("• %s — осталось %.0f таб. (~%.0f дней)\n",
			t.Name, t.Remaining(), t.DaysLeft())
	}
	ctx.SendWithInlineKeyboard(text, listKeyboard(trackers))
	return nil
}

// HandleCallback — обработка inline-кнопок pills:*.
func (m *Module) HandleCallback(ctx bot.BotContext, msgID int, data string) error {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 2 {
		return nil
	}
	action := parts[1]

	switch action {
	case "add":
		session := bot.NewSession("pills")
		session.Step = 0
		ctx.Sessions.Set(ctx.ChatID, session)
		ctx.SendRemovingKeyboard("Как называется препарат?")

	case "edit", "restock":
		if len(parts) < 3 {
			return nil
		}
		trackerID := parts[2]
		session := bot.NewSession("pills")
		session.Set(keyEditingID, trackerID)
		session.Step = 1
		ctx.Sessions.Set(ctx.ChatID, session)
		ctx.SendWithKeyboard("Сколько таблеток купила?", countKeyboard())

	case "snooze":
		if len(parts) < 3 {
			return nil
		}
		trackerID := parts[2]
		t, err := m.pillRepo.FindByID(ctx.Ctx, uuid.MustParse(trackerID))
		if err == nil {
			t.NotifiedEmpty = false
			t.UpdatedAt = time.Now().UTC()
			m.pillRepo.Update(ctx.Ctx, t) //nolint:errcheck
		}
		ctx.EditMessageRemoveKeyboard(msgID)
		ctx.Send("Хорошо, напомню через 2 дня 🔔")
	}
	return nil
}

// HandleTextStep — шаги диалога добавления/редактирования таблетки.
func (m *Module) HandleTextStep(ctx bot.BotContext, session *bot.Session, text string) error {
	switch session.Step {
	case 0:
		session.Set(keyName, strings.TrimSpace(text))
		session.Step = 1
		ctx.SendWithKeyboard("Сколько таблеток купила?", countKeyboard())

	case 1:
		n, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || n <= 0 {
			ctx.Send("Введи число, например: 30")
			return nil
		}
		session.Set(keyTotal, n)
		session.Step = 2

		currentDose := ""
		if editingID := session.GetString(keyEditingID); editingID != "" {
			t, err := m.pillRepo.FindByID(ctx.Ctx, uuid.MustParse(editingID))
			if err == nil && t != nil {
				currentDose = strconv.FormatFloat(t.DailyDose, 'f', -1, 64)
			}
		}
		ctx.SendWithKeyboard("Сколько таблеток в день?", doseKeyboard(currentDose))

	case 2:
		input := strings.TrimSpace(text)
		if strings.HasPrefix(input, "Оставить ") {
			input = strings.TrimSuffix(strings.TrimPrefix(input, "Оставить "), " в день")
		}
		dose, err := strconv.ParseFloat(strings.ReplaceAll(input, ",", "."), 64)
		if err != nil || dose <= 0 {
			ctx.Send("Введи число, например: 1 или 0.5")
			return nil
		}
		session.Set(keyDailyDose, dose)
		return m.finishSetup(ctx, session)
	}
	return nil
}

func (m *Module) finishSetup(ctx bot.BotContext, session *bot.Session) error {
	u, err := ctx.Users.FindByTelegramID(ctx.Ctx, ctx.UserID)
	if err != nil {
		ctx.Send("Ошибка загрузки пользователя.")
		return err
	}

	editingID := session.GetString(keyEditingID)
	name := session.GetString(keyName)
	total := session.GetInt(keyTotal)
	dailyDose, _ := session.Get(keyDailyDose)
	dose, _ := dailyDose.(float64)

	var tracker *pill_tracker.PillTracker

	if editingID != "" {
		tracker, err = m.pillRepo.FindByID(ctx.Ctx, uuid.MustParse(editingID))
		if err != nil {
			ctx.Send("Ошибка загрузки трекера.")
			return err
		}
		now := time.Now().UTC()
		tracker.Total = total
		tracker.DailyDose = dose
		tracker.StartDate = now
		tracker.UpdatedAt = now
		tracker.Notified7d = false
		tracker.Notified3d = false
		tracker.Notified1d = false
		tracker.NotifiedEmpty = false
		m.pillRepo.Update(ctx.Ctx, tracker) //nolint:errcheck
	} else {
		tracker = pill_tracker.NewPillTracker(u.ID, name, total, dose)
		m.pillRepo.Create(ctx.Ctx, tracker) //nolint:errcheck
	}

	daysLeft := tracker.DaysLeft()
	emptyDate := tracker.EmptyDate().Format("2 January")

	text := fmt.Sprintf("✅ Сохранила!\n\n💊 %s: %d таб., по %.4g в день\nХватит примерно на %.0f дней\nЗакончатся ~%s 🗓",
		tracker.Name, tracker.Total, tracker.DailyDose, daysLeft, emptyDate)

	ctx.SendRemovingKeyboard(text)
	ctx.Sessions.Delete(ctx.ChatID)
	return nil
}

// RunAlerts проверяет таблетки пользователя и отправляет уведомления при необходимости.
// Вызывается из alert worker'а после завершения дневника.
func (m *Module) RunAlerts(ctx bot.BotContext) {
	u, err := ctx.Users.FindByTelegramID(ctx.Ctx, ctx.UserID)
	if err != nil {
		return
	}

	trackers, err := m.pillRepo.FindByUser(ctx.Ctx, u.ID)
	if err != nil {
		return
	}

	for _, t := range trackers {
		daysLeft := t.DaysLeft()
		changed := false

		switch {
		case t.IsEmpty() && !t.NotifiedEmpty:
			m.sendEmptyAlert(ctx, t)
			t.NotifiedEmpty = true
			changed = true

		case daysLeft <= 1 && !t.Notified1d:
			m.sendLowAlert(ctx, t, 1)
			t.Notified1d = true
			changed = true

		case daysLeft <= 3 && !t.Notified3d:
			m.sendLowAlert(ctx, t, 3)
			t.Notified3d = true
			changed = true

		case daysLeft <= 7 && !t.Notified7d:
			m.sendLowAlert(ctx, t, 7)
			t.Notified7d = true
			changed = true
		}

		if changed {
			m.pillRepo.Update(ctx.Ctx, t) //nolint:errcheck
		}
	}
}

func (m *Module) sendLowAlert(ctx bot.BotContext, t *pill_tracker.PillTracker, days int) {
	var emoji, urgency string
	switch days {
	case 7:
		emoji = "💊"
		urgency = "заканчивается через ~7 дней"
	case 3:
		emoji = "⚠️"
		urgency = "осталось ~3 дня!"
	case 1:
		emoji = "🔴"
		urgency = "заканчивается завтра!"
	}

	text := fmt.Sprintf("%s %s — %s\nОсталось ~%.0f таб.\nЗакончатся %s",
		emoji, t.Name, urgency, t.Remaining(), t.EmptyDate().Format("2 January"))

	if days == 7 {
		text += "\nСамое время заказать 🛒"
	}
	ctx.Send(text)
}

func (m *Module) sendEmptyAlert(ctx bot.BotContext, t *pill_tracker.PillTracker) {
	ctx.SendWithInlineKeyboard(
		fmt.Sprintf("💊 %s закончились.\nКупила новую упаковку?", t.Name),
		emptyKeyboard(t.ID.String()),
	)
}
