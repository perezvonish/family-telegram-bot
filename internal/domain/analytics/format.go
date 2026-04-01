package analytics

import "perezvonish/health-tracker/internal/domain/daily_report"

// Sparkline строит строку из блочных символов по срезу значений.
// maxVal — максимально возможное значение шкалы.
func Sparkline(values []float64, maxVal float64) string {
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	result := ""
	for _, v := range values {
		idx := int((v / maxVal) * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		result += string(blocks[idx])
	}
	return result
}

// DeltaLabel возвращает стрелку-метку относительно нормы.
func DeltaLabel(current, norm float64) string {
	diff := current - norm
	switch {
	case diff > 1.5:
		return "↑↑"
	case diff > 0.5:
		return "↑"
	case diff < -1.5:
		return "↓ ⚠️"
	case diff < -0.5:
		return "↓"
	default:
		return "—"
	}
}

// WellnessEmoji возвращает цветовой индикатор по значению ИОС.
func WellnessEmoji(score float64) string {
	switch {
	case score >= 8:
		return "🟢"
	case score >= 6:
		return "🟡"
	case score >= 4:
		return "🟠"
	default:
		return "🔴"
	}
}

// RegimeStats — статистика соблюдения режима за период.
type RegimeStats struct {
	SleepOnTime int // дней легла до 23:00 включительно
	WokeOnTime  int // дней встала до 10:00 включительно
	AllMeals    int // дней без пропущенных приёмов пищи
	AllMeds     int // дней без проблем с таблетками
	ActiveDays  int // дней с физической активностью
	Total       int // всего дней в периоде
}

// RegimeScore подсчитывает показатели режима за набор записей.
func RegimeScore(reports []*daily_report.DailyReport) RegimeStats {
	stats := RegimeStats{Total: len(reports)}
	for _, r := range reports {
		if r.SleepTime == "раньше 22:00" || r.SleepTime == "22:00" || r.SleepTime == "23:00" {
			stats.SleepOnTime++
		}
		if r.WakeTime != "позже 12:00" && r.WakeTime != "12:00" {
			stats.WokeOnTime++
		}
		if len(r.MealsSkipped) == 0 {
			stats.AllMeals++
		}
		if len(r.MedsIssues) == 0 {
			stats.AllMeds++
		}
		if r.Activity != "Не было" {
			stats.ActiveDays++
		}
	}
	return stats
}
