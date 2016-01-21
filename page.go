package glubcms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	bm "github.com/microcosm-cc/bluemonday"
	bf "github.com/russross/blackfriday"
)

const (
	PSep = string(os.PathSeparator)
)

type Entry interface {
	Active() bool
	Author() string
	Date() time.Time
	HTML() template.HTML
	IsArticle() bool
	Link() string
	Priority() int
	Title() string
	Next() Entry
	Prev() Entry
}

type entries []entry

func (e entries) Len() int {
	return len(e)
}
func (e entries) Less(i, j int) bool {
	ei, ej := e[i], e[j]
	if ei.Priority() != ej.Priority() {
		return ei.Priority() > ej.Priority()
	}
	return e[i].Date().After(e[j].Date())
}
func (e entries) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
func (e entries) Split() (Menu, Articles entries) {
	for _, v := range e {
		if v.IsArticle() {
			Articles = append(Articles, v)
		} else {
			Menu = append(Menu, v)
		}
	}

	return
}

type GCTime time.Time

const GCTimeLayout = "2006-01-02 15:04"

func (t *GCTime) UnmarshalJSON(b []byte) error {
	tmp, err := time.Parse(GCTimeLayout, strings.Trim(string(b), "\""))
	*t = GCTime(tmp)
	return err
}

func (t GCTime) String() string {
	return time.Time(t).Format(GCTimeLayout)
}
func (t GCTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

type Meta struct {
	Author   string
	Date     GCTime
	Title    string
	Priority int
	Hidden   bool `json:",omitempty"`
	Unsafe   bool `json:",omitempty"`
}

type entry struct {
	meta      Meta
	active    bool
	html      []byte
	isarticle bool
	link      url.URL
	next      *entry
	prev      *entry
}

func (e entry) Active() bool {
	return e.active
}
func (a entry) Author() string {
	return a.meta.Author
}
func (e entry) Date() time.Time {
	return time.Time(e.meta.Date)
}
func (e entry) HTML() template.HTML {
	return template.HTML(e.html)
}
func (e entry) IsArticle() bool {
	return e.isarticle
}
func (e entry) Link() string {
	return e.link.String()
}
func (e entry) Priority() int {
	return e.meta.Priority
}
func (e entry) Title() string {
	return e.meta.Title
}
func (e entry) Next() Entry {
	return e.next
}
func (e entry) Prev() Entry {
	return e.prev
}

func entryFromMeta(fs http.FileSystem, path string) (entry, error) {
	ret := entry{}
	f, err := fs.Open(filepath.Join(path))
	if err != nil {
		return ret, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&ret.meta)
	if err != nil {
		err = &os.PathError{
			Op:   "Parsing json in",
			Path: path,
			Err:  err,
		}
	}
	return ret, err
}

func entryFromDir(fs http.FileSystem, path, activepath string) (ret entry, err error) {
	ret, err = entryFromMeta(fs, filepath.Join(path, "meta.json"))
	if err != nil {
		log.Println(err)
		return
	}

	// skip hidden folders, unless directly asked for
	if ret.meta.Hidden && activepath != path {
		return
	}

	ret.link = url.URL{Path: path}
	// .Dir() removes trailing slashes, this prevents
	// /a/b/ beeing seen as prefix from /a/bc/
	if strings.HasPrefix(activepath+PSep, path+PSep) {
		ret.active = true
	}

	md, err := fs.Open(filepath.Join(path, "article.md"))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println(err)
		}
		return
	}
	defer md.Close()

	// only attempt to convert the markdown if it's the active path
	if activepath == path {
		b, err := ioutil.ReadAll(md)
		if err != nil {
			log.Println(err)
			return ret, err
		}
		ret.html = bf.Markdown(b, bf.HtmlRenderer(bf.HTML_USE_XHTML, "", ""), bf.EXTENSION_TABLES)
		if !ret.meta.Unsafe {
			ret.html = bm.UGCPolicy().SanitizeBytes(ret.html)
		}
	}

	ret.isarticle = true

	return
}

func entriesFromDir(fs http.FileSystem, path, activepath string) entries {
	var ret entries

	dir, err := fs.Open(path)
	if err != nil {
		log.Println(err)
		return nil
	}

	dirlist, err := dir.Readdir(0)
	if err != nil {
		log.Println(err)
	}
	dir.Close()

	for _, fi := range dirlist {
		if !fi.IsDir() {
			continue
		}
		entry, err := entryFromDir(fs, filepath.Join(path, fi.Name()), activepath)
		if err == nil {
			ret = append(ret, entry)
		}
	}

	sort.Sort(ret)
	return ret
}

type Page struct {
	Menu     []entries
	Articles entries
	Content  *entry
}

func PageFromDir(fs http.FileSystem, path string) Page {
	var p Page
	path = filepath.Clean(path)
	activepath := path

	// look for an article in current path
	if c, err := entryFromDir(fs, path, activepath); err != nil && c.IsArticle() {
		p.Content = &c
	}

	for {
		menu, articles := entriesFromDir(fs, path, activepath).Split()
		if len(menu) > 0 {
			sort.Sort(menu)
			p.Menu = append(p.Menu, menu)
		}
		if p.Articles == nil && len(articles) > 0 {
			sort.Sort(articles)
			p.Articles = articles
			if p.Content == nil {
				// Parse again with activepath set, to get the markdown
				cpath := p.Articles[0].Link()
				p.Content, _ = entryFromDir(fs, cpath, cpath) // TODO: check error
				p.Articles[0] = p.Content
			}
		}
		if path == "." || path == "/" {
			break
		}
		path = filepath.Dir(path)
	}

	// building from deepest path to /, need to reverse
	for l, r := 0, len(p.Menu)-1; l < r; l, r = l+1, r-1 {
		p.Menu[l], p.Menu[r] = p.Menu[r], p.Menu[l]
	}

	return p
}

// For debugging
func (p Page) Outline() string {
	buf := bytes.Buffer{}

	fmt.Fprintln(&buf, "Menu:")
	for level, ms := range p.Menu {
		for _, m := range ms {
			fmt.Fprintf(&buf, "%s%q", strings.Repeat("\t", level), m.Title())
			if m.Active() {
				fmt.Fprintf(&buf, " (active)")
			}
			fmt.Fprintln(&buf)
		}
	}

	fmt.Fprintln(&buf, "Articles:")
	for _, a := range p.Articles {
		fmt.Fprintf(&buf, "%q", a.Title())
		if a.Active() {
			fmt.Fprintf(&buf, " (active)")
		}
		fmt.Fprintln(&buf)
	}
	fmt.Fprintln(&buf, "Content:")
	if p.Content != nil {
		fmt.Fprint(&buf, p.Content.Title())
	}

	return buf.String()
}

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
}
