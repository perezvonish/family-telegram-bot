package pill_tracker

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type PillTracker struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Name      string    `json:"name"`
	Total     int       `json:"total"`
	DailyDose float64   `json:"dailyDose"`
	StartDate time.Time `json:"startDate"`

	Notified7d    bool `json:"notified7d"`
	Notified3d    bool `json:"notified3d"`
	Notified1d    bool `json:"notified1d"`
	NotifiedEmpty bool `json:"notifiedEmpty"`
}

func (p *PillTracker) Remaining() float64 {
	days := time.Since(p.StartDate).Hours() / 24
	used := days * p.DailyDose
	return math.Max(0, float64(p.Total)-used)
}

func (p *PillTracker) DaysLeft() float64 {
	if p.DailyDose == 0 {
		return 0
	}
	return p.Remaining() / p.DailyDose
}

func (p *PillTracker) EmptyDate() time.Time {
	return time.Now().Add(time.Duration(p.DaysLeft() * float64(24*time.Hour)))
}

func (p *PillTracker) IsEmpty() bool {
	return p.Remaining() <= 0
}

func NewPillTracker(userID uuid.UUID, name string, total int, dailyDose float64) *PillTracker {
	now := time.Now().UTC()
	return &PillTracker{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
		Name:      name,
		Total:     total,
		DailyDose: dailyDose,
		StartDate: now,
	}
}
