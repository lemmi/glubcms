package main

import (
	"flag"
	"log"
	"net/http"
	"text/template"

	"github.com/lemmi/glubcms"
)

type handler struct {
	prefix string
	tmpl   *template.Template
}

func newhandler(prefix, templatepath string) handler {
	return handler{
		prefix: prefix,
		tmpl:   template.Must(template.ParseFiles(templatepath)),
	}
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := glubcms.PageFromDir(h.prefix, r.URL.Path)
	if err := h.tmpl.Execute(w, p); err != nil {
		log.Println(err)
	}
}

func main() {
	prefix := flag.String("prefix", "../example_page", "path to the root dir")
	tmplpath := flag.String("template", "../page_template.html", "path to the template to use")
	addr := flag.String("bind", "localhost:8080", "address to bind to")
	flag.Parse()
	log.Fatal(http.ListenAndServe(*addr, newhandler(*prefix, *tmplpath)))
}
