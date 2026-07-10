// Package models holds the plain data structures shared across the app.
package models

type Survey struct {
	ID          int64
	Description string
	CreatedAt   string
}

type Contest struct {
	ID        int64
	SurveyID  int64
	EndDate   string // ISO 8601, UTC; inclusive end-of-day
	Prize     string
	CreatedAt string
}

type Poster struct {
	ID                 int64
	ContestID          int64
	InternalPosterInfo string
	CreatedAt          string
}

type SurveyItem struct {
	ID        int64
	SurveyID  int64
	Question  string
	Response1 string
	Response2 string
	Response3 string
	Response4 string
	Response5 string
	SortOrder int
}

// Responses returns the five response labels in order, 1-indexed by
// position (Responses()[0] is the label for value_selected=1).
func (s SurveyItem) Responses() [5]string {
	return [5]string{s.Response1, s.Response2, s.Response3, s.Response4, s.Response5}
}

type Contestant struct {
	ID          int64
	ContestID   int64
	PosterID    int64  // which poster's link (survey or direct-entry) they entered through; 0 if unknown
	PosterLabel string // poster.internal_poster_info, for admin display; "" if unknown
	Name        string
	Email       string
	Phone       string
	Address     string
	CreatedAt   string
}

type Answer struct {
	ID            int64
	ContestantID  int64
	Date          string
	PosterID      int64
	ContestID     int64
	SurveyItemID  int64
	ValueSelected int
}

// SurveyBundle is everything needed to render or validate a poster scan:
// the poster, its contest, the contest's survey, and that survey's items
// in display order.
type SurveyBundle struct {
	Poster  Poster
	Contest Contest
	Survey  Survey
	Items   []SurveyItem
}
