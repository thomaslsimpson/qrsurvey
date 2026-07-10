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
	"github.com/thomaslsimpson/qrsurvey/internal/entryhash"
	"github.com/thomaslsimpson/qrsurvey/internal/web"
)

// pathPosterAndHash parses and verifies the {posterID}/{hash} path values
// shared by DirectEntry and DirectEntrySubmit. On any failure it writes the
// response itself (always a plain 404 — an unknown poster ID and a wrong
// hash for a real poster ID must be indistinguishable) and returns ok=false.
func (h *Handlers) pathPosterAndHash(w http.ResponseWriter, r *http.Request) (posterID int64, ok bool) {
	posterID, err := strconv.ParseInt(r.PathValue("posterID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return 0, false
	}
	if !entryhash.Verify(h.HashSecret, posterID, r.PathValue("hash")) {
		http.NotFound(w, r)
		return 0, false
	}
	return posterID, true
}

// DirectEntry serves the "alternate method of entry" page: a non-guessable,
// per-poster link straight to the contest-entry form, skipping the survey
// entirely. Required for a NO-PURCHASE-NECESSARY sweepstakes, where
// completing the survey must not be a condition of entry.
func (h *Handlers) DirectEntry(w http.ResponseWriter, r *http.Request) {
	posterID, ok := h.pathPosterAndHash(w, r)
	if !ok {
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := web.RenderDirectEntry(w, web.DirectEntryData{BusinessName: bundle.Survey.Description}); err != nil {
		h.Logger.Error("render direct_entry", "err", err)
	}
}

// DirectEntrySubmit records a contest entry with no survey answers. It
// deliberately does not accept an "answers" field at all — this path exists
// specifically to make survey completion optional.
func (h *Handlers) DirectEntrySubmit(w http.ResponseWriter, r *http.Request) {
	posterID, ok := h.pathPosterAndHash(w, r)
	if !ok {
		return
	}

	var req submitRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "malformed request")
		return
	}

	if req.Honeypot != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

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

	contestant := contestantFromRequest(req)
	_, err = h.DB.SubmitEntry(r.Context(), bundle.Contest.ID, bundle.Poster.ID, contestant, nil)
	if err != nil {
		if errors.Is(err, db.ErrDuplicateEntry) {
			writeJSONError(w, http.StatusConflict, "Looks like you've already entered this contest with that mobile number.")
			return
		}
		h.Logger.Error("submit direct entry", "err", err, "poster_id", posterID)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
