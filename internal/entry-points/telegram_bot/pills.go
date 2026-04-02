package telegram_bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"perezvonish/health-tracker/internal/domain/pill_tracker"

	"github.com/google/uuid"
)

func (c *ChatBot) handlePillsCommand(chatID int64, telegramUserID int64) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "Ошибка загрузки пользователя.")
		return
	}

	trackers, err := c.pillRepo.FindByUser(c.ctx, u.ID)
	if err != nil {
		c.sendMessage(chatID, "Ошибка загрузки трекеров.")
		return
	}

	if len(trackers) == 0 {
		c.sendWithInlineKeyboard(chatID,
			"💊 Трекеров таблеток пока нет.\nДобавь первый препарат:",
			pillsListKeyboard(trackers))
		return
	}

	text := "💊 Твои препараты:\n\n"
	for _, t := range trackers {
		text += fmt.Sprintf("• %s — осталось %.0f таб. (~%.0f дней)\n",
			t.Name, t.Remaining(), t.DaysLeft())
	}
	c.sendWithInlineKeyboard(chatID, text, pillsListKeyboard(trackers))
}

func (c *ChatBot) handlePillsCallback(chatID int64, telegramUserID int64, messageID int, data string) {
	parts := strings.SplitN(data, ":", 3)
	action := parts[1]

	switch action {
	case "add":
		session := c.sessionStore.ResetPills(chatID)
		session.Step = 0
		c.sendRemovingKeyboard(chatID, "Как называется препарат?")

	case "edit":
		trackerID := parts[2]
		session := c.sessionStore.ResetPills(chatID)
		session.PillsSetup.EditingID = trackerID
		session.Step = 1
		c.sendWithKeyboard(chatID, "Сколько таблеток купила?", pillsCountKeyboard())

	case "restock":
		trackerID := parts[2]
		session := c.sessionStore.ResetPills(chatID)
		session.PillsSetup.EditingID = trackerID
		session.Step = 1
		c.sendWithKeyboard(chatID, "Сколько таблеток купила?", pillsCountKeyboard())

	case "snooze":
		trackerID := parts[2]
		t, err := c.pillRepo.FindByID(c.ctx, uuid.MustParse(trackerID))
		if err == nil {
			t.NotifiedEmpty = false
			t.UpdatedAt = time.Now().UTC()
			c.pillRepo.Update(c.ctx, t)
		}
		c.editMessageRemoveKeyboard(chatID, messageID)
		c.sendMessage(chatID, "Хорошо, напомню через 2 дня 🔔")
	}
}

func (c *ChatBot) handlePillsTextStep(chatID int64, telegramUserID int64, session *Session, text string) {
	switch session.Step {
	case 0:
		session.PillsSetup.Name = strings.TrimSpace(text)
		session.Step = 1
		c.sendWithKeyboard(chatID, "Сколько таблеток купила?", pillsCountKeyboard())

	case 1:
		n, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || n <= 0 {
			c.sendMessage(chatID, "Введи число, например: 30")
			return
		}
		session.PillsSetup.Total = n
		session.Step = 2

		currentDose := ""
		if session.PillsSetup.EditingID != "" {
			t, err := c.pillRepo.FindByID(c.ctx, uuid.MustParse(session.PillsSetup.EditingID))
			if err == nil && t != nil {
				currentDose = strconv.FormatFloat(t.DailyDose, 'f', -1, 64)
			}
		}
		c.sendWithKeyboard(chatID, "Сколько таблеток в день?", pillsDoseKeyboard(currentDose))

	case 2:
		input := strings.TrimSpace(text)
		if strings.HasPrefix(input, "Оставить ") {
			input = strings.TrimSuffix(strings.TrimPrefix(input, "Оставить "), " в день")
		}
		dose, err := strconv.ParseFloat(strings.ReplaceAll(input, ",", "."), 64)
		if err != nil || dose <= 0 {
			c.sendMessage(chatID, "Введи число, например: 1 или 0.5")
			return
		}
		session.PillsSetup.DailyDose = dose
		c.finishPillsSetup(chatID, telegramUserID, session)
	}
}

func (c *ChatBot) finishPillsSetup(chatID int64, telegramUserID int64, session *Session) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "Ошибка загрузки пользователя.")
		return
	}

	setup := session.PillsSetup
	var tracker *pill_tracker.PillTracker

	if setup.EditingID != "" {
		tracker, err = c.pillRepo.FindByID(c.ctx, uuid.MustParse(setup.EditingID))
		if err != nil {
			c.sendMessage(chatID, "Ошибка загрузки трекера.")
			return
		}
		now := time.Now().UTC()
		tracker.Total = setup.Total
		tracker.DailyDose = setup.DailyDose
		tracker.StartDate = now
		tracker.UpdatedAt = now
		tracker.Notified7d = false
		tracker.Notified3d = false
		tracker.Notified1d = false
		tracker.NotifiedEmpty = false
		c.pillRepo.Update(c.ctx, tracker)
	} else {
		tracker = pill_tracker.NewPillTracker(u.ID, setup.Name, setup.Total, setup.DailyDose)
		c.pillRepo.Create(c.ctx, tracker)
	}

	daysLeft := tracker.DaysLeft()
	emptyDate := tracker.EmptyDate().Format("2 January")

	text := fmt.Sprintf("✅ Сохранила!\n\n💊 %s: %d таб., по %.4g в день\nХватит примерно на %.0f дней\nЗакончатся ~%s 🗓",
		tracker.Name, tracker.Total, tracker.DailyDose, daysLeft, emptyDate)

	c.sendRemovingKeyboard(chatID, text)
	c.sessionStore.Delete(chatID)
}

func (c *ChatBot) checkPillsForUser(chatID int64, telegramUserID int64) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		return
	}

	trackers, err := c.pillRepo.FindByUser(c.ctx, u.ID)
	if err != nil {
		return
	}

	for _, t := range trackers {
		daysLeft := t.DaysLeft()
		changed := false

		switch {
		case t.IsEmpty() && !t.NotifiedEmpty:
			c.sendPillEmptyAlert(chatID, t)
			t.NotifiedEmpty = true
			changed = true

		case daysLeft <= 1 && !t.Notified1d:
			c.sendPillLowAlert(chatID, t, 1)
			t.Notified1d = true
			changed = true

		case daysLeft <= 3 && !t.Notified3d:
			c.sendPillLowAlert(chatID, t, 3)
			t.Notified3d = true
			changed = true

		case daysLeft <= 7 && !t.Notified7d:
			c.sendPillLowAlert(chatID, t, 7)
			t.Notified7d = true
			changed = true
		}

		if changed {
			c.pillRepo.Update(c.ctx, t)
		}
	}
}

func (c *ChatBot) sendPillLowAlert(chatID int64, t *pill_tracker.PillTracker, days int) {
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
		emoji, t.Name, urgency,
		t.Remaining(),
		t.EmptyDate().Format("2 January"))

	if days == 7 {
		text += "\nСамое время заказать 🛒"
	}

	c.sendMessage(chatID, text)
}

func (c *ChatBot) sendPillEmptyAlert(chatID int64, t *pill_tracker.PillTracker) {
	text := fmt.Sprintf("💊 %s закончились.\nКупила новую упаковку?", t.Name)
	c.sendWithInlineKeyboard(chatID, text, pillsEmptyKeyboard(t.ID.String()))
}
