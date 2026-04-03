package daily_report

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, report *DailyReport) error

	// Все записи за период (включительно), отсортированные по дате
	FindByPeriod(ctx context.Context, userID string, from, to time.Time) ([]*DailyReport, error)

	// Последние N записей (для нормы)
	FindLatest(ctx context.Context, userID string, limit int) ([]*DailyReport, error)

	// Запись за конкретный день (для /today)
	FindByDate(ctx context.Context, userID string, date time.Time) (*DailyReport, error)
}
