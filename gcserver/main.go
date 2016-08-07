package main

import (
	"bytes"
	"flag"
	"github.com/pkg/errors"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	g "github.com/gogits/git"
	"github.com/lemmi/ghfs"
	"github.com/lemmi/glubcms"
	"github.com/raymondbutcher/tidyhtml"
)

const (
	tmplPath = "templates"
)

func POE(err error, prefix ...interface{}) {
	if err != nil {
		log.Print(prefix...)
		log.Fatal(err)
	}
}

func parseTemplates(fs http.FileSystem) (*template.Template, error) {
	dir, err := fs.Open(tmplPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot open directory: %q", tmplPath)
	}
	defer dir.Close()
	tmain := template.New("main")
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

// Static file handling without showing directories
// TODO:
// - factor out

func ServeContentFs(w http.ResponseWriter, req *http.Request, fs http.FileSystem) {
	path := filepath.Clean(req.URL.Path)
	f, err := fs.Open(path)
	if err != nil {
		http.Error(w, path, http.StatusNotFound)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		http.Error(w, path, http.StatusInternalServerError)
		return
	}
	if stat.IsDir() {
		http.Error(w, path, http.StatusForbidden)
		return
	}
	http.ServeContent(w, req, stat.Name(), stat.ModTime(), f)
}

type StaticHandler struct {
	fs http.FileSystem
}

func (sh StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ServeContentFs(w, r, sh.fs)
}
func newStaticHandler(c *g.Commit) (http.Handler, error) {
	stree, err := c.Tree.SubTree("static")
	return http.FileServer(ghfs.FromCommit(c, stree)), err
}

// Handling of an article or menu entry

type pageHandler struct {
	c *g.Commit
}

func newPageHandler(c *g.Commit) http.Handler {
	return pageHandler{c}
}
func (h pageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(parseTemplates(ghfs.FromCommit(h.c)))
	stree, err := h.c.Tree.SubTree("pages")
	POE(err, "Pages")

	p := glubcms.PageFromDir(ghfs.FromCommit(h.c, stree), r.URL.Path)
	buf := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&buf, "main", p); err != nil {
		log.Println(errors.Wrapf(err, "template execution failed: %q", r.URL.Path))
		return
	}
	tbuf := bytes.Buffer{}
	if err := tidyhtml.Copy(&tbuf, &buf); err != nil {
		log.Println(errors.Wrapf(err, "tidyhtml failed: %q", r.URL.Path))
		return
	}
	w.Header().Set("Content-Type", "text/html")
	http.ServeContent(w, r, "", h.c.Author.When, bytes.NewReader(tbuf.Bytes()))
}

// Main handling of the site
// TODO:
// - parse meta entries first
// - choose handler based on that, not by path

type handler struct {
	prefix string
}

func newHandler(prefix string) handler {
	return handler{
		prefix: prefix,
	}
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, err := filepath.Abs(h.prefix)
	// TODO errors package, proper http codes
	POE(err, "Filepath")

	repo, err := g.OpenRepository(path)
	POE(err, "OpenRepository")

	commit, err := repo.GetCommitOfBranch("master")
	POE(err, "LookupBranch")

	mux := http.NewServeMux()

	staticHandler, err := newStaticHandler(commit)
	POE(err, "StaticHandler")

	mux.Handle("/static/", http.StripPrefix("/static/", staticHandler))
	mux.Handle("/robots.txt", staticHandler)
	mux.Handle("/favicon.ico", staticHandler)
	mux.Handle("/", newPageHandler(commit))
	w.Header().Set("ETag", strings.Trim(commit.Id.String(), "\""))
	w.Header().Set("Cache-Control", "max-age=32")
	mux.ServeHTTP(w, r)
}

func main() {
	prefix := flag.String("prefix", "../example_page", "path to the root dir")
	addr := flag.String("bind", "localhost:8080", "address or path to bind to")
	network := flag.String("net", "tcp", `"tcp", "tcp4", "tcp6", "unix" or "unixpacket"`)
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
	log.Fatal(http.Serve(ln, newHandler(*prefix)))
}
