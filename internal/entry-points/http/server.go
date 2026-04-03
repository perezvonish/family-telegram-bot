package httpentry

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"perezvonish/health-tracker/internal/domain/analytics"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/domain/user"
	"perezvonish/health-tracker/internal/shared/config"

	"github.com/google/uuid"
)

//go:embed web/*
var embeddedWeb embed.FS

type Server struct {
	ctx             context.Context
	cfg             config.ServerConfig
	userRepo        user.Repository
	dailyReportRepo daily_report.Repository
	httpServer      *http.Server
}

func NewServer(
	ctx context.Context,
	cfg config.ServerConfig,
	userRepo user.Repository,
	dailyReportRepo daily_report.Repository,
) *Server {
	s := &Server{
		ctx:             ctx,
		cfg:             cfg,
		userRepo:        userRepo,
		dailyReportRepo: dailyReportRepo,
	}

	mux := http.NewServeMux()
	mux.Handle("/", s.makeWebHandler())
	mux.Handle("/api/reports", s.withAuth(http.HandlerFunc(s.handleReports)))
	mux.Handle("/api/stats", s.withAuth(http.HandlerFunc(s.handleStats)))
	mux.Handle("/api/correlations", s.withAuth(http.HandlerFunc(s.handleCorrelations)))
	mux.Handle("/api/migraine", s.withAuth(http.HandlerFunc(s.handleMigraine)))
	mux.Handle("/api/weekday", s.withAuth(http.HandlerFunc(s.handleWeekday)))

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	log.Printf("HTTP dashboard started on %s", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) makeWebHandler() http.Handler {
	sub, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		log.Printf("failed to init embedded web filesystem: %v", err)
		return http.NotFoundHandler()
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			if !hasTelegramIdentity(r) {
				serveAccessDeniedPage(w)
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.BearerToken == "" {
			writeError(w, http.StatusServiceUnavailable, "HTTP_BEARER_TOKEN is not configured")
			return
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.BearerToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	days, err := parseDays(r, 30)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	u, status, err := s.resolveUser(r)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}

	reports, err := s.reportsByDays(u.ID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reports")
		return
	}

	type reportWithWellness struct {
		*daily_report.DailyReport
		WellnessIndex float64 `json:"wellnessIndex"`
	}

	payload := make([]reportWithWellness, 0, len(reports))
	for _, item := range reports {
		payload = append(payload, reportWithWellness{
			DailyReport:   item,
			WellnessIndex: analytics.WellnessIndex(item),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"days":          days,
		"user":          u,
		"reports":       payload,
		"totalReports":  len(payload),
		"generatedAt":   time.Now().UTC(),
		"requestedDays": days,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	days, err := parseDays(r, 30)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	u, status, err := s.resolveUser(r)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}

	reports, err := s.reportsByDays(u.ID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reports")
		return
	}
	if len(reports) == 0 {
		writeError(w, http.StatusNotFound, "no reports for requested period")
		return
	}

	best, worst := topDays(reports, 3)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"days":         days,
		"user":         u,
		"reportsCount": len(reports),
		"norm":         analytics.PersonalNorm(reports),
		"regime":       analytics.RegimeScore(reports),
		"bestDays":     best,
		"worstDays":    worst,
		"generatedAt":  time.Now().UTC(),
	})
}

func (s *Server) handleCorrelations(w http.ResponseWriter, r *http.Request) {
	days, err := parseDays(r, 30)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	u, status, err := s.resolveUser(r)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}

	reports, err := s.reportsByDays(u.ID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reports")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"days":          days,
		"user":          u,
		"reportsCount":  len(reports),
		"correlations":  analytics.TopCorrelations(reports),
		"generatedAt":   time.Now().UTC(),
		"requestedDays": days,
	})
}

func (s *Server) handleMigraine(w http.ResponseWriter, r *http.Request) {
	days, err := parseDays(r, 60)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	u, status, err := s.resolveUser(r)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}

	reports, err := s.reportsByDays(u.ID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reports")
		return
	}

	type migraineEpisode struct {
		Date  time.Time `json:"date"`
		Score int       `json:"score"`
		Side  string    `json:"side,omitempty"`
		Dose  string    `json:"dose,omitempty"`
	}
	episodes := make([]migraineEpisode, 0)
	for _, item := range reports {
		if item.Migraine >= 2 {
			episodes = append(episodes, migraineEpisode{
				Date:  item.ReportDate,
				Score: item.Migraine,
				Side:  item.MigraineSide,
				Dose:  item.MigraineDose,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"days":         days,
		"user":         u,
		"reportsCount": len(reports),
		"report":       analytics.AnalyzeMigraineTriggers(reports),
		"episodes":     episodes,
		"generatedAt":  time.Now().UTC(),
	})
}

func (s *Server) handleWeekday(w http.ResponseWriter, r *http.Request) {
	days, err := parseDays(r, 90)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	u, status, err := s.resolveUser(r)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}

	reports, err := s.reportsByDays(u.ID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reports")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"days":         days,
		"user":         u,
		"reportsCount": len(reports),
		"weekday":      analytics.ByWeekday(reports),
		"generatedAt":  time.Now().UTC(),
	})
}

func (s *Server) resolveUser(r *http.Request) (*user.User, int, error) {
	username := strings.TrimSpace(r.URL.Query().Get("telegram_username"))
	if username == "" {
		username = strings.TrimSpace(r.URL.Query().Get("username"))
	}
	if username != "" {
		u, err := s.userRepo.FindByUsername(s.ctx, username)
		if err != nil {
			return nil, http.StatusNotFound, fmt.Errorf("user with username=%q not found", username)
		}
		return u, http.StatusOK, nil
	}

	telegramIDRaw := strings.TrimSpace(r.URL.Query().Get("telegram_id"))
	if telegramIDRaw != "" {
		telegramID, err := strconv.ParseInt(telegramIDRaw, 10, 64)
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("telegram_id must be an integer")
		}

		u, err := s.userRepo.FindByTelegramID(s.ctx, telegramID)
		if err != nil {
			return nil, http.StatusNotFound, fmt.Errorf("user with telegram_id=%d not found", telegramID)
		}
		return u, http.StatusOK, nil
	}

	users, err := s.userRepo.FindAll(s.ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load users")
	}
	if len(users) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("no users found")
	}
	if len(users) > 1 {
		return nil, http.StatusBadRequest, fmt.Errorf("multiple users found; pass telegram_id query parameter")
	}
	return users[0], http.StatusOK, nil
}

func (s *Server) reportsByDays(userID uuid.UUID, days int) ([]*daily_report.DailyReport, error) {
	to := time.Now().UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -days)
	return s.dailyReportRepo.FindByPeriod(s.ctx, userID, from, to)
}

func parseDays(r *http.Request, fallback int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("days"))
	if raw == "" {
		return fallback, nil
	}

	days, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("days must be an integer")
	}
	if days < 1 || days > 3650 {
		return 0, fmt.Errorf("days must be in range 1..3650")
	}
	return days, nil
}

type scoredDay struct {
	ReportDate    time.Time `json:"reportDate"`
	WellnessIndex float64   `json:"wellnessIndex"`
}

func topDays(reports []*daily_report.DailyReport, n int) (best, worst []scoredDay) {
	scored := make([]scoredDay, 0, len(reports))
	for _, r := range reports {
		scored = append(scored, scoredDay{
			ReportDate:    r.ReportDate,
			WellnessIndex: analytics.WellnessIndex(r),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].WellnessIndex > scored[j].WellnessIndex
	})

	for i := 0; i < n && i < len(scored); i++ {
		best = append(best, scored[i])
	}
	for i := len(scored) - 1; i >= 0 && len(worst) < n; i-- {
		worst = append(worst, scored[i])
	}
	return best, worst
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func hasTelegramIdentity(r *http.Request) bool {
	q := r.URL.Query()
	if strings.TrimSpace(q.Get("telegram_username")) != "" {
		return true
	}
	if strings.TrimSpace(q.Get("username")) != "" {
		return true
	}
	if strings.TrimSpace(q.Get("telegram_id")) != "" {
		return true
	}
	// Telegram WebApp payload (if dashboard will be moved to WebApp mode).
	if strings.TrimSpace(q.Get("tgWebAppData")) != "" {
		return true
	}

	// Telegram WebView often does not pass identity in URL query.
	ua := strings.ToLower(strings.TrimSpace(r.UserAgent()))
	if strings.Contains(ua, "telegram") {
		return true
	}

	referer := strings.ToLower(strings.TrimSpace(r.Referer()))
	if strings.Contains(referer, "t.me") || strings.Contains(referer, "telegram") {
		return true
	}

	return false
}

func serveAccessDeniedPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Доступ запрещен</title>
  <style>
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      background:
        radial-gradient(circle at 18% 20%, rgba(251, 146, 60, 0.25), transparent 34%),
        radial-gradient(circle at 82% 80%, rgba(59, 130, 246, 0.24), transparent 36%),
        linear-gradient(160deg, #0b1020, #182138 48%, #0f172a);
      color: #f8fafc;
      font-family: "Segoe UI", sans-serif;
    }
    h1 {
      margin: 0 16px;
      font-size: clamp(32px, 6vw, 72px);
      font-weight: 800;
      letter-spacing: 0.02em;
      text-transform: uppercase;
      text-shadow: 0 10px 30px rgba(15, 23, 42, 0.55);
    }
  </style>
</head>
<body>
  <h1>Доступ запрещен</h1>
</body>
</html>`))
}
