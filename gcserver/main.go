package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"text/template"

	"github.com/lemmi/glubcms"
)

type handler struct {
	prefix     string
	pageprefix string
	tmpl       *template.Template
}

func newhandler(prefix string) handler {
	return handler{
		prefix:     prefix,
		pageprefix: filepath.Join(prefix, "pages"),
		tmpl:       template.Must(template.ParseGlob(filepath.Join(prefix, "templates", "*.tmpl"))),
	}
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := glubcms.PageFromDir(h.pageprefix, r.URL.Path)
	if err := h.tmpl.ExecuteTemplate(w, "main.tmpl", p); err != nil {
		log.Println(err)
	}
}

func main() {
	prefix := flag.String("prefix", "../example_page", "path to the root dir")
	addr := flag.String("bind", "localhost:8080", "address to bind to")
	flag.Parse()
	staticHandler := http.FileServer(http.Dir(filepath.Join(*prefix, "static")))
	http.Handle("/static/", http.StripPrefix("/static/", staticHandler))
	http.Handle("/robots.txt", staticHandler)
	http.Handle("/favicon.ico", staticHandler)
	http.Handle("/", newhandler(*prefix))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
