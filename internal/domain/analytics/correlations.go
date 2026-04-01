package analytics

import (
	"math"
	"slices"
	"sort"

	"perezvonish/health-tracker/internal/domain/daily_report"
)

// Pearson вычисляет коэффициент корреляции Пирсона между двумя рядами.
// Возвращает 0 если данных меньше 3.
func Pearson(x, y []float64) float64 {
	n := float64(len(x))
	if n < 3 {
		return 0
	}
	var sx, sy, sxy, sx2, sy2 float64
	for i := range x {
		sx += x[i]
		sy += y[i]
		sxy += x[i] * y[i]
		sx2 += x[i] * x[i]
		sy2 += y[i] * y[i]
	}
	num := sxy - sx*sy/n
	den := math.Sqrt((sx2 - sx*sx/n) * (sy2 - sy*sy/n))
	if den == 0 {
		return 0
	}
	return num / den
}

// CorrelationResult описывает значимую корреляцию между двумя показателями.
type CorrelationResult struct {
	LabelA, LabelB string
	R              float64 // -1..+1
	Strength       string  // "очень сильная" / "сильная" / "умеренная"
}

func correlationStrength(r float64) string {
	abs := math.Abs(r)
	switch {
	case abs >= 0.7:
		return "очень сильная"
	case abs >= 0.5:
		return "сильная"
	default:
		return "умеренная"
	}
}

// TopCorrelations возвращает значимые пары (|r| > 0.35), отсортированные по убыванию |r|.
func TopCorrelations(reports []*daily_report.DailyReport) []CorrelationResult {
	n := len(reports)
	if n < 3 {
		return nil
	}

	// Числовые ряды
	mood := make([]float64, n)
	anxiety := make([]float64, n)
	energy := make([]float64, n)
	sleepQ := make([]float64, n)
	migraine := make([]float64, n)
	relationship := make([]float64, n)
	closeness := make([]float64, n)
	alcohol := make([]float64, n)
	coffee := make([]float64, n)
	fasting := make([]float64, n)
	menstruation := make([]float64, n)

	for i, r := range reports {
		mood[i] = float64(r.Mood)
		anxiety[i] = float64(r.Anxiety)
		energy[i] = float64(r.Energy)
		sleepQ[i] = float64(r.SleepQuality)
		migraine[i] = float64(r.Migraine)
		relationship[i] = float64(r.Relationship)
		closeness[i] = float64(r.Closeness)
		if slices.Contains(r.Extras, "Алкоголь") {
			alcohol[i] = 1
		}
		if slices.Contains(r.Extras, "Кофе") {
			coffee[i] = 1
		}
		if r.Fasting != "нет" {
			fasting[i] = 1
		}
		if r.Menstruation == "да" {
			menstruation[i] = 1
		}
	}

	// Ряды со сдвигом +1 день для анализа эффекта следующего дня
	alcoholNext := shiftNext(alcohol)
	fastingNext := shiftNext(fasting)
	menstruationNext := shiftNext(menstruation)
	coffeeNext := shiftNext(coffee)

	pairs := []struct {
		a, b           []float64
		labelA, labelB string
	}{
		{sleepQ, mood, "Сон", "Настроение"},
		{sleepQ, energy, "Сон", "Энергия"},
		{anxiety, mood, "Тревога", "Настроение"},
		{migraine, mood, "Мигрень", "Настроение"},
		{migraine, energy, "Мигрень", "Энергия"},
		{relationship, mood, "Отношения", "Настроение"},
		{closeness, mood, "Близость", "Настроение"},
		{alcoholNext, migraine, "Алкоголь", "Мигрень (+1д)"},
		{coffeeNext, anxiety, "Кофе", "Тревога"},
		{fastingNext, migraine, "Голодание", "Мигрень"},
		{menstruationNext, migraine, "Менструация", "Мигрень"},
	}

	var results []CorrelationResult
	for _, p := range pairs {
		r := Pearson(p.a, p.b)
		if math.Abs(r) >= 0.35 {
			results = append(results, CorrelationResult{
				LabelA:   p.labelA,
				LabelB:   p.labelB,
				R:        r,
				Strength: correlationStrength(r),
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return math.Abs(results[i].R) > math.Abs(results[j].R)
	})

	return results
}

// shiftNext сдвигает срез на один элемент вперёд (эффект следующего дня).
// reports[i] → влияние на reports[i+1], поэтому возвращаем срез длиной n,
// где [0] = 0 (нет предыдущего), [i] = original[i-1].
func shiftNext(values []float64) []float64 {
	result := make([]float64, len(values))
	for i := 1; i < len(values); i++ {
		result[i] = values[i-1]
	}
	return result
}
