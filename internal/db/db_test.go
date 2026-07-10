package db

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/thomaslsimpson/qrsurvey/internal/models"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	ctx := context.Background()
	// A unique named in-memory database per test avoids cross-test bleed
	// while still exercising the same file:-DSN + pragma path as production.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	database, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return database
}

// seedSurveyWithItems creates a survey with n questions, a contest ending
// far in the future, and one poster for that contest.
func seedSurveyWithItems(t *testing.T, d *DB, n int) (surveyID, contestID, posterID int64, itemIDs []int64) {
	t.Helper()
	ctx := context.Background()

	surveyID, err := d.CreateSurvey(ctx, "Test Business")
	if err != nil {
		t.Fatalf("create survey: %v", err)
	}
	for i := 0; i < n; i++ {
		id, err := d.CreateSurveyItem(ctx, models.SurveyItem{
			SurveyID: surveyID, Question: fmt.Sprintf("Question %d?", i),
			Response1: "Poor", Response2: "Meh", Response3: "OK", Response4: "Good", Response5: "Great",
			SortOrder: i,
		})
		if err != nil {
			t.Fatalf("create survey item: %v", err)
		}
		itemIDs = append(itemIDs, id)
	}
	contestID, err = d.CreateContest(ctx, surveyID, "2999-12-31T23:59:59Z", "A Prize")
	if err != nil {
		t.Fatalf("create contest: %v", err)
	}
	posterID, err = d.CreatePoster(ctx, contestID, "Test Poster")
	if err != nil {
		t.Fatalf("create poster: %v", err)
	}
	return surveyID, contestID, posterID, itemIDs
}

func countRows(t *testing.T, d *DB, table string) int {
	t.Helper()
	var n int
	if err := d.conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestSubmitEntry_HappyPath(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	_, contestID, posterID, itemIDs := seedSurveyWithItems(t, d, 2)

	contestantID, err := d.SubmitEntry(ctx, contestID, posterID,
		models.Contestant{Name: "Jane Doe", Email: "jane@example.com", Phone: "555-0100"},
		[]SubmitAnswer{
			{SurveyItemID: itemIDs[0], ValueSelected: 4},
			{SurveyItemID: itemIDs[1], ValueSelected: 2},
		})
	if err != nil {
		t.Fatalf("SubmitEntry: %v", err)
	}
	if contestantID == 0 {
		t.Fatalf("expected nonzero contestant id")
	}
	if got := countRows(t, d, "contestant"); got != 1 {
		t.Errorf("contestant rows = %d, want 1", got)
	}
	if got := countRows(t, d, "answer"); got != 2 {
		t.Errorf("answer rows = %d, want 2", got)
	}
}

// TestSubmitEntry_ZeroAnswers covers the direct-entry (AMOE) path: a
// contestant who skips the survey entirely still gets recorded, with
// poster_id set so there's a record of which link they used, and zero
// answer rows.
func TestSubmitEntry_ZeroAnswers(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	_, contestID, posterID, _ := seedSurveyWithItems(t, d, 2)

	contestantID, err := d.SubmitEntry(ctx, contestID, posterID,
		models.Contestant{Name: "Direct Entrant", Phone: "555-0200"}, nil)
	if err != nil {
		t.Fatalf("SubmitEntry: %v", err)
	}
	if got := countRows(t, d, "answer"); got != 0 {
		t.Errorf("answer rows = %d, want 0", got)
	}

	contestants, err := d.ListContestantsByContest(ctx, contestID)
	if err != nil {
		t.Fatalf("ListContestantsByContest: %v", err)
	}
	var found bool
	for _, c := range contestants {
		if c.ID == contestantID {
			found = true
			if c.PosterID != posterID {
				t.Errorf("PosterID = %d, want %d", c.PosterID, posterID)
			}
		}
	}
	if !found {
		t.Fatalf("contestant %d not found in ListContestantsByContest", contestantID)
	}
}

// TestSubmitEntry_Atomicity is the highest-value test in the suite: the
// product requirement (issue #2) is that a survey submission is all-or-
// nothing. A bad answer partway through the batch must roll back the
// contestant row too, not just stop short.
func TestSubmitEntry_Atomicity(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	_, contestID, posterID, itemIDs := seedSurveyWithItems(t, d, 2)

	const nonexistentItemID = 999999
	_, err := d.SubmitEntry(ctx, contestID, posterID,
		models.Contestant{Name: "Jane Doe", Phone: "555-0100"},
		[]SubmitAnswer{
			{SurveyItemID: itemIDs[0], ValueSelected: 4},
			{SurveyItemID: nonexistentItemID, ValueSelected: 2}, // violates FK, fails mid-batch
		})
	if err == nil {
		t.Fatal("expected an error from an invalid survey_item_id, got nil")
	}

	if got := countRows(t, d, "contestant"); got != 0 {
		t.Errorf("contestant rows = %d, want 0 (transaction should have rolled back)", got)
	}
	if got := countRows(t, d, "answer"); got != 0 {
		t.Errorf("answer rows = %d, want 0 (transaction should have rolled back)", got)
	}
}

func TestSubmitEntry_DuplicatePhone(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	_, contestID, posterID, itemIDs := seedSurveyWithItems(t, d, 1)

	answers := []SubmitAnswer{{SurveyItemID: itemIDs[0], ValueSelected: 3}}
	contestant := models.Contestant{Name: "Jane Doe", Phone: "555-0100"}

	if _, err := d.SubmitEntry(ctx, contestID, posterID, contestant, answers); err != nil {
		t.Fatalf("first SubmitEntry: %v", err)
	}

	_, err := d.SubmitEntry(ctx, contestID, posterID, contestant, answers)
	if !errors.Is(err, ErrDuplicateEntry) {
		t.Fatalf("second SubmitEntry error = %v, want ErrDuplicateEntry", err)
	}
	if got := countRows(t, d, "contestant"); got != 1 {
		t.Errorf("contestant rows = %d, want 1 (duplicate attempt must not add a row)", got)
	}
	if got := countRows(t, d, "answer"); got != 1 {
		t.Errorf("answer rows = %d, want 1 (duplicate attempt must not add a row)", got)
	}
}

func TestConstraints_ValueSelectedOutOfRange(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	_, contestID, posterID, itemIDs := seedSurveyWithItems(t, d, 1)

	_, err := d.SubmitEntry(ctx, contestID, posterID,
		models.Contestant{Name: "Jane Doe", Phone: "555-0100"},
		[]SubmitAnswer{{SurveyItemID: itemIDs[0], ValueSelected: 6}})
	if err == nil {
		t.Fatal("expected CHECK constraint violation for value_selected=6, got nil")
	}
	if got := countRows(t, d, "contestant"); got != 0 {
		t.Errorf("contestant rows = %d, want 0", got)
	}
}

func TestConstraints_PosterRequiresExistingContest(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	if _, err := d.CreatePoster(ctx, 999999, "orphan poster"); err == nil {
		t.Fatal("expected FK violation creating a poster under a nonexistent contest, got nil")
	}
}
