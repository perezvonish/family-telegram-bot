package daily_report

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, report *DailyReport) error
}
