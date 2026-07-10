package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/thomaslsimpson/qrsurvey/internal/models"
)

func (d *DB) CreateSurvey(ctx context.Context, description string) (int64, error) {
	res, err := d.conn.ExecContext(ctx, `INSERT INTO survey (description) VALUES (?)`, description)
	if err != nil {
		return 0, fmt.Errorf("insert survey: %w", err)
	}
	return res.LastInsertId()
}

func (d *DB) UpdateSurvey(ctx context.Context, id int64, description string) error {
	_, err := d.conn.ExecContext(ctx, `UPDATE survey SET description = ? WHERE id = ?`, description, id)
	if err != nil {
		return fmt.Errorf("update survey: %w", err)
	}
	return nil
}

func (d *DB) GetSurvey(ctx context.Context, id int64) (models.Survey, error) {
	var s models.Survey
	row := d.conn.QueryRowContext(ctx, `SELECT id, description, created_at FROM survey WHERE id = ?`, id)
	if err := row.Scan(&s.ID, &s.Description, &s.CreatedAt); err != nil {
		return models.Survey{}, err
	}
	return s, nil
}

func (d *DB) ListSurveys(ctx context.Context) ([]models.Survey, error) {
	rows, err := d.conn.QueryContext(ctx, `SELECT id, description, created_at FROM survey ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list surveys: %w", err)
	}
	defer rows.Close()

	var out []models.Survey
	for rows.Next() {
		var s models.Survey
		if err := rows.Scan(&s.ID, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (d *DB) CreateSurveyItem(ctx context.Context, item models.SurveyItem) (int64, error) {
	res, err := d.conn.ExecContext(ctx, `
		INSERT INTO survey_item (survey_id, question, response_1, response_2, response_3, response_4, response_5, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.SurveyID, item.Question, item.Response1, item.Response2, item.Response3, item.Response4, item.Response5, item.SortOrder)
	if err != nil {
		return 0, fmt.Errorf("insert survey_item: %w", err)
	}
	return res.LastInsertId()
}

func (d *DB) UpdateSurveyItem(ctx context.Context, item models.SurveyItem) error {
	_, err := d.conn.ExecContext(ctx, `
		UPDATE survey_item
		SET question = ?, response_1 = ?, response_2 = ?, response_3 = ?, response_4 = ?, response_5 = ?, sort_order = ?
		WHERE id = ?`,
		item.Question, item.Response1, item.Response2, item.Response3, item.Response4, item.Response5, item.SortOrder, item.ID)
	if err != nil {
		return fmt.Errorf("update survey_item: %w", err)
	}
	return nil
}

func (d *DB) DeleteSurveyItem(ctx context.Context, id int64) error {
	_, err := d.conn.ExecContext(ctx, `DELETE FROM survey_item WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete survey_item: %w", err)
	}
	return nil
}

func (d *DB) ListSurveyItems(ctx context.Context, surveyID int64) ([]models.SurveyItem, error) {
	rows, err := d.conn.QueryContext(ctx, `
		SELECT id, survey_id, question, response_1, response_2, response_3, response_4, response_5, sort_order
		FROM survey_item WHERE survey_id = ? ORDER BY sort_order, id`, surveyID)
	if err != nil {
		return nil, fmt.Errorf("list survey_items: %w", err)
	}
	defer rows.Close()

	var out []models.SurveyItem
	for rows.Next() {
		var it models.SurveyItem
		if err := rows.Scan(&it.ID, &it.SurveyID, &it.Question, &it.Response1, &it.Response2, &it.Response3, &it.Response4, &it.Response5, &it.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// CountSurveyItems is used by admin validation to refuse activating a
// contest/poster whose survey has zero questions.
func (d *DB) CountSurveyItems(ctx context.Context, surveyID int64) (int, error) {
	var n int
	row := d.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM survey_item WHERE survey_id = ?`, surveyID)
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

var ErrNotFound = sql.ErrNoRows
