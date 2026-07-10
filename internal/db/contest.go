package db

import (
	"context"
	"fmt"

	"github.com/thomaslsimpson/qrsurvey/internal/models"
)

func (d *DB) CreateContest(ctx context.Context, surveyID int64, endDate, prize string) (int64, error) {
	res, err := d.conn.ExecContext(ctx, `INSERT INTO contest (survey_id, end_date, prize) VALUES (?, ?, ?)`, surveyID, endDate, prize)
	if err != nil {
		return 0, fmt.Errorf("insert contest: %w", err)
	}
	return res.LastInsertId()
}

func (d *DB) UpdateContest(ctx context.Context, id, surveyID int64, endDate, prize string) error {
	_, err := d.conn.ExecContext(ctx, `UPDATE contest SET survey_id = ?, end_date = ?, prize = ? WHERE id = ?`, surveyID, endDate, prize, id)
	if err != nil {
		return fmt.Errorf("update contest: %w", err)
	}
	return nil
}

func (d *DB) GetContest(ctx context.Context, id int64) (models.Contest, error) {
	var c models.Contest
	row := d.conn.QueryRowContext(ctx, `SELECT id, survey_id, end_date, prize, created_at FROM contest WHERE id = ?`, id)
	if err := row.Scan(&c.ID, &c.SurveyID, &c.EndDate, &c.Prize, &c.CreatedAt); err != nil {
		return models.Contest{}, err
	}
	return c, nil
}

func (d *DB) ListContests(ctx context.Context) ([]models.Contest, error) {
	rows, err := d.conn.QueryContext(ctx, `SELECT id, survey_id, end_date, prize, created_at FROM contest ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list contests: %w", err)
	}
	defer rows.Close()

	var out []models.Contest
	for rows.Next() {
		var c models.Contest
		if err := rows.Scan(&c.ID, &c.SurveyID, &c.EndDate, &c.Prize, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
