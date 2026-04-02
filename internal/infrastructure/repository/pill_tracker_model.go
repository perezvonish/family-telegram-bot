package repository

import (
	"time"

	"perezvonish/health-tracker/internal/domain/pill_tracker"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PillTrackerModel struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UUID      string             `bson:"uuid"`
	UserID    string             `bson:"user_id"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`

	Name      string    `bson:"name"`
	Total     int       `bson:"total"`
	DailyDose float64   `bson:"daily_dose"`
	StartDate time.Time `bson:"start_date"`

	Notified7d    bool `bson:"notified_7d"`
	Notified3d    bool `bson:"notified_3d"`
	Notified1d    bool `bson:"notified_1d"`
	NotifiedEmpty bool `bson:"notified_empty"`
}

func toPillTrackerModel(e *pill_tracker.PillTracker) *PillTrackerModel {
	return &PillTrackerModel{
		UUID:          e.ID.String(),
		UserID:        e.UserID.String(),
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
		Name:          e.Name,
		Total:         e.Total,
		DailyDose:     e.DailyDose,
		StartDate:     e.StartDate,
		Notified7d:    e.Notified7d,
		Notified3d:    e.Notified3d,
		Notified1d:    e.Notified1d,
		NotifiedEmpty: e.NotifiedEmpty,
	}
}

func toPillTrackerEntity(m *PillTrackerModel) *pill_tracker.PillTracker {
	id, _ := uuid.Parse(m.UUID)
	userID, _ := uuid.Parse(m.UserID)
	return &pill_tracker.PillTracker{
		ID:            id,
		UserID:        userID,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		Name:          m.Name,
		Total:         m.Total,
		DailyDose:     m.DailyDose,
		StartDate:     m.StartDate,
		Notified7d:    m.Notified7d,
		Notified3d:    m.Notified3d,
		Notified1d:    m.Notified1d,
		NotifiedEmpty: m.NotifiedEmpty,
	}
}
