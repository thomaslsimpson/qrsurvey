package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/thomaslsimpson/qrsurvey/internal/config"
	"github.com/thomaslsimpson/qrsurvey/internal/db"
	adminh "github.com/thomaslsimpson/qrsurvey/internal/handlers/admin"
	publich "github.com/thomaslsimpson/qrsurvey/internal/handlers/public"
	"github.com/thomaslsimpson/qrsurvey/internal/middleware"
	"github.com/thomaslsimpson/qrsurvey/internal/web"
)

func NewRouter(cfg config.Config, database *db.DB, logger *slog.Logger) http.Handler {
	pub := &publich.Handlers{DB: database, Logger: logger, HashSecret: cfg.HashSecret}
	adm := &adminh.Handlers{DB: database, Logger: logger, BaseURL: cfg.BaseURL, QRCacheDir: cfg.QRCacheDir, HashSecret: cfg.HashSecret}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mux.Handle("GET /static/", web.StaticHandler())

	// Public survey flow — unauthenticated, reached by scanning a poster's
	// QR code. The submit endpoint is rate-limited by IP since it's the
	// only public route that writes to the database.
	mux.HandleFunc("GET /p/{posterID}", pub.Scan)
	submitLimiter := middleware.NewIPRateLimiter(5, time.Hour)
	mux.Handle("POST /p/{posterID}/submit", submitLimiter.Middleware(http.HandlerFunc(pub.Submit)))

	// Direct-entry (alternate method of entry) — non-guessable per-poster
	// link that skips the survey. Rate-limited on both GET and POST: GET
	// because an attacker could otherwise brute-force the 8-hex-char hash
	// by sheer request volume, POST for the same reason submit is limited.
	directLimiter := middleware.NewIPRateLimiter(5, time.Hour)
	mux.Handle("GET /e/{posterID}/{hash}", directLimiter.Middleware(http.HandlerFunc(pub.DirectEntry)))
	mux.Handle("POST /e/{posterID}/{hash}/submit", directLimiter.Middleware(http.HandlerFunc(pub.DirectEntrySubmit)))

	// Admin/back office — single-operator tool behind HTTP Basic Auth.
	adminMux := http.NewServeMux()
	redirectToSurveys := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/surveys", http.StatusSeeOther)
	}
	adminMux.HandleFunc("GET /admin", redirectToSurveys)
	adminMux.HandleFunc("GET /admin/", redirectToSurveys)
	adminMux.HandleFunc("GET /admin/surveys", adm.ListSurveys)
	adminMux.HandleFunc("POST /admin/surveys", adm.CreateSurvey)
	adminMux.HandleFunc("GET /admin/surveys/{id}", adm.SurveyDetail)
	adminMux.HandleFunc("POST /admin/surveys/{id}", adm.UpdateSurvey)
	adminMux.HandleFunc("POST /admin/surveys/{id}/items", adm.CreateSurveyItem)
	adminMux.HandleFunc("POST /admin/surveys/{id}/items/{itemID}/delete", adm.DeleteSurveyItem)
	adminMux.HandleFunc("GET /admin/contests", adm.ListContests)
	adminMux.HandleFunc("POST /admin/contests", adm.CreateContest)
	adminMux.HandleFunc("GET /admin/contests/{id}", adm.ContestDetail)
	adminMux.HandleFunc("POST /admin/contests/{id}", adm.UpdateContest)
	adminMux.HandleFunc("POST /admin/contests/{id}/posters", adm.CreatePoster)
	adminMux.HandleFunc("GET /admin/contests/{id}/contestants.csv", adm.ContestantsCSV)
	adminMux.HandleFunc("GET /admin/posters/{id}/qrcode.png", adm.PosterQRCode)
	adminMux.HandleFunc("GET /admin/posters/{id}/poster.pdf", adm.PosterPDF)

	mux.Handle("/admin/", adminh.BasicAuth(cfg.AdminUser, cfg.AdminPassHash)(adminMux))

	return middleware.Chain(mux, middleware.Recover(logger), middleware.Logging(logger))
}
