package public

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thomaslsimpson/qrsurvey/internal/db"
	"github.com/thomaslsimpson/qrsurvey/internal/models"
)

func newTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	database, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &Handlers{DB: database, Logger: logger}
}

// seed creates a 2-question survey + contest + poster; endDate lets tests
// exercise the expired-contest path.
func seed(t *testing.T, h *Handlers, endDate string) (posterID int64, itemIDs []int64) {
	t.Helper()
	ctx := context.Background()
	surveyID, err := h.DB.CreateSurvey(ctx, "Test Mall")
	if err != nil {
		t.Fatalf("create survey: %v", err)
	}
	for i := 0; i < 2; i++ {
		id, err := h.DB.CreateSurveyItem(ctx, models.SurveyItem{
			SurveyID: surveyID, Question: "Q?",
			Response1: "1", Response2: "2", Response3: "3", Response4: "4", Response5: "5",
		})
		if err != nil {
			t.Fatalf("create item: %v", err)
		}
		itemIDs = append(itemIDs, id)
	}
	contestID, err := h.DB.CreateContest(ctx, surveyID, endDate, "Prize")
	if err != nil {
		t.Fatalf("create contest: %v", err)
	}
	posterID, err = h.DB.CreatePoster(ctx, contestID, "Poster A")
	if err != nil {
		t.Fatalf("create poster: %v", err)
	}
	return posterID, itemIDs
}

func doSubmit(t *testing.T, h *Handlers, posterID int64, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/p/%d/submit", posterID), bytes.NewReader(b))
	req.SetPathValue("posterID", fmt.Sprintf("%d", posterID))
	rec := httptest.NewRecorder()
	h.Submit(rec, req)
	return rec
}

func validPayload(itemIDs []int64) map[string]any {
	return map[string]any{
		"name":  "Jane Doe",
		"email": "jane@example.com",
		"phone": "555-0100",
		"answers": []map[string]any{
			{"survey_item_id": itemIDs[0], "value_selected": 4},
			{"survey_item_id": itemIDs[1], "value_selected": 2},
		},
	}
}

func TestSubmit_HappyPath(t *testing.T) {
	h := newTestHandlers(t)
	posterID, itemIDs := seed(t, h, "2999-12-31T23:59:59Z")

	rec := doSubmit(t, h, posterID, validPayload(itemIDs))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestSubmit_MissingAnswer(t *testing.T) {
	h := newTestHandlers(t)
	posterID, itemIDs := seed(t, h, "2999-12-31T23:59:59Z")

	payload := validPayload(itemIDs)
	payload["answers"] = []map[string]any{{"survey_item_id": itemIDs[0], "value_selected": 4}} // missing 2nd

	rec := doSubmit(t, h, posterID, payload)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestSubmit_UnknownItemID(t *testing.T) {
	h := newTestHandlers(t)
	posterID, itemIDs := seed(t, h, "2999-12-31T23:59:59Z")

	payload := validPayload(itemIDs)
	payload["answers"] = []map[string]any{
		{"survey_item_id": itemIDs[0], "value_selected": 4},
		{"survey_item_id": 999999, "value_selected": 2},
	}

	rec := doSubmit(t, h, posterID, payload)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestSubmit_OutOfRangeValue(t *testing.T) {
	h := newTestHandlers(t)
	posterID, itemIDs := seed(t, h, "2999-12-31T23:59:59Z")

	payload := validPayload(itemIDs)
	payload["answers"] = []map[string]any{
		{"survey_item_id": itemIDs[0], "value_selected": 4},
		{"survey_item_id": itemIDs[1], "value_selected": 7},
	}

	rec := doSubmit(t, h, posterID, payload)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestSubmit_MissingNameOrPhone(t *testing.T) {
	h := newTestHandlers(t)
	posterID, itemIDs := seed(t, h, "2999-12-31T23:59:59Z")

	payload := validPayload(itemIDs)
	payload["phone"] = ""

	rec := doSubmit(t, h, posterID, payload)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestSubmit_ExpiredContest(t *testing.T) {
	h := newTestHandlers(t)
	posterID, itemIDs := seed(t, h, "2000-01-01T23:59:59Z")

	rec := doSubmit(t, h, posterID, validPayload(itemIDs))
	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410, body = %s", rec.Code, rec.Body.String())
	}
}

func TestSubmit_Honeypot(t *testing.T) {
	h := newTestHandlers(t)
	posterID, itemIDs := seed(t, h, "2999-12-31T23:59:59Z")

	payload := validPayload(itemIDs)
	payload["honeypot"] = "i-am-a-bot"

	rec := doSubmit(t, h, posterID, payload)
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

func TestSubmit_DuplicatePhone(t *testing.T) {
	h := newTestHandlers(t)
	posterID, itemIDs := seed(t, h, "2999-12-31T23:59:59Z")

	first := doSubmit(t, h, posterID, validPayload(itemIDs))
	if first.Code != http.StatusCreated {
		t.Fatalf("first submit status = %d, body = %s", first.Code, first.Body.String())
	}

	second := doSubmit(t, h, posterID, validPayload(itemIDs))
	if second.Code != http.StatusConflict {
		t.Fatalf("second submit status = %d, want 409, body = %s", second.Code, second.Body.String())
	}
}
