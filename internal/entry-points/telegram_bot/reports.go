package telegram_bot

import (
	"fmt"
	"strings"
	"time"

	"perezvonish/health-tracker/internal/domain/analytics"
	"perezvonish/health-tracker/internal/domain/daily_report"
)

// ─── /help ───────────────────────────────────────────────────────────────────

func (c *ChatBot) handleHelpCommand(chatID int64) {
	text := `📋 Доступные команды:

/diary — заполнить дневник здоровья
/today — итог сегодняшнего дня с ИОС и сравнением с нормой
/week — спарклайны и режимный score за последние 7 дней
/stats — средние показатели, корреляции, лучшие и тяжёлые дни за 30 дней
/stats 90 — то же за 90 дней
/migraine — статистика мигреней и топ триггеров за 60 дней
/help — это сообщение`
	c.sendMessage(chatID, text)
}

// ─── /today ──────────────────────────────────────────────────────────────────

func (c *ChatBot) handleTodayCommand(chatID int64, telegramUserID int64) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "Не могу найти пользователя")
		return
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	report, err := c.dailyReportRepo.FindByDate(c.ctx, u.ID, today)
	if err != nil {
		c.sendMessage(chatID, "Сегодня дневник ещё не заполнен. Используй /diary")
		return
	}

	past, _ := c.dailyReportRepo.FindLatest(c.ctx, u.ID, 30)
	norm := analytics.PersonalNorm(past)
	ios := analytics.WellnessIndex(report)

	text := fmt.Sprintf(`📊 Итог дня — %s

ИОС: %.1f %s

😌 Настроение:   %d/10  %s
😰 Тревога:      %d/5   %s
⚡ Энергия:      %d/5   %s
😴 Сон:          %d/5   %s
🌹 Либидо:       %d/5   %s

🔄 Стабильность: %s

%s

🤝 Отношения: %d/5 · Близость: %d/5
🧠 Мигрень: %s`,
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
		text += fmt.Sprintf("\n\n💬 \"%s\"", report.DayComment)
	}

	c.sendMessage(chatID, text)
}

// ─── /week ────────────────────────────────────────────────────────────────────

func (c *ChatBot) handleWeekCommand(chatID int64, telegramUserID int64) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "Не могу найти пользователя")
		return
	}

	fmt.Println(u)

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -6)

	reports, _ := c.dailyReportRepo.FindByPeriod(c.ctx, u.ID, from, to)
	if len(reports) == 0 {
		c.sendMessage(chatID, "Данных за неделю нет")
		return
	}

	moods, anxieties, energies, sleeps := extractWeekSeries(reports)
	regime := analytics.RegimeScore(reports)

	// Лучший и худший день по ИОС
	bestDay, worstDay := bestAndWorst(reports)

	text := fmt.Sprintf(`📈 Неделя %s — %s (%d дней)

Настроение  %s  avg %.1f
Тревога     %s  avg %.1f
Энергия     %s  avg %.1f
Сон         %s  avg %.1f

📋 Режим:
  Сон до 23:00:    %d/%d  %s
  Подъём до 10:00: %d/%d  %s
  Все приёмы пищи: %d/%d  %s
  Все таблетки:    %d/%d  %s
  Физактивность:   %d/%d  %s

🔥 Лучший день:  %s (ИОС %.1f)
😓 Тяжёлый день: %s (ИОС %.1f)`,
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

// ─── /migraine ────────────────────────────────────────────────────────────────

func (c *ChatBot) handleMigraineCommand(chatID int64, telegramUserID int64) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "Не могу найти пользователя")
		return
	}

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -60)

	reports, _ := c.dailyReportRepo.FindByPeriod(c.ctx, u.ID, from, to)
	report := analytics.AnalyzeMigraineTriggers(reports)

	if report.TotalEpisodes == 0 {
		c.sendMessage(chatID, "🧠 За последние 60 дней мигреней (≥2) не зафиксировано")
		return
	}

	sideStr := formatSideStats(report.SideStats)

	var sb strings.Builder
	fmt.Fprintf(&sb, "🧠 Мигрени за 60 дней: %d эпизодов\n\nСредний балл: %.1f\nЛокализация: %s\n\nЧастые предшественники (накануне):",
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

// ─── /stats ───────────────────────────────────────────────────────────────────

func (c *ChatBot) handleStatsCommand(chatID int64, telegramUserID int64, days int) {
	u, err := c.userRepo.FindByTelegramID(c.ctx, telegramUserID)
	if err != nil {
		c.sendMessage(chatID, "Не могу найти пользователя")
		return
	}

	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -days)

	reports, _ := c.dailyReportRepo.FindByPeriod(c.ctx, u.ID, from, to)
	if len(reports) < 7 {
		c.sendMessage(chatID, fmt.Sprintf("Маловато данных — %d дней. Нужно хотя бы 7.", len(reports)))
		return
	}

	norm := analytics.PersonalNorm(reports)
	corrs := analytics.TopCorrelations(reports)
	weekday := analytics.ByWeekday(reports)

	text := formatStatsReport(reports, norm, corrs, weekday, days)
	c.sendMessage(chatID, text)
}

// ─── formatters ──────────────────────────────────────────────────────────────

func formatExtras(extras []string) string {
	if len(extras) == 0 {
		return "☕ Без особенностей"
	}
	return "☕ " + strings.Join(extras, ", ")
}

func formatMigraine(r *daily_report.DailyReport) string {
	if r.Migraine == 0 {
		return "нет"
	}
	s := fmt.Sprintf("%d/5", r.Migraine)
	if r.MigraineSide != "" {
		s += " · " + r.MigraineSide
	}
	if r.MigraineDose != "" {
		s += " · " + r.MigraineDose
	}
	return s
}

func formatSideStats(stats map[string]int) string {
	parts := []string{}
	labels := map[string]string{
		"bilateral": "двусторонняя",
		"left":      "слева",
		"right":     "справа",
	}
	for key, label := range labels {
		if n, ok := stats[key]; ok && n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d×", label, n))
		}
	}
	if len(parts) == 0 {
		return "не указана"
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

	fmt.Fprintf(&sb, "📊 Статистика за %d дней (%d заполнено)\n\n", days, len(reports))
	fmt.Fprintf(&sb, "Средние показатели:\n")
	fmt.Fprintf(&sb, "  Настроение:   %.1f   Тревога:   %.1f\n", norm.Mood, norm.Anxiety)
	fmt.Fprintf(&sb, "  Энергия:      %.1f   Сон:       %.1f\n", norm.Energy, norm.SleepQuality)
	fmt.Fprintf(&sb, "  Либидо:       %.1f   Мигрень:   %.1f\n", norm.Libido, norm.Migraine)
	fmt.Fprintf(&sb, "  ИОС:          %.1f %s\n", norm.WellnessIndex, analytics.WellnessEmoji(norm.WellnessIndex))

	if len(corrs) > 0 {
		fmt.Fprintf(&sb, "\n🔗 Сильные связи:\n")
		for _, cr := range corrs {
			arrow := "📈"
			if cr.R < 0 {
				arrow = "📉"
			}
			fmt.Fprintf(&sb, "  %s %s → %s  r=%+.2f (%s)\n", arrow, cr.LabelA, cr.LabelB, cr.R, cr.Strength)
		}
	}

	// Лучший и худший день недели по ИОС (средний Mood как прокси)
	bestWD, worstWD := bestAndWorstWeekday(weekday)
	if bestWD.Count > 0 {
		fmt.Fprintf(&sb, "\n📅 Лучший день недели: %s (%.1f)\n", ruWeekday(bestWD.Day), bestWD.AvgMood)
	}
	if worstWD.Count > 0 {
		fmt.Fprintf(&sb, "📅 Тяжёлый день недели: %s (%.1f)\n", ruWeekday(worstWD.Day), worstWD.AvgMood)
	}

	// Топ-3 лучших и тяжёлых дней по ИОС
	best3, worst3 := topDays(reports, 3)
	if len(best3) > 0 {
		fmt.Fprintf(&sb, "\n🔥 Топ лучших дней:  ")
		for i, r := range best3 {
			if i > 0 {
				fmt.Fprintf(&sb, ", ")
			}
			fmt.Fprintf(&sb, "%s (%.1f)", r.ReportDate.Format("02.01"), analytics.WellnessIndex(r))
		}
		fmt.Fprintf(&sb, "\n")
	}
	if len(worst3) > 0 {
		fmt.Fprintf(&sb, "😓 Топ тяжёлых дней: ")
		for i, r := range worst3 {
			if i > 0 {
				fmt.Fprintf(&sb, ", ")
			}
			fmt.Fprintf(&sb, "%s (%.1f)", r.ReportDate.Format("02.01"), analytics.WellnessIndex(r))
		}
	}

	return sb.String()
}

// ─── helpers ─────────────────────────────────────────────────────────────────

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
		return "✅"
	case pct >= 70:
		return "🟡"
	default:
		return "🔴"
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

	// простая сортировка пузырьком для небольших N
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
		return "🔴"
	case pct >= 30:
		return "🟠"
	default:
		return "🟡"
	}
}

func ruWeekday(wd time.Weekday) string {
	names := [7]string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"}
	return names[wd]
}
