package views

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/gorilla/csrf"
	"github.com/torresjeff/gallery/context"
)

type View struct {
	Template *template.Template
	Layout   string
}

const (
	SharedDir   = "views/shared/"
	TemplateExt = ".gohtml"
	TemplateDir = "views/"
)

func NewView(layout string, files ...string) *View {
	addTemplatePath(files)
	addTemplateExt(files)
	files = append(files, sharedFiles()...)
	// t, err := template.ParseFiles(files...)
	// if err != nil {
	// 	panic(err)
	// }

	// New("") gives us a template that we can add a function to before finally parsing our files
	t, err := template.New("").Funcs(template.FuncMap{
		"csrfField": func() (template.HTML, error) {
			return "", errors.New("csrfField is not implemented")
		},
		"pathEscape": func(s string) string {
			return url.PathEscape(s)
		},
	}).ParseFiles(files...)
	if err != nil {
		panic(err)
	}

	return &View{
		Template: t,
		Layout:   layout,
	}
}

func sharedFiles() []string {
	files, err := filepath.Glob(SharedDir + "*" + TemplateExt)
	if err != nil {
		panic(err)
	}
	return files
}

func (v *View) Render(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "text/html")
	var vd Data
	switch d := data.(type) {
	case Data:
		vd = d
	default:
		data = Data{
			Yield: data,
		}
	}

	// Look up the alert and assign it if one is persisted
	if alert := getAlert(r); alert != nil {
		vd.Alert = alert
		clearAlert(w)
	}

	vd.User = context.User(r.Context())
	var buf bytes.Buffer

	// Create the CSRF field using the current request
	csrfField := csrf.TemplateField(r)
	templ := v.Template.Funcs(template.FuncMap{
		"csrfField": func() template.HTML {
			// Use this closure to return the CSRF Field for any templates that need to access it
			return csrfField
		},
	})
	err := templ.ExecuteTemplate(&buf, v.Layout, vd)
	if err != nil {
		http.Error(w, "Something went wrong. If the problem persists, please email support@lenslocked.com", http.StatusInternalServerError)
		return
	}
	io.Copy(w, &buf)
}

func (v *View) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.Render(w, r, nil)
}

func addTemplatePath(files []string) {
	for i, f := range files {
		files[i] = TemplateDir + f
	}
}

func addTemplateExt(files []string) {
	for i := range files {
		files[i] += TemplateExt
	}
}
