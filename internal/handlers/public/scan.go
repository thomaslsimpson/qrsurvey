// Package public holds the unauthenticated handlers a customer hits by
// scanning a poster's QR code: viewing the survey wizard and submitting it.
package public

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/thomaslsimpson/qrsurvey/internal/db"
	"github.com/thomaslsimpson/qrsurvey/internal/models"
	"github.com/thomaslsimpson/qrsurvey/internal/web"
)

type Handlers struct {
	DB         *db.DB
	Logger     *slog.Logger
	HashSecret string
}

// EndDateLayout is the format contest.end_date is stored in: an RFC3339
// UTC timestamp treated as an inclusive end-of-day boundary (admin CRUD
// appends "T23:59:59Z" to whatever calendar date is picked).
const EndDateLayout = time.RFC3339

func parseEndDate(s string) (time.Time, error) {
	return time.Parse(EndDateLayout, s)
}

func (h *Handlers) Scan(w http.ResponseWriter, r *http.Request) {
	posterID, err := strconv.ParseInt(r.PathValue("posterID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	bundle, err := h.DB.ResolveSurveyBundle(r.Context(), posterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		h.Logger.Error("resolve survey bundle", "err", err, "poster_id", posterID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if end, err := parseEndDate(bundle.Contest.EndDate); err == nil && time.Now().UTC().After(end) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := web.RenderEnded(w, bundle.Survey.Description); err != nil {
			h.Logger.Error("render ended", "err", err)
		}
		return
	}

	if len(bundle.Items) == 0 {
		h.Logger.Error("survey has zero items — refusing to render wizard", "survey_id", bundle.Survey.ID, "poster_id", posterID)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := web.RenderNotReady(w); err != nil {
			h.Logger.Error("render not_ready", "err", err)
		}
		return
	}

	data := buildWizardData(bundle)
	dataJSON, err := web.MarshalWizardJSON(wizardJSONPayload(bundle))
	if err != nil {
		h.Logger.Error("marshal wizard json", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	data.DataJSON = dataJSON

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := web.RenderWizard(w, data); err != nil {
		h.Logger.Error("render wizard", "err", err)
	}
}

func buildWizardData(b models.SurveyBundle) web.WizardData {
	return web.WizardData{
		BusinessName: b.Survey.Description,
		PosterID:     b.Poster.ID,
		ContestID:    b.Contest.ID,
		ItemCount:    len(b.Items),
	}
}

type jsonItem struct {
	ID        int64     `json:"id"`
	Question  string    `json:"question"`
	Responses [5]string `json:"responses"`
}

type jsonPayload struct {
	PosterID  int64      `json:"posterId"`
	ContestID int64      `json:"contestId"`
	Items     []jsonItem `json:"items"`
}

// wizardJSONPayload is the data the client-side wizard (wizard.js) builds
// every screen from; the "kicker" text ("Question N of M") is computed
// client-side from the items array's length and index, not sent here.
func wizardJSONPayload(b models.SurveyBundle) jsonPayload {
	items := make([]jsonItem, len(b.Items))
	for i, it := range b.Items {
		items[i] = jsonItem{ID: it.ID, Question: it.Question, Responses: it.Responses()}
	}
	return jsonPayload{PosterID: b.Poster.ID, ContestID: b.Contest.ID, Items: items}
}
