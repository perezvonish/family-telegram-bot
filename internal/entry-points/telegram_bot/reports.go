package telegram_bot

import (
	"fmt"
	"strings"
	"time"

	"perezvonish/health-tracker/internal/domain/analytics"
	"perezvonish/health-tracker/internal/domain/daily_report"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// â”€â”€â”€ /help â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (c *ChatBot) handleHelpCommand(chatID int64, _ string) {
	text := `ðŸ“‹ Ð”Ð¾ÑÑ‚ÑƒÐ¿Ð½Ñ‹Ðµ ÐºÐ¾Ð¼Ð°Ð½Ð´Ñ‹:

/diary â€” Ð·Ð°Ð¿Ð¾Ð»Ð½Ð¸Ñ‚ÑŒ Ð´Ð½ÐµÐ²Ð½Ð¸Ðº Ð·Ð´Ð¾Ñ€Ð¾Ð²ÑŒÑ
/today â€” Ð¸Ñ‚Ð¾Ð³ ÑÐµÐ³Ð¾Ð´Ð½ÑÑˆÐ½ÐµÐ³Ð¾ Ð´Ð½Ñ Ñ Ð˜ÐžÐ¡ Ð¸ ÑÑ€Ð°Ð²Ð½ÐµÐ½Ð¸ÐµÐ¼ Ñ Ð½Ð¾Ñ€Ð¼Ð¾Ð¹
/week â€” ÑÐ¿Ð°Ñ€ÐºÐ»Ð°Ð¹Ð½Ñ‹ Ð¸ Ñ€ÐµÐ¶Ð¸Ð¼Ð½Ñ‹Ð¹ score Ð·Ð° Ð¿Ð¾ÑÐ»ÐµÐ´Ð½Ð¸Ðµ 7 Ð´Ð½ÐµÐ¹
/stats â€” ÑÑ€ÐµÐ´Ð½Ð¸Ðµ Ð¿Ð¾ÐºÐ°Ð·Ð°Ñ‚ÐµÐ»Ð¸, ÐºÐ¾Ñ€Ñ€ÐµÐ»ÑÑ†Ð¸Ð¸, Ð»ÑƒÑ‡ÑˆÐ¸Ðµ Ð¸ Ñ‚ÑÐ¶Ñ‘Ð»Ñ‹Ðµ Ð´Ð½Ð¸ Ð·Ð° 30 Ð´Ð½ÐµÐ¹
/stats 90 â€” Ñ‚Ð¾ Ð¶Ðµ Ð·Ð° 90 Ð´Ð½ÐµÐ¹
/migraine â€” ÑÑ‚Ð°Ñ‚Ð¸ÑÑ‚Ð¸ÐºÐ° Ð¼Ð¸Ð³Ñ€ÐµÐ½ÐµÐ¹ Ð¸ Ñ‚Ð¾Ð¿ Ñ‚Ñ€Ð¸Ð³Ð³ÐµÑ€Ð¾Ð² Ð·Ð° 60 Ð´Ð½ÐµÐ¹
/pills â€” Ñ‚Ñ€ÐµÐºÐµÑ€ Ñ‚Ð°Ð±Ð»ÐµÑ‚Ð¾Ðº: Ð¾ÑÑ‚Ð°Ñ‚Ð¾Ðº, Ð¿Ð¾Ð¿Ð¾Ð»Ð½ÐµÐ½Ð¸Ðµ, ÑƒÐ²ÐµÐ´Ð¾Ð¼Ð»ÐµÐ½Ð¸Ñ Ð¾Ð± Ð¾ÐºÐ¾Ð½Ñ‡Ð°Ð½Ð¸Ð¸
/help â€” ÑÑ‚Ð¾ ÑÐ¾Ð¾Ð±Ñ‰ÐµÐ½Ð¸Ðµ`
	msg := tgbotapi.NewMessage(chatID, text)

	c.telegramBotApi.Send(msg) //nolint:errcheck
}

// â”€â”€â”€ /today â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (c *ChatBot) handleTodayCommand(chatID int64, telegramUserID int64) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "ÐÐµ Ð¼Ð¾Ð³Ñƒ Ð½Ð°Ð¹Ñ‚Ð¸ Ð¿Ð¾Ð»ÑŒÐ·Ð¾Ð²Ð°Ñ‚ÐµÐ»Ñ")
		return
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	report, err := c.dailyReportRepo.FindByDate(c.ctx, u.ID, today)
	if err != nil {
		c.sendMessage(chatID, "Ð¡ÐµÐ³Ð¾Ð´Ð½Ñ Ð´Ð½ÐµÐ²Ð½Ð¸Ðº ÐµÑ‰Ñ‘ Ð½Ðµ Ð·Ð°Ð¿Ð¾Ð»Ð½ÐµÐ½. Ð˜ÑÐ¿Ð¾Ð»ÑŒÐ·ÑƒÐ¹ /diary")
		return
	}

	past, _ := c.dailyReportRepo.FindLatest(c.ctx, u.ID, 30)
	norm := analytics.PersonalNorm(past)
	ios := analytics.WellnessIndex(report)

	text := fmt.Sprintf(`ðŸ“Š Ð˜Ñ‚Ð¾Ð³ Ð´Ð½Ñ â€” %s

Ð˜ÐžÐ¡: %.1f %s

ðŸ˜Œ ÐÐ°ÑÑ‚Ñ€Ð¾ÐµÐ½Ð¸Ðµ:   %d/10  %s
ðŸ˜° Ð¢Ñ€ÐµÐ²Ð¾Ð³Ð°:      %d/5   %s
âš¡ Ð­Ð½ÐµÑ€Ð³Ð¸Ñ:      %d/5   %s
ðŸ˜´ Ð¡Ð¾Ð½:          %d/5   %s
ðŸŒ¹ Ð›Ð¸Ð±Ð¸Ð´Ð¾:       %d/5   %s

ðŸ”„ Ð¡Ñ‚Ð°Ð±Ð¸Ð»ÑŒÐ½Ð¾ÑÑ‚ÑŒ: %s

%s

ðŸ¤ ÐžÑ‚Ð½Ð¾ÑˆÐµÐ½Ð¸Ñ: %d/5 Â· Ð‘Ð»Ð¸Ð·Ð¾ÑÑ‚ÑŒ: %d/5
ðŸ§  ÐœÐ¸Ð³Ñ€ÐµÐ½ÑŒ: %s`,
		report.ReportDate.Format("2 January"),
		ios, analytics.WellnessEmoji(ios),
		report.Mood, analytics.DeltaLabel(float64(report.Mood), norm.Mood),
		report.Anxiety, analytics.DeltaLabel(-float64(report.Anxiety), -norm.Anxiety),
		report.Energy, analytics.DeltaLabel(float64(report.Energy), norm.Energy),
		report.SleepQuality, analytics.DeltaLabel(float64(report.SleepQuality), norm.SleepQuality),
		report.Libido, analytics.DeltaLabel(float64(report.Libido), norm.Libido),
		report.MoodStability,
		formatExtras(report.Extras),
		report.Relationship, report.Closeness,
		formatMigraine(report),
	)

	if report.DayComment != "" {
		text += fmt.Sprintf("\n\nðŸ’¬ \"%s\"", report.DayComment)
	}

	c.sendMessage(chatID, text)
}

// â”€â”€â”€ /week â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (c *ChatBot) handleWeekCommand(chatID int64, telegramUserID int64) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "ÐÐµ Ð¼Ð¾Ð³Ñƒ Ð½Ð°Ð¹Ñ‚Ð¸ Ð¿Ð¾Ð»ÑŒÐ·Ð¾Ð²Ð°Ñ‚ÐµÐ»Ñ")
		return
	}

	fmt.Println(u)

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -6)

	reports, _ := c.dailyReportRepo.FindByPeriod(c.ctx, u.ID, from, to)
	if len(reports) == 0 {
		c.sendMessage(chatID, "Ð”Ð°Ð½Ð½Ñ‹Ñ… Ð·Ð° Ð½ÐµÐ´ÐµÐ»ÑŽ Ð½ÐµÑ‚")
		return
	}

	moods, anxieties, energies, sleeps := extractWeekSeries(reports)
	regime := analytics.RegimeScore(reports)

	// Ð›ÑƒÑ‡ÑˆÐ¸Ð¹ Ð¸ Ñ…ÑƒÐ´ÑˆÐ¸Ð¹ Ð´ÐµÐ½ÑŒ Ð¿Ð¾ Ð˜ÐžÐ¡
	bestDay, worstDay := bestAndWorst(reports)

	text := fmt.Sprintf(`ðŸ“ˆ ÐÐµÐ´ÐµÐ»Ñ %s â€” %s (%d Ð´Ð½ÐµÐ¹)

ÐÐ°ÑÑ‚Ñ€Ð¾ÐµÐ½Ð¸Ðµ  %s  avg %.1f
Ð¢Ñ€ÐµÐ²Ð¾Ð³Ð°     %s  avg %.1f
Ð­Ð½ÐµÑ€Ð³Ð¸Ñ     %s  avg %.1f
Ð¡Ð¾Ð½         %s  avg %.1f

ðŸ“‹ Ð ÐµÐ¶Ð¸Ð¼:
  Ð¡Ð¾Ð½ Ð´Ð¾ 23:00:    %d/%d  %s
  ÐŸÐ¾Ð´ÑŠÑ‘Ð¼ Ð´Ð¾ 10:00: %d/%d  %s
  Ð’ÑÐµ Ð¿Ñ€Ð¸Ñ‘Ð¼Ñ‹ Ð¿Ð¸Ñ‰Ð¸: %d/%d  %s
  Ð’ÑÐµ Ñ‚Ð°Ð±Ð»ÐµÑ‚ÐºÐ¸:    %d/%d  %s
  Ð¤Ð¸Ð·Ð°ÐºÑ‚Ð¸Ð²Ð½Ð¾ÑÑ‚ÑŒ:   %d/%d  %s

ðŸ”¥ Ð›ÑƒÑ‡ÑˆÐ¸Ð¹ Ð´ÐµÐ½ÑŒ:  %s (Ð˜ÐžÐ¡ %.1f)
ðŸ˜“ Ð¢ÑÐ¶Ñ‘Ð»Ñ‹Ð¹ Ð´ÐµÐ½ÑŒ: %s (Ð˜ÐžÐ¡ %.1f)`,
		from.Format("02.01"), to.Format("02.01"), len(reports),
		analytics.Sparkline(moods, 10), avg(moods),
		analytics.Sparkline(anxieties, 5), avg(anxieties),
		analytics.Sparkline(energies, 5), avg(energies),
		analytics.Sparkline(sleeps, 5), avg(sleeps),
		regime.SleepOnTime, regime.Total, progressBar(regime.SleepOnTime, regime.Total),
		regime.WokeOnTime, regime.Total, progressBar(regime.WokeOnTime, regime.Total),
		regime.AllMeals, regime.Total, progressBar(regime.AllMeals, regime.Total),
		regime.AllMeds, regime.Total, progressBar(regime.AllMeds, regime.Total),
		regime.ActiveDays, regime.Total, progressBar(regime.ActiveDays, regime.Total),
		bestDay.ReportDate.Format("02.01"), analytics.WellnessIndex(bestDay),
		worstDay.ReportDate.Format("02.01"), analytics.WellnessIndex(worstDay),
	)

	c.sendMessage(chatID, text)
}

// â”€â”€â”€ /migraine â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (c *ChatBot) handleMigraineCommand(chatID int64, telegramUserID int64) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "ÐÐµ Ð¼Ð¾Ð³Ñƒ Ð½Ð°Ð¹Ñ‚Ð¸ Ð¿Ð¾Ð»ÑŒÐ·Ð¾Ð²Ð°Ñ‚ÐµÐ»Ñ")
		return
	}

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -60)

	reports, _ := c.dailyReportRepo.FindByPeriod(c.ctx, u.ID, from, to)
	report := analytics.AnalyzeMigraineTriggers(reports)

	if report.TotalEpisodes == 0 {
		c.sendMessage(chatID, "ðŸ§  Ð—Ð° Ð¿Ð¾ÑÐ»ÐµÐ´Ð½Ð¸Ðµ 60 Ð´Ð½ÐµÐ¹ Ð¼Ð¸Ð³Ñ€ÐµÐ½ÐµÐ¹ (â‰¥2) Ð½Ðµ Ð·Ð°Ñ„Ð¸ÐºÑÐ¸Ñ€Ð¾Ð²Ð°Ð½Ð¾")
		return
	}

	sideStr := formatSideStats(report.SideStats)

	var sb strings.Builder
	fmt.Fprintf(&sb, "ðŸ§  ÐœÐ¸Ð³Ñ€ÐµÐ½Ð¸ Ð·Ð° 60 Ð´Ð½ÐµÐ¹: %d ÑÐ¿Ð¸Ð·Ð¾Ð´Ð¾Ð²\n\nÐ¡Ñ€ÐµÐ´Ð½Ð¸Ð¹ Ð±Ð°Ð»Ð»: %.1f\nÐ›Ð¾ÐºÐ°Ð»Ð¸Ð·Ð°Ñ†Ð¸Ñ: %s\n\nÐ§Ð°ÑÑ‚Ñ‹Ðµ Ð¿Ñ€ÐµÐ´ÑˆÐµÑÑ‚Ð²ÐµÐ½Ð½Ð¸ÐºÐ¸ (Ð½Ð°ÐºÐ°Ð½ÑƒÐ½Ðµ):",
		report.TotalEpisodes, report.AvgScore, sideStr)

	for _, t := range report.Triggers {
		if t.Count == 0 {
			continue
		}
		icon := triggerIcon(t.Percent)
		fmt.Fprintf(&sb, "\n  %s %s: %d/%d  %.0f%%", icon, t.Label, t.Count, t.Total, t.Percent)
	}

	c.sendMessage(chatID, sb.String())
}

// â”€â”€â”€ /stats â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (c *ChatBot) handleStatsCommand(chatID int64, telegramUserID int64, days int) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "ÐÐµ Ð¼Ð¾Ð³Ñƒ Ð½Ð°Ð¹Ñ‚Ð¸ Ð¿Ð¾Ð»ÑŒÐ·Ð¾Ð²Ð°Ñ‚ÐµÐ»Ñ")
		return
	}

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -days)

	reports, _ := c.dailyReportRepo.FindByPeriod(c.ctx, u.ID, from, to)
	if len(reports) < 7 {
		c.sendMessage(chatID, fmt.Sprintf("ÐœÐ°Ð»Ð¾Ð²Ð°Ñ‚Ð¾ Ð´Ð°Ð½Ð½Ñ‹Ñ… â€” %d Ð´Ð½ÐµÐ¹. ÐÑƒÐ¶Ð½Ð¾ Ñ…Ð¾Ñ‚Ñ Ð±Ñ‹ 7.", len(reports)))
		return
	}

	norm := analytics.PersonalNorm(reports)
	corrs := analytics.TopCorrelations(reports)
	weekday := analytics.ByWeekday(reports)

	text := formatStatsReport(reports, norm, corrs, weekday, days)
	c.sendMessage(chatID, text)
}

// â”€â”€â”€ formatters â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func formatExtras(extras []string) string {
	if len(extras) == 0 {
		return "â˜• Ð‘ÐµÐ· Ð¾ÑÐ¾Ð±ÐµÐ½Ð½Ð¾ÑÑ‚ÐµÐ¹"
	}
	return "â˜• " + strings.Join(extras, ", ")
}

func formatMigraine(r *daily_report.DailyReport) string {
	if r.Migraine == 0 {
		return "Ð½ÐµÑ‚"
	}
	s := fmt.Sprintf("%d/5", r.Migraine)
	if r.MigraineSide != "" {
		s += " Â· " + r.MigraineSide
	}
	if r.MigraineDose != "" {
		s += " Â· " + r.MigraineDose
	}
	return s
}

func formatSideStats(stats map[string]int) string {
	parts := []string{}
	labels := map[string]string{
		"bilateral": "Ð´Ð²ÑƒÑÑ‚Ð¾Ñ€Ð¾Ð½Ð½ÑÑ",
		"left":      "ÑÐ»ÐµÐ²Ð°",
		"right":     "ÑÐ¿Ñ€Ð°Ð²Ð°",
	}
	for key, label := range labels {
		if n, ok := stats[key]; ok && n > 0 {
			parts = append(parts, fmt.Sprintf("%s %dÃ—", label, n))
		}
	}
	if len(parts) == 0 {
		return "Ð½Ðµ ÑƒÐºÐ°Ð·Ð°Ð½Ð°"
	}
	return strings.Join(parts, ", ")
}

func formatStatsReport(
	reports []*daily_report.DailyReport,
	norm analytics.Norm,
	corrs []analytics.CorrelationResult,
	weekday [7]analytics.WeekdayStats,
	days int,
) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "ðŸ“Š Ð¡Ñ‚Ð°Ñ‚Ð¸ÑÑ‚Ð¸ÐºÐ° Ð·Ð° %d Ð´Ð½ÐµÐ¹ (%d Ð·Ð°Ð¿Ð¾Ð»Ð½ÐµÐ½Ð¾)\n\n", days, len(reports))
	fmt.Fprintf(&sb, "Ð¡Ñ€ÐµÐ´Ð½Ð¸Ðµ Ð¿Ð¾ÐºÐ°Ð·Ð°Ñ‚ÐµÐ»Ð¸:\n")
	fmt.Fprintf(&sb, "  ÐÐ°ÑÑ‚Ñ€Ð¾ÐµÐ½Ð¸Ðµ:   %.1f   Ð¢Ñ€ÐµÐ²Ð¾Ð³Ð°:   %.1f\n", norm.Mood, norm.Anxiety)
	fmt.Fprintf(&sb, "  Ð­Ð½ÐµÑ€Ð³Ð¸Ñ:      %.1f   Ð¡Ð¾Ð½:       %.1f\n", norm.Energy, norm.SleepQuality)
	fmt.Fprintf(&sb, "  Ð›Ð¸Ð±Ð¸Ð´Ð¾:       %.1f   ÐœÐ¸Ð³Ñ€ÐµÐ½ÑŒ:   %.1f\n", norm.Libido, norm.Migraine)
	fmt.Fprintf(&sb, "  Ð˜ÐžÐ¡:          %.1f %s\n", norm.WellnessIndex, analytics.WellnessEmoji(norm.WellnessIndex))

	if len(corrs) > 0 {
		fmt.Fprintf(&sb, "\nðŸ”— Ð¡Ð¸Ð»ÑŒÐ½Ñ‹Ðµ ÑÐ²ÑÐ·Ð¸:\n")
		for _, cr := range corrs {
			arrow := "ðŸ“ˆ"
			if cr.R < 0 {
				arrow = "ðŸ“‰"
			}
			fmt.Fprintf(&sb, "  %s %s â†’ %s  r=%+.2f (%s)\n", arrow, cr.LabelA, cr.LabelB, cr.R, cr.Strength)
		}
	}

	// Ð›ÑƒÑ‡ÑˆÐ¸Ð¹ Ð¸ Ñ…ÑƒÐ´ÑˆÐ¸Ð¹ Ð´ÐµÐ½ÑŒ Ð½ÐµÐ´ÐµÐ»Ð¸ Ð¿Ð¾ Ð˜ÐžÐ¡ (ÑÑ€ÐµÐ´Ð½Ð¸Ð¹ Mood ÐºÐ°Ðº Ð¿Ñ€Ð¾ÐºÑÐ¸)
	bestWD, worstWD := bestAndWorstWeekday(weekday)
	if bestWD.Count > 0 {
		fmt.Fprintf(&sb, "\nðŸ“… Ð›ÑƒÑ‡ÑˆÐ¸Ð¹ Ð´ÐµÐ½ÑŒ Ð½ÐµÐ´ÐµÐ»Ð¸: %s (%.1f)\n", ruWeekday(bestWD.Day), bestWD.AvgMood)
	}
	if worstWD.Count > 0 {
		fmt.Fprintf(&sb, "ðŸ“… Ð¢ÑÐ¶Ñ‘Ð»Ñ‹Ð¹ Ð´ÐµÐ½ÑŒ Ð½ÐµÐ´ÐµÐ»Ð¸: %s (%.1f)\n", ruWeekday(worstWD.Day), worstWD.AvgMood)
	}

	// Ð¢Ð¾Ð¿-3 Ð»ÑƒÑ‡ÑˆÐ¸Ñ… Ð¸ Ñ‚ÑÐ¶Ñ‘Ð»Ñ‹Ñ… Ð´Ð½ÐµÐ¹ Ð¿Ð¾ Ð˜ÐžÐ¡
	best3, worst3 := topDays(reports, 3)
	if len(best3) > 0 {
		fmt.Fprintf(&sb, "\nðŸ”¥ Ð¢Ð¾Ð¿ Ð»ÑƒÑ‡ÑˆÐ¸Ñ… Ð´Ð½ÐµÐ¹:  ")
		for i, r := range best3 {
			if i > 0 {
				fmt.Fprintf(&sb, ", ")
			}
			fmt.Fprintf(&sb, "%s (%.1f)", r.ReportDate.Format("02.01"), analytics.WellnessIndex(r))
		}
		fmt.Fprintf(&sb, "\n")
	}
	if len(worst3) > 0 {
		fmt.Fprintf(&sb, "ðŸ˜“ Ð¢Ð¾Ð¿ Ñ‚ÑÐ¶Ñ‘Ð»Ñ‹Ñ… Ð´Ð½ÐµÐ¹: ")
		for i, r := range worst3 {
			if i > 0 {
				fmt.Fprintf(&sb, ", ")
			}
			fmt.Fprintf(&sb, "%s (%.1f)", r.ReportDate.Format("02.01"), analytics.WellnessIndex(r))
		}
	}

	return sb.String()
}

// â”€â”€â”€ helpers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func extractWeekSeries(reports []*daily_report.DailyReport) (moods, anxieties, energies, sleeps []float64) {
	for _, r := range reports {
		moods = append(moods, float64(r.Mood))
		anxieties = append(anxieties, float64(r.Anxiety))
		energies = append(energies, float64(r.Energy))
		sleeps = append(sleeps, float64(r.SleepQuality))
	}
	return
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func progressBar(count, total int) string {
	if total == 0 {
		return ""
	}
	pct := count * 100 / total
	switch {
	case pct == 100:
		return "âœ…"
	case pct >= 70:
		return "ðŸŸ¡"
	default:
		return "ðŸ”´"
	}
}

func bestAndWorst(reports []*daily_report.DailyReport) (best, worst *daily_report.DailyReport) {
	if len(reports) == 0 {
		return nil, nil
	}
	best, worst = reports[0], reports[0]
	for _, r := range reports[1:] {
		if analytics.WellnessIndex(r) > analytics.WellnessIndex(best) {
			best = r
		}
		if analytics.WellnessIndex(r) < analytics.WellnessIndex(worst) {
			worst = r
		}
	}
	return
}

func topDays(reports []*daily_report.DailyReport, n int) (best, worst []*daily_report.DailyReport) {
	type scored struct {
		r     *daily_report.DailyReport
		score float64
	}
	all := make([]scored, len(reports))
	for i, r := range reports {
		all[i] = scored{r, analytics.WellnessIndex(r)}
	}

	// Ð¿Ñ€Ð¾ÑÑ‚Ð°Ñ ÑÐ¾Ñ€Ñ‚Ð¸Ñ€Ð¾Ð²ÐºÐ° Ð¿ÑƒÐ·Ñ‹Ñ€ÑŒÐºÐ¾Ð¼ Ð´Ð»Ñ Ð½ÐµÐ±Ð¾Ð»ÑŒÑˆÐ¸Ñ… N
	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].score > all[i].score {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	for i := 0; i < n && i < len(all); i++ {
		best = append(best, all[i].r)
	}
	for i := len(all) - 1; i >= 0 && len(worst) < n; i-- {
		worst = append(worst, all[i].r)
	}
	return
}

func bestAndWorstWeekday(stats [7]analytics.WeekdayStats) (best, worst analytics.WeekdayStats) {
	best, worst = stats[0], stats[0]
	for _, s := range stats[1:] {
		if s.Count == 0 {
			continue
		}
		if best.Count == 0 || s.AvgMood > best.AvgMood {
			best = s
		}
		if worst.Count == 0 || s.AvgMood < worst.AvgMood {
			worst = s
		}
	}
	return
}

func triggerIcon(pct float64) string {
	switch {
	case pct >= 60:
		return "ðŸ”´"
	case pct >= 30:
		return "ðŸŸ "
	default:
		return "ðŸŸ¡"
	}
}

func ruWeekday(wd time.Weekday) string {
	names := [7]string{"Ð²Ñ", "Ð¿Ð½", "Ð²Ñ‚", "ÑÑ€", "Ñ‡Ñ‚", "Ð¿Ñ‚", "ÑÐ±"}
	return names[wd]
}
