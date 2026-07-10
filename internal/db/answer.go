package db

import (
	"context"
	"fmt"
)

// ItemDistribution is the count of responses at each of the 5 values for
// one survey item within one contest.
type ItemDistribution struct {
	SurveyItemID int64
	Counts       [5]int // Counts[0] = count of value_selected=1, ... Counts[4] = value_selected=5
	Total        int
}

// Pct returns the rounded percentage of responses at index i (0-4,
// corresponding to value_selected i+1), or 0 if there are no responses yet.
func (d *ItemDistribution) Pct(i int) int {
	if d == nil || d.Total == 0 {
		return 0
	}
	return (d.Counts[i]*100 + d.Total/2) / d.Total
}

// ResultsByContest aggregates answer rows per survey item for the admin
// results view.
func (d *DB) ResultsByContest(ctx context.Context, contestID int64) (map[int64]*ItemDistribution, error) {
	rows, err := d.conn.QueryContext(ctx, `
		SELECT survey_item_id, value_selected, COUNT(*)
		FROM answer WHERE contest_id = ?
		GROUP BY survey_item_id, value_selected`, contestID)
	if err != nil {
		return nil, fmt.Errorf("results by contest: %w", err)
	}
	defer rows.Close()

	out := map[int64]*ItemDistribution{}
	for rows.Next() {
		var itemID int64
		var value, count int
		if err := rows.Scan(&itemID, &value, &count); err != nil {
			return nil, err
		}
		if value < 1 || value > 5 {
			continue
		}
		d, ok := out[itemID]
		if !ok {
			d = &ItemDistribution{SurveyItemID: itemID}
			out[itemID] = d
		}
		d.Counts[value-1] = count
		d.Total += count
	}
	return out, rows.Err()
}
