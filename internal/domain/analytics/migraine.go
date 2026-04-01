package analytics

import (
	"slices"
	"sort"

	"perezvonish/health-tracker/internal/domain/daily_report"
)

// MigraineTriggerReport — сводка по эпизодам мигрени и их предшественникам.
type MigraineTriggerReport struct {
	TotalEpisodes int
	AvgScore      float64
	SideStats     map[string]int // "bilateral" / "left" / "right"
	Triggers      []TriggerStat  // отсортировано по убыванию %
}

// TriggerStat описывает один потенциальный триггер мигрени.
type TriggerStat struct {
	Label   string
	Count   int
	Total   int
	Percent float64
}

// AnalyzeMigraineTriggers анализирует триггеры для дней с migraine >= 2,
// проверяя предыдущий день по каждому фактору.
func AnalyzeMigraineTriggers(reports []*daily_report.DailyReport) MigraineTriggerReport {
	result := MigraineTriggerReport{
		SideStats: make(map[string]int),
	}

	var totalScore float64
	triggers := map[string]int{
		"Алкоголь накануне":       0,
		"Менструация накануне":    0,
		"Голодание накануне":      0,
		"Тревога ≥ 4 накануне":    0,
		"Плохой сон ≤ 2 накануне": 0,
	}

	for i, r := range reports {
		if r.Migraine < 2 {
			continue
		}
		result.TotalEpisodes++
		totalScore += float64(r.Migraine)

		if r.MigraineSide != "" {
			result.SideStats[r.MigraineSide]++
		}

		if i == 0 {
			continue
		}
		prev := reports[i-1]

		if slices.Contains(prev.Extras, "Алкоголь") {
			triggers["Алкоголь накануне"]++
		}
		if prev.Menstruation == "да" {
			triggers["Менструация накануне"]++
		}
		if prev.Fasting != "нет" {
			triggers["Голодание накануне"]++
		}
		if prev.Anxiety >= 4 {
			triggers["Тревога ≥ 4 накануне"]++
		}
		if prev.SleepQuality <= 2 {
			triggers["Плохой сон ≤ 2 накануне"]++
		}
	}

	if result.TotalEpisodes > 0 {
		result.AvgScore = totalScore / float64(result.TotalEpisodes)
	}

	for label, count := range triggers {
		result.Triggers = append(result.Triggers, TriggerStat{
			Label:   label,
			Count:   count,
			Total:   result.TotalEpisodes,
			Percent: percentOf(count, result.TotalEpisodes),
		})
	}

	sort.Slice(result.Triggers, func(i, j int) bool {
		return result.Triggers[i].Percent > result.Triggers[j].Percent
	})

	return result
}

func percentOf(count, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(count) / float64(total) * 100
}
