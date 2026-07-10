package public

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thomaslsimpson/qrsurvey/internal/db"
	"github.com/thomaslsimpson/qrsurvey/internal/models"
)

type submitAnswer struct {
	SurveyItemID  int64 `json:"survey_item_id"`
	ValueSelected int   `json:"value_selected"`
}

type submitRequest struct {
	Name     string         `json:"name"`
	Email    string         `json:"email"`
	Phone    string         `json:"phone"`
	Address  string         `json:"address"`
	Honeypot string         `json:"honeypot"`
	Answers  []submitAnswer `json:"answers"`
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *Handlers) Submit(w http.ResponseWriter, r *http.Request) {
	posterID, err := strconv.ParseInt(r.PathValue("posterID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var req submitRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "malformed request")
		return
	}

	// Honeypot: real visitors never see or fill this field. Bots that
	// blindly fill every input get a fake success with no DB write, so we
	// don't tip them off that they were caught.
	if req.Honeypot != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

	// Re-resolve poster -> contest -> survey server-side; never trust
	// contest/survey IDs from the client.
	bundle, err := h.DB.ResolveSurveyBundle(r.Context(), posterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		h.Logger.Error("resolve survey bundle", "err", err, "poster_id", posterID)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if end, err := parseEndDate(bundle.Contest.EndDate); err == nil && time.Now().UTC().After(end) {
		writeJSONError(w, http.StatusGone, "Sorry, this contest has already ended.")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Name == "" || req.Phone == "" {
		writeJSONError(w, http.StatusBadRequest, "Name and mobile number are required.")
		return
	}

	expected := make(map[int64]bool, len(bundle.Items))
	for _, item := range bundle.Items {
		expected[item.ID] = true
	}
	if len(req.Answers) != len(expected) {
		writeJSONError(w, http.StatusBadRequest, "Please answer every question.")
		return
	}
	seen := make(map[int64]bool, len(req.Answers))
	answers := make([]db.SubmitAnswer, 0, len(req.Answers))
	for _, a := range req.Answers {
		if !expected[a.SurveyItemID] || seen[a.SurveyItemID] {
			writeJSONError(w, http.StatusBadRequest, "Please answer every question.")
			return
		}
		if a.ValueSelected < 1 || a.ValueSelected > 5 {
			writeJSONError(w, http.StatusBadRequest, "Invalid answer value.")
			return
		}
		seen[a.SurveyItemID] = true
		answers = append(answers, db.SubmitAnswer{SurveyItemID: a.SurveyItemID, ValueSelected: a.ValueSelected})
	}

	contestant := contestantFromRequest(req)
	_, err = h.DB.SubmitEntry(r.Context(), bundle.Contest.ID, bundle.Poster.ID, contestant, answers)
	if err != nil {
		if errors.Is(err, db.ErrDuplicateEntry) {
			writeJSONError(w, http.StatusConflict, "Looks like you've already entered this contest with that mobile number.")
			return
		}
		h.Logger.Error("submit entry", "err", err, "poster_id", posterID)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func contestantFromRequest(req submitRequest) models.Contestant {
	return models.Contestant{
		Name:    req.Name,
		Email:   req.Email,
		Phone:   req.Phone,
		Address: strings.TrimSpace(req.Address),
	}
}
