package admin

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/thomaslsimpson/qrsurvey/internal/db"
	"github.com/thomaslsimpson/qrsurvey/internal/models"
	"github.com/thomaslsimpson/qrsurvey/internal/web"
)

type Handlers struct {
	DB         *db.DB
	Logger     *slog.Logger
	BaseURL    string
	QRCacheDir string
	HashSecret string
}

func (h *Handlers) internalError(w http.ResponseWriter, err error, msg string) {
	h.Logger.Error(msg, "err", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

type surveysPageData struct {
	Surveys []models.Survey
}

func (h *Handlers) ListSurveys(w http.ResponseWriter, r *http.Request) {
	surveys, err := h.DB.ListSurveys(r.Context())
	if err != nil {
		h.internalError(w, err, "list surveys")
		return
	}
	if err := web.RenderAdmin(w, "admin_surveys.html", surveysPageData{Surveys: surveys}); err != nil {
		h.Logger.Error("render admin_surveys", "err", err)
	}
}

func (h *Handlers) CreateSurvey(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	description := r.FormValue("description")
	if description == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	}
	id, err := h.DB.CreateSurvey(r.Context(), description)
	if err != nil {
		h.internalError(w, err, "create survey")
		return
	}
	http.Redirect(w, r, "/admin/surveys/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

type surveyDetailPageData struct {
	Survey models.Survey
	Items  []models.SurveyItem
}

func (h *Handlers) SurveyDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	survey, err := h.DB.GetSurvey(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	items, err := h.DB.ListSurveyItems(r.Context(), id)
	if err != nil {
		h.internalError(w, err, "list survey items")
		return
	}
	if err := web.RenderAdmin(w, "admin_survey_detail.html", surveyDetailPageData{Survey: survey, Items: items}); err != nil {
		h.Logger.Error("render admin_survey_detail", "err", err)
	}
}

func (h *Handlers) UpdateSurvey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	if err := h.DB.UpdateSurvey(r.Context(), id, r.FormValue("description")); err != nil {
		h.internalError(w, err, "update survey")
		return
	}
	http.Redirect(w, r, "/admin/surveys/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (h *Handlers) CreateSurveyItem(w http.ResponseWriter, r *http.Request) {
	surveyID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	item := models.SurveyItem{
		SurveyID:  surveyID,
		Question:  r.FormValue("question"),
		Response1: r.FormValue("response_1"),
		Response2: r.FormValue("response_2"),
		Response3: r.FormValue("response_3"),
		Response4: r.FormValue("response_4"),
		Response5: r.FormValue("response_5"),
		SortOrder: sortOrder,
	}
	if item.Question == "" || item.Response1 == "" || item.Response2 == "" || item.Response3 == "" || item.Response4 == "" || item.Response5 == "" {
		http.Error(w, "question and all five responses are required", http.StatusBadRequest)
		return
	}
	if _, err := h.DB.CreateSurveyItem(r.Context(), item); err != nil {
		h.internalError(w, err, "create survey item")
		return
	}
	http.Redirect(w, r, "/admin/surveys/"+strconv.FormatInt(surveyID, 10), http.StatusSeeOther)
}

func (h *Handlers) DeleteSurveyItem(w http.ResponseWriter, r *http.Request) {
	surveyID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	itemID, err := strconv.ParseInt(r.PathValue("itemID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.DB.DeleteSurveyItem(r.Context(), itemID); err != nil {
		h.internalError(w, err, "delete survey item")
		return
	}
	http.Redirect(w, r, "/admin/surveys/"+strconv.FormatInt(surveyID, 10), http.StatusSeeOther)
}
