package db

import (
	"context"
	"fmt"

	"github.com/thomaslsimpson/qrsurvey/internal/models"
)

func (d *DB) CreatePoster(ctx context.Context, contestID int64, info string) (int64, error) {
	res, err := d.conn.ExecContext(ctx, `INSERT INTO poster (contest_id, internal_poster_info) VALUES (?, ?)`, contestID, info)
	if err != nil {
		return 0, fmt.Errorf("insert poster: %w", err)
	}
	return res.LastInsertId()
}

func (d *DB) GetPoster(ctx context.Context, id int64) (models.Poster, error) {
	var p models.Poster
	row := d.conn.QueryRowContext(ctx, `SELECT id, contest_id, internal_poster_info, created_at FROM poster WHERE id = ?`, id)
	if err := row.Scan(&p.ID, &p.ContestID, &p.InternalPosterInfo, &p.CreatedAt); err != nil {
		return models.Poster{}, err
	}
	return p, nil
}

func (d *DB) ListPostersByContest(ctx context.Context, contestID int64) ([]models.Poster, error) {
	rows, err := d.conn.QueryContext(ctx, `SELECT id, contest_id, internal_poster_info, created_at FROM poster WHERE contest_id = ? ORDER BY id`, contestID)
	if err != nil {
		return nil, fmt.Errorf("list posters: %w", err)
	}
	defer rows.Close()

	var out []models.Poster
	for rows.Next() {
		var p models.Poster
		if err := rows.Scan(&p.ID, &p.ContestID, &p.InternalPosterInfo, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ResolveSurveyBundle performs the poster -> contest -> survey -> items
// resolution needed to render (or validate a submission against) a single
// poster scan.
func (d *DB) ResolveSurveyBundle(ctx context.Context, posterID int64) (models.SurveyBundle, error) {
	var b models.SurveyBundle
	row := d.conn.QueryRowContext(ctx, `
		SELECT
			p.id, p.contest_id, p.internal_poster_info, p.created_at,
			c.id, c.survey_id, c.end_date, c.prize, c.created_at,
			s.id, s.description, s.created_at
		FROM poster p
		JOIN contest c ON c.id = p.contest_id
		JOIN survey s ON s.id = c.survey_id
		WHERE p.id = ?`, posterID)
	err := row.Scan(
		&b.Poster.ID, &b.Poster.ContestID, &b.Poster.InternalPosterInfo, &b.Poster.CreatedAt,
		&b.Contest.ID, &b.Contest.SurveyID, &b.Contest.EndDate, &b.Contest.Prize, &b.Contest.CreatedAt,
		&b.Survey.ID, &b.Survey.Description, &b.Survey.CreatedAt,
	)
	if err != nil {
		return models.SurveyBundle{}, err
	}

	items, err := d.ListSurveyItems(ctx, b.Survey.ID)
	if err != nil {
		return models.SurveyBundle{}, err
	}
	b.Items = items

	return b, nil
}
