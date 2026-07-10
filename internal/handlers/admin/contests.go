package admin

import (
	"encoding/csv"
	"net/http"
	"strconv"

	"github.com/thomaslsimpson/qrsurvey/internal/db"
	"github.com/thomaslsimpson/qrsurvey/internal/models"
	"github.com/thomaslsimpson/qrsurvey/internal/qrcode"
	"github.com/thomaslsimpson/qrsurvey/internal/web"
)

type contestsPageData struct {
	Contests []models.Contest
	Surveys  []models.Survey
}

func (h *Handlers) ListContests(w http.ResponseWriter, r *http.Request) {
	contests, err := h.DB.ListContests(r.Context())
	if err != nil {
		h.internalError(w, err, "list contests")
		return
	}
	surveys, err := h.DB.ListSurveys(r.Context())
	if err != nil {
		h.internalError(w, err, "list surveys")
		return
	}
	if err := web.RenderAdmin(w, "admin_contests.html", contestsPageData{Contests: contests, Surveys: surveys}); err != nil {
		h.Logger.Error("render admin_contests", "err", err)
	}
}

// dateToEndOfDayRFC3339 converts an HTML date input value ("2026-08-31")
// into the RFC3339 UTC end-of-day timestamp contest.end_date is stored as,
// so a contest remains enterable through the entire last calendar day.
func dateToEndOfDayRFC3339(date string) string {
	if len(date) != len("2006-01-02") {
		return date
	}
	return date + "T23:59:59Z"
}

func (h *Handlers) CreateContest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	surveyID, err := strconv.ParseInt(r.FormValue("survey_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid survey", http.StatusBadRequest)
		return
	}
	endDate := r.FormValue("end_date")
	prize := r.FormValue("prize")
	if endDate == "" || prize == "" {
		http.Error(w, "end date and prize are required", http.StatusBadRequest)
		return
	}
	id, err := h.DB.CreateContest(r.Context(), surveyID, dateToEndOfDayRFC3339(endDate), prize)
	if err != nil {
		h.internalError(w, err, "create contest")
		return
	}
	http.Redirect(w, r, "/admin/contests/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

type contestDetailPageData struct {
	Contest      models.Contest
	Survey       models.Survey
	EndDateInput string
	Posters      []models.Poster
	Items        []models.SurveyItem
	ItemCount    int
	Distribution map[int64]*db.ItemDistribution
	Contestants  []models.Contestant
	BaseURL      string
}

func (h *Handlers) ContestDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contest, err := h.DB.GetContest(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	survey, err := h.DB.GetSurvey(r.Context(), contest.SurveyID)
	if err != nil {
		h.internalError(w, err, "get survey")
		return
	}
	items, err := h.DB.ListSurveyItems(r.Context(), survey.ID)
	if err != nil {
		h.internalError(w, err, "list survey items")
		return
	}
	posters, err := h.DB.ListPostersByContest(r.Context(), id)
	if err != nil {
		h.internalError(w, err, "list posters")
		return
	}
	dist, err := h.DB.ResultsByContest(r.Context(), id)
	if err != nil {
		h.internalError(w, err, "results by contest")
		return
	}
	contestants, err := h.DB.ListContestantsByContest(r.Context(), id)
	if err != nil {
		h.internalError(w, err, "list contestants")
		return
	}

	data := contestDetailPageData{
		Contest:      contest,
		Survey:       survey,
		EndDateInput: contest.EndDate[:10],
		Posters:      posters,
		Items:        items,
		ItemCount:    len(items),
		Distribution: dist,
		Contestants:  contestants,
		BaseURL:      h.BaseURL,
	}
	if err := web.RenderAdmin(w, "admin_contest_detail.html", data); err != nil {
		h.Logger.Error("render admin_contest_detail", "err", err)
	}
}

func (h *Handlers) UpdateContest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	contest, err := h.DB.GetContest(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	endDate := dateToEndOfDayRFC3339(r.FormValue("end_date"))
	if err := h.DB.UpdateContest(r.Context(), id, contest.SurveyID, endDate, r.FormValue("prize")); err != nil {
		h.internalError(w, err, "update contest")
		return
	}
	http.Redirect(w, r, "/admin/contests/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (h *Handlers) CreatePoster(w http.ResponseWriter, r *http.Request) {
	contestID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	info := r.FormValue("internal_poster_info")
	if info == "" {
		http.Error(w, "label is required", http.StatusBadRequest)
		return
	}

	contest, err := h.DB.GetContest(r.Context(), contestID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Refuse to create a poster for a survey with no questions — a scan
	// against it would have nothing to show.
	count, err := h.DB.CountSurveyItems(r.Context(), contest.SurveyID)
	if err != nil {
		h.internalError(w, err, "count survey items")
		return
	}
	if count == 0 {
		http.Error(w, "this survey has no questions yet", http.StatusBadRequest)
		return
	}

	if _, err := h.DB.CreatePoster(r.Context(), contestID, info); err != nil {
		h.internalError(w, err, "create poster")
		return
	}
	http.Redirect(w, r, "/admin/contests/"+strconv.FormatInt(contestID, 10), http.StatusSeeOther)
}

func (h *Handlers) PosterQRCode(w http.ResponseWriter, r *http.Request) {
	posterID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if _, err := h.DB.GetPoster(r.Context(), posterID); err != nil {
		http.NotFound(w, r)
		return
	}
	png, err := qrcode.PNGForPoster(h.QRCacheDir, posterID, h.BaseURL)
	if err != nil {
		h.internalError(w, err, "generate qr code")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func (h *Handlers) ContestantsCSV(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contestants, err := h.DB.ListContestantsByContest(r.Context(), id)
	if err != nil {
		h.internalError(w, err, "list contestants")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"contest-"+strconv.FormatInt(id, 10)+"-contestants.csv\"")
	cw := csv.NewWriter(w)
	cw.Write([]string{"name", "email", "phone", "address", "entered_at"})
	for _, c := range contestants {
		cw.Write([]string{c.Name, c.Email, c.Phone, c.Address, c.CreatedAt})
	}
	cw.Flush()
}
