package telegram_bot

import (
	"fmt"
	"log"
	"time"

	"perezvonish/health-tracker/internal/domain/analytics"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/domain/user"
	"perezvonish/health-tracker/internal/modules/diary"
)

func (c *ChatBot) startAlertWorker() {
	for {
		now := time.Now()
		next := nextAlertTime(now, 9, 0, "Europe/Moscow")
		timer := time.NewTimer(next.Sub(now))
		select {
		case <-timer.C:
			c.runDailyAlerts()
		case <-c.ctx.Done():
			timer.Stop()
			return
		}
	}
}

func nextAlertTime(now time.Time, hour, minute int, timezone string) time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	nowInLoc := now.In(loc)
	next := time.Date(nowInLoc.Year(), nowInLoc.Month(), nowInLoc.Day(), hour, minute, 0, 0, loc)
	if !next.After(nowInLoc) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (c *ChatBot) runDailyAlerts() {
	users, err := c.userRepo.FindAll(c.ctx)
	if err != nil {
		log.Printf("alerts: failed to get users: %v", err)
		return
	}
	for _, u := range users {
		// FindLatest возвращает записи в порядке убывания даты (новые первые)
		last7, err := c.dailyReportRepo.FindLatest(c.ctx, u.ID, 7)
		if err != nil || len(last7) < 2 {
			continue
		}
		c.checkAnxietyAlert(u, last7)
		c.checkWellnessDropAlert(u, last7)
		c.checkMigraineFollowup(u, last7)
		c.checkMedStreakCelebration(u, last7)
		c.checkWeeklySundayInsight(u)
		c.pillsModule.RunAlerts(c.makeBotContext(u.TelegramID, u.TelegramID))
	}
}

// checkAnxietyAlert — тревога ≥ 4 три дня подряд
func (c *ChatBot) checkAnxietyAlert(u *user.User, reports []*daily_report.DailyReport) {
	if len(reports) < 3 {
		return
	}
	for i := 0; i < 3; i++ {
		if reports[i].Anxiety < 4 {
			return
		}
	}
	c.sendMessage(u.TelegramID, "Заметила, что тревога высокая уже 3 дня подряд 🫂")
}

// checkWellnessDropAlert — ИОС < 5 два дня подряд
func (c *ChatBot) checkWellnessDropAlert(u *user.User, reports []*daily_report.DailyReport) {
	if len(reports) < 2 {
		return
	}
	if analytics.WellnessIndex(reports[0]) < 5 && analytics.WellnessIndex(reports[1]) < 5 {
		c.sendMessage(u.TelegramID, "Последние два дня даются тяжело. Как ты сейчас?")
	}
}

// checkMigraineFollowup — вчера мигрень ≥ 3
func (c *ChatBot) checkMigraineFollowup(u *user.User, reports []*daily_report.DailyReport) {
	if len(reports) < 1 {
		return
	}
	if reports[0].Migraine >= 3 {
		c.sendMessage(u.TelegramID, "Вчера была сильная мигрень. Сегодня как голова?")
	}
}

// checkMedStreakCelebration — все таблетки 7 дней подряд
func (c *ChatBot) checkMedStreakCelebration(u *user.User, reports []*daily_report.DailyReport) {
	if len(reports) < 7 {
		return
	}
	for i := 0; i < 7; i++ {
		if len(reports[i].MedsIssues) != len(diary.MedsOptions) {
			return
		}
	}
	c.sendMessage(u.TelegramID, "Неделя без пропусков! Так держать 💊🎉")
}

// checkWeeklySundayInsight — каждое воскресенье: сильнейшая корреляция за 30 дней
func (c *ChatBot) checkWeeklySundayInsight(u *user.User) {
	if time.Now().Weekday() != time.Sunday {
		return
	}
	last30, err := c.dailyReportRepo.FindLatest(c.ctx, u.ID, 30)
	if err != nil || len(last30) < 7 {
		return
	}
	corrs := analytics.TopCorrelations(last30)
	if len(corrs) == 0 {
		return
	}
	top := corrs[0]
	arrow := "📈"
	if top.R < 0 {
		arrow = "📉"
	}
	text := fmt.Sprintf("📊 Инсайт недели:\n%s %s → %s  r=%+.2f (%s)", arrow, top.LabelA, top.LabelB, top.R, top.Strength)
	c.sendMessage(u.TelegramID, text)
}
