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

	"github.com/lemmi/ghfs"
	g "github.com/lemmi/git"
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
		return nil, err
	}
	tmain := template.New("main")
	fis, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}
	for _, fi := range fis {
		if !strings.HasSuffix(fi.Name(), ".tmpl") {
			continue
		}
		data, err := fs.Open(filepath.Join(tmplPath, fi.Name()))
		if err != nil {
			return nil, err
		}
		databytes, err := ioutil.ReadAll(data)
		data.Close()
		if err != nil {
			return nil, err
		}

		tname := strings.TrimSuffix(fi.Name(), ".tmpl")
		_, err = tmain.New(tname).Parse(string(databytes))
		if err != nil {
			return nil, err
		}
	}

	return tmain, nil
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
		log.Println(err)
		return
	}
	tbuf := bytes.Buffer{}
	if err := tidyhtml.Copy(&tbuf, &buf); err != nil {
		log.Println(err)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	http.ServeContent(w, r, "", h.c.Author.When, bytes.NewReader(tbuf.Bytes()))
}

// func ServeContent(w ResponseWriter, req *Request, name string, modtime time.Time, content io.ReadSeeker)
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
