package main

import (
	"bytes"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lemmi/compress"
	"github.com/lemmi/ghfs"
	g "github.com/lemmi/git"
	"github.com/lemmi/glubcms"
	"github.com/pkg/errors"
	"github.com/raymondbutcher/tidyhtml"
)

const (
	tmplPath = "templates"
)

var (
	DEBUG bool
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func HttpError(w http.ResponseWriter, code int, logErr error) {
	if DEBUG {
		switch err := logErr.(type) {
		case stackTracer:
			log.Print(err)
			log.Printf("%+v", err.StackTrace())
		default:
			log.Print(err)
		}
	} else {
		log.Print(logErr)
	}
	http.Error(w, http.StatusText(code), code)
}

func parseTemplates(fs http.FileSystem) (*template.Template, error) {
	dir, err := fs.Open(tmplPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot open directory: %q", tmplPath)
	}
	defer dir.Close()
	tmain := template.New("_")
	fis, err := dir.Readdir(-1)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read directory: %q", tmplPath)
	}
	for _, fi := range fis {
		if !strings.HasSuffix(fi.Name(), ".tmpl") {
			continue
		}
		fpath := filepath.Join(tmplPath, fi.Name())
		data, err := fs.Open(fpath)
		if err != nil {
			return nil, errors.Wrapf(err, "Cannot open file: %q", fpath)
		}
		databytes, err := ioutil.ReadAll(data)
		data.Close()
		if err != nil {
			return nil, errors.Wrapf(err, "Cannot read file: %q", fpath)
		}

		tname := strings.TrimSuffix(fi.Name(), ".tmpl")
		_, err = tmain.New(tname).Parse(string(databytes))
		if err != nil {
			return nil, errors.Wrapf(err, "Cannot parse template: %q", fpath)
		}
	}

	return tmain, nil
}

// Handling of an article or menu entry
// TODO:
// - use http.FileSystem only

type pageHandler struct {
	glubcms.StaticHandler
}

func (h pageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(parseTemplates(h))
	pages := h.Cd("pages")
	//stree, err := h.c.Tree.SubTree("pages")
	//if err != nil {
	//	HttpError(w, http.StatusNotFound, errors.Wrap(err, "Pages"))
	//	return
	//}

	p, err := glubcms.PageFromDir(pages, r.URL.Path)
	if err != nil {
		HttpError(w, http.StatusInternalServerError, errors.Wrapf(err, "page generation failed"))
		return
	}
	buf := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&buf, "main", p); err != nil {
		HttpError(w, http.StatusInternalServerError, errors.Wrapf(err, "template execution failed: %q\n%s", r.URL.Path, tmpl.DefinedTemplates()))
		return
	}
	tbuf := bytes.Buffer{}
	if err := tidyhtml.Copy(&tbuf, &buf); err != nil {
		HttpError(w, http.StatusInternalServerError, errors.Wrapf(err, "tidyhtml failed: %q", r.URL.Path))
		return
	}
	w.Header().Set("Content-Type", "text/html")
	http.ServeContent(w, r, "", p.ModTime, bytes.NewReader(tbuf.Bytes()))
}

// Main handling of the site
// TODO:
// - parse meta entries first
// - choose handler based on that, not by path

type handler struct {
	prefix string
	git    bool
}

func newHandler(prefix string, git bool) handler {
	return handler{
		prefix: prefix,
		git:    git,
	}
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, err := filepath.Abs(h.prefix)
	if err != nil {
		HttpError(w, http.StatusInternalServerError, errors.Wrap(err, "filepath.Abs("+h.prefix+")"))
		return
	}

	var fs http.FileSystem

	if h.git {
		repo, err := g.OpenRepository(path)
		if err != nil {
			HttpError(w, http.StatusInternalServerError, errors.Wrap(err, "g.OpenRepository("+path+")"))
			return
		}

		commit, err := repo.GetCommitOfBranch("master")
		if err != nil {
			HttpError(w, http.StatusInternalServerError, errors.Wrap(err, "Can not open master branch"))
			return
		}
		fs = ghfs.FromCommit(commit)
		w.Header().Set("ETag", strings.Trim(commit.Id.String(), "\""))
	} else {
		fs = http.Dir(path)
	}

	staticHandler := glubcms.NewStaticHandler(fs)

	mux := http.NewServeMux()
	mux.Handle("/static/", staticHandler)
	mux.Handle("/robots.txt", staticHandler.Cd("/static"))
	mux.Handle("/favicon.ico", staticHandler.Cd("/static"))
	mux.Handle("/", pageHandler{staticHandler})
	w.Header().Set("Cache-Control", "max-age=32")
	mux.ServeHTTP(w, r)
}

func main() {
	prefix := flag.String("prefix", "../example_page", "path to the root dir")
	addr := flag.String("bind", "localhost:8080", "address or path to bind to")
	network := flag.String("net", "tcp", `"tcp", "tcp4", "tcp6", "unix" or "unixpacket"`)
	git := flag.Bool("git", false, "prefix is a git repo")
	flag.BoolVar(&DEBUG, "debug", false, "set debug output")
	flag.Parse()
	ln, err := net.Listen(*network, *addr)
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	if strings.HasPrefix(*network, "unix") {
		err = os.Chmod(*addr, 0666)
	}
	if err != nil {
		panic(err)
	}
	log.Println("Starting")
	if DEBUG {
		log.Println("prefix: ", *prefix)
		log.Println("addr: ", *addr)
		log.Println("network: ", *network)
		log.Println("git: ", *git)
	}
	log.Fatal(http.Serve(ln, compress.New(newHandler(*prefix, *git))))
}
