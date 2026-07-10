package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/thomaslsimpson/qrsurvey/internal/models"
)

var ErrDuplicateEntry = errors.New("contestant already entered this contest")

// SubmitAnswer is one answer to be inserted alongside a new contestant.
type SubmitAnswer struct {
	SurveyItemID  int64
	ValueSelected int
}

// SubmitEntry inserts one contestant row plus one answer row per element of
// answers, all in a single transaction, matching the product requirement
// that the whole survey be recorded atomically once contact info is known.
// A duplicate (contest_id, phone) returns ErrDuplicateEntry after rolling
// back, so a caller can respond with a friendly "already entered" message.
func (d *DB) SubmitEntry(ctx context.Context, contestID, posterID int64, contestant models.Contestant, answers []SubmitAnswer) (int64, error) {
	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op if already committed

	res, err := tx.ExecContext(ctx, `
		INSERT INTO contestant (contest_id, name, email, phone, address) VALUES (?, ?, ?, ?, ?)`,
		contestID, contestant.Name, nullIfEmpty(contestant.Email), contestant.Phone, nullIfEmpty(contestant.Address))
	if err != nil {
		if isUniqueConstraintErr(err) {
			return 0, ErrDuplicateEntry
		}
		return 0, fmt.Errorf("insert contestant: %w", err)
	}
	contestantID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("contestant last insert id: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO answer (contestant_id, poster_id, contest_id, survey_item_id, value_selected)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare answer insert: %w", err)
	}
	defer stmt.Close()

	for _, a := range answers {
		if _, err := stmt.ExecContext(ctx, contestantID, posterID, contestID, a.SurveyItemID, a.ValueSelected); err != nil {
			return 0, fmt.Errorf("insert answer: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return contestantID, nil
}

func (d *DB) ListContestantsByContest(ctx context.Context, contestID int64) ([]models.Contestant, error) {
	rows, err := d.conn.QueryContext(ctx, `
		SELECT id, contest_id, name, COALESCE(email, ''), phone, COALESCE(address, ''), created_at
		FROM contestant WHERE contest_id = ? ORDER BY created_at`, contestID)
	if err != nil {
		return nil, fmt.Errorf("list contestants: %w", err)
	}
	defer rows.Close()

	var out []models.Contestant
	for rows.Next() {
		var c models.Contestant
		if err := rows.Scan(&c.ID, &c.ContestID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// contestant has exactly one UNIQUE constraint (contest_id, phone) beyond
// its primary key, so matching on the generic message is unambiguous here.
func isUniqueConstraintErr(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
