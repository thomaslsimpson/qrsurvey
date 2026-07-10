package public

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thomaslsimpson/qrsurvey/internal/entryhash"
)

func newDirectEntryRequest(t *testing.T, method string, posterID int64, hash, path string, body map[string]any) *http.Request {
	t.Helper()
	var r *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.SetPathValue("posterID", fmt.Sprintf("%d", posterID))
	r.SetPathValue("hash", hash)
	return r
}

func TestDirectEntry_ValidHash(t *testing.T) {
	h := newTestHandlers(t)
	posterID, _ := seed(t, h, "2999-12-31T23:59:59Z")
	hash := entryhash.Hash(h.HashSecret, posterID)

	rec := httptest.NewRecorder()
	h.DirectEntry(rec, newDirectEntryRequest(t, http.MethodGet, posterID, hash, fmt.Sprintf("/e/%d/%s", posterID, hash), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Enter to win")) {
		t.Errorf("expected direct-entry page content, got: %s", rec.Body.String())
	}
}

func TestDirectEntry_WrongHash(t *testing.T) {
	h := newTestHandlers(t)
	posterID, _ := seed(t, h, "2999-12-31T23:59:59Z")

	rec := httptest.NewRecorder()
	h.DirectEntry(rec, newDirectEntryRequest(t, http.MethodGet, posterID, "deadbeef", fmt.Sprintf("/e/%d/deadbeef", posterID), nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDirectEntry_UnknownPosterSameShapeAsWrongHash(t *testing.T) {
	h := newTestHandlers(t)
	// A hash that's well-formed but for a poster ID that doesn't exist
	// should 404 identically to a wrong hash for a real poster — no
	// distinguishing information leaked either way.
	hash := entryhash.Hash(h.HashSecret, 999999)

	rec := httptest.NewRecorder()
	h.DirectEntry(rec, newDirectEntryRequest(t, http.MethodGet, 999999, hash, "/e/999999/"+hash, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDirectEntry_ExpiredContest(t *testing.T) {
	h := newTestHandlers(t)
	posterID, _ := seed(t, h, "2000-01-01T23:59:59Z")
	hash := entryhash.Hash(h.HashSecret, posterID)

	rec := httptest.NewRecorder()
	h.DirectEntry(rec, newDirectEntryRequest(t, http.MethodGet, posterID, hash, fmt.Sprintf("/e/%d/%s", posterID, hash), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("ended")) {
		t.Errorf("expected the 'contest ended' page, got: %s", rec.Body.String())
	}
}

func doDirectSubmit(t *testing.T, h *Handlers, posterID int64, hash string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	req := newDirectEntryRequest(t, http.MethodPost, posterID, hash, fmt.Sprintf("/e/%d/%s/submit", posterID, hash), body)
	rec := httptest.NewRecorder()
	h.DirectEntrySubmit(rec, req)
	return rec
}

func TestDirectEntrySubmit_HappyPath(t *testing.T) {
	h := newTestHandlers(t)
	posterID, _ := seed(t, h, "2999-12-31T23:59:59Z")
	hash := entryhash.Hash(h.HashSecret, posterID)

	rec := doDirectSubmit(t, h, posterID, hash, map[string]any{
		"name": "Jane Doe", "email": "jane@example.com", "phone": "555-0300",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	contestants, err := h.DB.ListContestantsByContest(context.Background(), 1)
	if err != nil {
		t.Fatalf("list contestants: %v", err)
	}
	if len(contestants) != 1 {
		t.Fatalf("contestants = %d, want 1", len(contestants))
	}
	if contestants[0].PosterID != posterID {
		t.Errorf("PosterID = %d, want %d", contestants[0].PosterID, posterID)
	}
}

func TestDirectEntrySubmit_WrongHash(t *testing.T) {
	h := newTestHandlers(t)
	posterID, _ := seed(t, h, "2999-12-31T23:59:59Z")

	rec := doDirectSubmit(t, h, posterID, "deadbeef", map[string]any{"name": "Jane Doe", "phone": "555-0300"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDirectEntrySubmit_MissingNameOrPhone(t *testing.T) {
	h := newTestHandlers(t)
	posterID, _ := seed(t, h, "2999-12-31T23:59:59Z")
	hash := entryhash.Hash(h.HashSecret, posterID)

	rec := doDirectSubmit(t, h, posterID, hash, map[string]any{"name": "", "phone": "555-0300"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDirectEntrySubmit_ExpiredContest(t *testing.T) {
	h := newTestHandlers(t)
	posterID, _ := seed(t, h, "2000-01-01T23:59:59Z")
	hash := entryhash.Hash(h.HashSecret, posterID)

	rec := doDirectSubmit(t, h, posterID, hash, map[string]any{"name": "Jane Doe", "phone": "555-0300"})
	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDirectEntrySubmit_Honeypot(t *testing.T) {
	h := newTestHandlers(t)
	posterID, _ := seed(t, h, "2999-12-31T23:59:59Z")
	hash := entryhash.Hash(h.HashSecret, posterID)

	rec := doDirectSubmit(t, h, posterID, hash, map[string]any{
		"name": "Bot", "phone": "555-9999", "honeypot": "gotcha",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (fake success), body = %s", rec.Code, rec.Body.String())
	}
	contestants, err := h.DB.ListContestantsByContest(context.Background(), 1)
	if err != nil {
		t.Fatalf("list contestants: %v", err)
	}
	if len(contestants) != 0 {
		t.Errorf("expected no contestant rows written for a honeypot submission, got %d", len(contestants))
	}
}

func TestDirectEntrySubmit_DuplicatePhone(t *testing.T) {
	h := newTestHandlers(t)
	posterID, _ := seed(t, h, "2999-12-31T23:59:59Z")
	hash := entryhash.Hash(h.HashSecret, posterID)

	first := doDirectSubmit(t, h, posterID, hash, map[string]any{"name": "Jane Doe", "phone": "555-0300"})
	if first.Code != http.StatusCreated {
		t.Fatalf("first submit status = %d, body = %s", first.Code, first.Body.String())
	}
	second := doDirectSubmit(t, h, posterID, hash, map[string]any{"name": "Jane Doe", "phone": "555-0300"})
	if second.Code != http.StatusConflict {
		t.Fatalf("second submit status = %d, want 409, body = %s", second.Code, second.Body.String())
	}
}
