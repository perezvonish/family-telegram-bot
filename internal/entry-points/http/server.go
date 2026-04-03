package httpentry

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"perezvonish/health-tracker/internal/domain/analytics"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/domain/user"
	"perezvonish/health-tracker/internal/shared/config"
)

//go:embed web/*
var embeddedWeb embed.FS

type Server struct {
	ctx             context.Context
	cfg             config.ServerConfig
	botToken        string
	userRepo        user.Repository
	dailyReportRepo daily_report.Repository
	httpServer      *http.Server
}

func NewServer(
	ctx context.Context,
	cfg config.ServerConfig,
	botToken string,
	userRepo user.Repository,
	dailyReportRepo daily_report.Repository,
) *Server {
	s := &Server{
		ctx:             ctx,
		cfg:             cfg,
		botToken:        botToken,
		userRepo:        userRepo,
		dailyReportRepo: dailyReportRepo,
	}

	mux := http.NewServeMux()
	mux.Handle("/", s.makeWebHandler())
	mux.Handle("/api/reports", s.withTelegramAuth(http.HandlerFunc(s.handleReports)))
	mux.Handle("/api/stats", s.withTelegramAuth(http.HandlerFunc(s.handleStats)))
	mux.Handle("/api/correlations", s.withTelegramAuth(http.HandlerFunc(s.handleCorrelations)))
	mux.Handle("/api/migraine", s.withTelegramAuth(http.HandlerFunc(s.handleMigraine)))
	mux.Handle("/api/weekday", s.withTelegramAuth(http.HandlerFunc(s.handleWeekday)))

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
	return http.FileServer(http.FS(sub))
}

func (s *Server) withTelegramAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(s.botToken) == "" {
			writeError(w, http.StatusServiceUnavailable, "TELEGRAM_BOT_TOKEN is not configured")
			return
		}

		initData := strings.TrimSpace(r.Header.Get("X-Telegram-Init-Data"))
		if initData == "" {
			writeError(w, http.StatusUnauthorized, "missing telegram init data")
			return
		}

		if err := validateTelegramInitData(initData, s.botToken); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid telegram init data")
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
	yMetric := parseYMetric(r)

	u, status, err := s.resolveUser(r)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}

	reports, err := s.reportsByDays(u.PrimaryStorageID(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reports")
		return
	}

	type reportWithWellness struct {
		*daily_report.DailyReport
		WellnessIndex float64 `json:"wellnessIndex"`
		YValue        float64 `json:"yValue"`
	}

	payload := make([]reportWithWellness, 0, len(reports))
	for _, item := range reports {
		payload = append(payload, reportWithWellness{
			DailyReport:   item,
			WellnessIndex: analytics.WellnessIndex(item),
			YValue:        yMetric.Extract(item),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"days":          days,
		"user":          u,
		"reports":       payload,
		"totalReports":  len(payload),
		"generatedAt":   time.Now().UTC(),
		"requestedDays": days,
		"yAxis": map[string]interface{}{
			"field": yMetric.Field,
			"label": yMetric.Label,
			"min":   yMetric.Min,
			"max":   yMetric.Max,
		},
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

	reports, err := s.reportsByDays(u.PrimaryStorageID(), days)
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

	reports, err := s.reportsByDays(u.PrimaryStorageID(), days)
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

	reports, err := s.reportsByDays(u.PrimaryStorageID(), days)
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
	metric, err := parseWeekdayMetric(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	u, status, err := s.resolveUser(r)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}

	reports, err := s.reportsByDays(u.PrimaryStorageID(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reports")
		return
	}

	weekdayRaw := analytics.ByWeekday(reports)
	weekday := make([]map[string]interface{}, 0, len(weekdayRaw))
	for _, day := range weekdayRaw {
		weekday = append(weekday, map[string]interface{}{
			"day":   day.Day,
			"value": weekdayMetricValue(day, metric),
			"count": day.Count,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"days":         days,
		"user":         u,
		"reportsCount": len(reports),
		"metric":       metric,
		"weekday":      weekday,
		"generatedAt":  time.Now().UTC(),
	})
}

func (s *Server) resolveUser(r *http.Request) (*user.User, int, error) {
	initData := strings.TrimSpace(r.Header.Get("X-Telegram-Init-Data"))
	if initData == "" {
		return nil, http.StatusUnauthorized, fmt.Errorf("missing telegram init data")
	}

	tgUser, err := telegramUserFromInitData(initData)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("failed to parse telegram user")
	}
	if strings.TrimSpace(tgUser.Username) == "" {
		return nil, http.StatusForbidden, fmt.Errorf("telegram username is required")
	}

	u, err := s.userRepo.FindByUsername(s.ctx, tgUser.Username)
	if err != nil {
		return nil, http.StatusNotFound, fmt.Errorf("user with username=%q not found", tgUser.Username)
	}
	return u, http.StatusOK, nil
}

func (s *Server) reportsByDays(userID string, days int) ([]*daily_report.DailyReport, error) {
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

type yMetricSpec struct {
	Field   string
	Label   string
	Min     float64
	Max     float64
	Extract func(*daily_report.DailyReport) float64
}

var yMetricByField = map[string]yMetricSpec{
	"wellnessIndex": {
		Field: "wellnessIndex",
		Label: "Wellness Index",
		Min:   0,
		Max:   10,
		Extract: func(r *daily_report.DailyReport) float64 {
			return analytics.WellnessIndex(r)
		},
	},
	"mood": {
		Field: "mood",
		Label: "Mood",
		Min:   0,
		Max:   10,
		Extract: func(r *daily_report.DailyReport) float64 {
			return float64(r.Mood)
		},
	},
	"anxiety": {
		Field: "anxiety",
		Label: "Anxiety",
		Min:   0,
		Max:   5,
		Extract: func(r *daily_report.DailyReport) float64 {
			return float64(r.Anxiety)
		},
	},
	"energy": {
		Field: "energy",
		Label: "Energy",
		Min:   0,
		Max:   5,
		Extract: func(r *daily_report.DailyReport) float64 {
			return float64(r.Energy)
		},
	},
	"sleepQuality": {
		Field: "sleepQuality",
		Label: "Sleep Quality",
		Min:   0,
		Max:   5,
		Extract: func(r *daily_report.DailyReport) float64 {
			return float64(r.SleepQuality)
		},
	},
	"migraine": {
		Field: "migraine",
		Label: "Migraine",
		Min:   0,
		Max:   5,
		Extract: func(r *daily_report.DailyReport) float64 {
			return float64(r.Migraine)
		},
	},
	"libido": {
		Field: "libido",
		Label: "Libido",
		Min:   0,
		Max:   5,
		Extract: func(r *daily_report.DailyReport) float64 {
			return float64(r.Libido)
		},
	},
	"relationship": {
		Field: "relationship",
		Label: "Relationship",
		Min:   0,
		Max:   5,
		Extract: func(r *daily_report.DailyReport) float64 {
			return float64(r.Relationship)
		},
	},
	"closeness": {
		Field: "closeness",
		Label: "Closeness",
		Min:   0,
		Max:   5,
		Extract: func(r *daily_report.DailyReport) float64 {
			return float64(r.Closeness)
		},
	},
}

func parseYMetric(r *http.Request) yMetricSpec {
	requested := strings.TrimSpace(r.URL.Query().Get("y_field"))
	if requested == "" {
		requested = "wellnessIndex"
	}
	if metric, ok := yMetricByField[requested]; ok {
		return metric
	}
	return yMetricByField["wellnessIndex"]
}

func parseWeekdayMetric(r *http.Request) (string, error) {
	metric := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("metric")))
	if metric == "" {
		return "mood", nil
	}

	switch metric {
	case "mood", "anxiety", "energy", "migraine", "sleep_quality":
		return metric, nil
	default:
		return "", fmt.Errorf("metric must be one of: mood, anxiety, energy, migraine, sleep_quality")
	}
}

func weekdayMetricValue(day analytics.WeekdayStats, metric string) float64 {
	switch metric {
	case "anxiety":
		return day.AvgAnxiety
	case "energy":
		return day.AvgEnergy
	case "migraine":
		return day.AvgMigraine
	case "sleep_quality":
		return day.AvgSleepQuality
	default:
		return day.AvgMood
	}
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

type telegramInitUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

func telegramUserFromInitData(initData string) (*telegramInitUser, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, err
	}
	userRaw := strings.TrimSpace(values.Get("user"))
	if userRaw == "" {
		return nil, fmt.Errorf("missing user in init data")
	}

	var u telegramInitUser
	if err := json.Unmarshal([]byte(userRaw), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func validateTelegramInitData(initData, botToken string) error {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return err
	}

	receivedHash := strings.TrimSpace(values.Get("hash"))
	if receivedHash == "" {
		return fmt.Errorf("missing hash")
	}

	dataPairs := make([]string, 0, len(values))
	for key, value := range values {
		if key == "hash" || len(value) == 0 {
			continue
		}
		dataPairs = append(dataPairs, key+"="+value[0])
	}
	sort.Strings(dataPairs)
	dataCheckString := strings.Join(dataPairs, "\n")

	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	secretMAC.Write([]byte(botToken))
	secretKey := secretMAC.Sum(nil)

	signMAC := hmac.New(sha256.New, secretKey)
	signMAC.Write([]byte(dataCheckString))
	calculatedHash := fmt.Sprintf("%x", signMAC.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(calculatedHash), []byte(receivedHash)) != 1 {
		return fmt.Errorf("hash mismatch")
	}

	return nil
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
