package views

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type Renderer struct {
	templates map[string]*template.Template
	isDev     bool
	baseDir   string
}

func NewRenderer(baseDir string, isDev bool) *Renderer {
	r := &Renderer{
		templates: make(map[string]*template.Template),
		isDev:     isDev,
		baseDir:   baseDir,
	}
	r.loadTemplates()
	return r
}

func (r *Renderer) funcMap() template.FuncMap {
	return template.FuncMap{
		"json": func(v any) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"itof": func(i int) float64 {
			return float64(i)
		},
		"mulf": func(a, b float64) float64 {
			return a * b
		},
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
	}
}

func (r *Renderer) loadTemplates() {
	baseLayout := filepath.Join(r.baseDir, "layouts", "base.html")
	components, _ := filepath.Glob(filepath.Join(r.baseDir, "components", "*.html"))

	// Discover layouts (excluding base.html)
	layoutGlob, _ := filepath.Glob(filepath.Join(r.baseDir, "layouts", "*.html"))
	layouts := make(map[string]string)
	for _, l := range layoutGlob {
		name := strings.TrimSuffix(filepath.Base(l), ".html")
		if name != "base" {
			layouts[name] = l
		}
	}

	// Load pages: each page is compiled with each layout
	pages, _ := filepath.Glob(filepath.Join(r.baseDir, "pages", "*.html"))
	for _, page := range pages {
		pageName := strings.TrimSuffix(filepath.Base(page), ".html")
		for layoutName, layoutFile := range layouts {
			files := []string{baseLayout, layoutFile}
			files = append(files, components...)
			files = append(files, page)
			tmpl := template.Must(template.New(filepath.Base(page)).Funcs(r.funcMap()).ParseFiles(files...))
			r.templates[layoutName+":"+pageName] = tmpl
		}
	}

	// Load partials
	partials, _ := filepath.Glob(filepath.Join(r.baseDir, "partials", "*.html"))
	for _, partial := range partials {
		name := strings.TrimSuffix(filepath.Base(partial), ".html")
		files := append([]string{partial}, components...)
		tmpl := template.Must(template.New(filepath.Base(partial)).Funcs(r.funcMap()).ParseFiles(files...))
		r.templates["partial:"+name] = tmpl
	}
}

func (r *Renderer) Page(w http.ResponseWriter, layout, name string, data any) {
	if r.isDev {
		r.loadTemplates()
	}

	key := layout + ":" + name
	tmpl, ok := r.templates[key]
	if !ok {
		log.Printf("template %s not found", key)
		http.Error(w, "Page not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("render error: %v", err)
	}
}

func (r *Renderer) Partial(w http.ResponseWriter, name string, data any) {
	if r.isDev {
		r.loadTemplates()
	}

	tmpl, ok := r.templates["partial:"+name]
	if !ok {
		log.Printf("template partial:%s not found", name)
		http.Error(w, "Partial not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render error: %v", err)
	}
}

// IsHTMX checks if the request is an HTMX request
func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Ensure the Renderer satisfies any interface if needed
var _ fs.FS = (*noop)(nil)

type noop struct{}

func (noop) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }
