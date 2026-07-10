// Package web embeds and serves the HTML templates and static assets
// (CSS/JS) for the public survey wizard, so the whole app ships as one
// Go binary with no separate frontend build step.
package web

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"net/http"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

var tmpl = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

// StaticHandler serves the embedded CSS/JS under /static/.
func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}

// WizardData is the payload rendered into the survey wizard page. The
// question content itself lives only in DataJSON — the client-side wizard
// builds each question screen dynamically from that blob — so the template
// only needs a count for the welcome-screen copy ("answer N questions").
type WizardData struct {
	BusinessName string // convention: survey.description holds the short business/location name
	PosterID     int64
	ContestID    int64
	ItemCount    int
	DataJSON     template.JS
}

func RenderWizard(w io.Writer, data WizardData) error {
	return tmpl.ExecuteTemplate(w, "wizard.html", data)
}

func RenderEnded(w io.Writer, businessName string) error {
	return tmpl.ExecuteTemplate(w, "ended.html", map[string]string{"BusinessName": businessName})
}

// DirectEntryData is the payload for the alternate-method-of-entry page:
// contact form only, no survey questions. The page's own inline script
// derives the submit URL from window.location.pathname, same as the main
// wizard does, so no path needs to be threaded through here.
type DirectEntryData struct {
	BusinessName string
}

func RenderDirectEntry(w io.Writer, data DirectEntryData) error {
	return tmpl.ExecuteTemplate(w, "direct_entry.html", data)
}

func RenderNotReady(w io.Writer) error {
	return tmpl.ExecuteTemplate(w, "not_ready.html", nil)
}

// RenderAdmin executes any named admin_*.html template with the given data.
func RenderAdmin(w io.Writer, name string, data any) error {
	return tmpl.ExecuteTemplate(w, name, data)
}

// MarshalWizardJSON is a small helper so handlers don't need to import
// encoding/json directly just to build the embedded survey-data blob.
func MarshalWizardJSON(v any) (template.JS, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	// encoding/json HTML-escapes <, >, & by default, so this is safe to
	// embed verbatim inside a <script> element.
	return template.JS(b), nil
}
