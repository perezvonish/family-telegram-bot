package analytics

import (
	"time"

	"perezvonish/health-tracker/internal/domain/daily_report"
)

// WeekdayStats — средние показатели по одному дню недели.
type WeekdayStats struct {
	Day                            time.Weekday
	AvgMood, AvgAnxiety, AvgEnergy float64
	AvgMigraine, AvgSleepQuality   float64
	Count                          int
}

// ByWeekday группирует записи по дням недели и возвращает средние показатели.
// Индекс массива соответствует time.Weekday (0=Sunday … 6=Saturday).
func ByWeekday(reports []*daily_report.DailyReport) [7]WeekdayStats {
	var sums [7]struct {
		mood, anxiety, energy, migraine, sleep float64
		count                                  int
	}

	for _, r := range reports {
		wd := r.ReportDate.Weekday()
		sums[wd].mood += float64(r.Mood)
		sums[wd].anxiety += float64(r.Anxiety)
		sums[wd].energy += float64(r.Energy)
		sums[wd].migraine += float64(r.Migraine)
		sums[wd].sleep += float64(r.SleepQuality)
		sums[wd].count++
	}

	var result [7]WeekdayStats
	for wd := 0; wd < 7; wd++ {
		result[wd].Day = time.Weekday(wd)
		result[wd].Count = sums[wd].count
		if sums[wd].count == 0 {
			continue
		}
		cnt := float64(sums[wd].count)
		result[wd].AvgMood = sums[wd].mood / cnt
		result[wd].AvgAnxiety = sums[wd].anxiety / cnt
		result[wd].AvgEnergy = sums[wd].energy / cnt
		result[wd].AvgMigraine = sums[wd].migraine / cnt
		result[wd].AvgSleepQuality = sums[wd].sleep / cnt
	}
	return result
}
