package analytics

import "perezvonish/health-tracker/internal/domain/daily_report"

// WellnessIndex — Индекс общего самочувствия, диапазон 1–10
func WellnessIndex(r *daily_report.DailyReport) float64 {
	mood := float64(r.Mood) * 0.35                  // 1–10, вес 35%
	energy := float64(r.Energy*2) * 0.20            // 1–5×2, вес 20%
	sleep := float64(r.SleepQuality*2) * 0.20       // 1–5×2, вес 20%
	anxietyInv := float64((6-r.Anxiety)*2) * 0.15   // инверсия, вес 15%
	migraineInv := float64((6-r.Migraine)*2) * 0.10 // инверсия, вес 10%
	return mood + energy + sleep + anxietyInv + migraineInv
}

// RelationshipIndex — Индекс отношений, диапазон 1–5
func RelationshipIndex(r *daily_report.DailyReport) float64 {
	return float64(r.Relationship)*0.55 + float64(r.Closeness)*0.45
}

// Norm — личная норма: среднее каждого поля по набору записей
type Norm struct {
	Mood, Anxiety, Energy, SleepQuality, Libido float64
	Migraine, Relationship, Closeness           float64
	WellnessIndex                               float64
}

// PersonalNorm считает среднее по переданному набору записей
func PersonalNorm(reports []*daily_report.DailyReport) Norm {
	if len(reports) == 0 {
		return Norm{}
	}
	var n Norm
	for _, r := range reports {
		n.Mood += float64(r.Mood)
		n.Anxiety += float64(r.Anxiety)
		n.Energy += float64(r.Energy)
		n.SleepQuality += float64(r.SleepQuality)
		n.Libido += float64(r.Libido)
		n.Migraine += float64(r.Migraine)
		n.Relationship += float64(r.Relationship)
		n.Closeness += float64(r.Closeness)
		n.WellnessIndex += WellnessIndex(r)
	}
	cnt := float64(len(reports))
	n.Mood /= cnt
	n.Anxiety /= cnt
	n.Energy /= cnt
	n.SleepQuality /= cnt
	n.Libido /= cnt
	n.Migraine /= cnt
	n.Relationship /= cnt
	n.Closeness /= cnt
	n.WellnessIndex /= cnt
	return n
}
