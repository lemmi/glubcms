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
	"sync"
	"time"

	"github.com/pkg/errors"

	bm "github.com/microcosm-cc/bluemonday"
	bf "github.com/russross/blackfriday"
)

const (
	PSep = string(os.PathSeparator)
)

var (
	ErrHidden = errors.New("Hidden")
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
	switch {
	case e[i].meta.IsIndex && !e[j].meta.IsIndex:
		return false
	case !e[i].meta.IsIndex && e[j].meta.IsIndex:
		return true
	case e[i].Priority() != e[j].Priority():
		return e[i].Priority() > e[j].Priority()
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
	if err != nil {
		err = errors.Wrapf(err, "time.Parse failed on %q", b)
	}
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
	IsIndex  bool `json:",omitempty"`
}

type ContentRenderer interface {
	Render() ([]byte, error)
}

type entry struct {
	meta       Meta
	active     bool
	html       []byte
	isarticle  bool
	link       url.URL
	next       *entry
	prev       *entry
	fs         http.FileSystem
	md_path    string
	once       sync.Once
	renderHTML ContentRenderer
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
func (e *entry) HTML() template.HTML {
	e.once.Do(func() {
		var err error
		e.html, err = e.renderHTML.Render()
		if err != nil {
			//TODO make errorpage
			log.Println(err)
		}
	})
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
	if e.next != nil {
		return e.next
	}
	return nil
}
func (e entry) Prev() Entry {
	if e.prev != nil {
		return e.prev
	}
	return nil
}
func (e entry) IsIndex() bool {
	return e.meta.IsIndex
}
func (e *entry) Context(c int) entries {
	next := e
	prev := e
	n := 1

	for {
		moved := false

		if n < c && next.next != nil {
			moved = true
			next = next.next
			n++
		}
		if n < c && prev.prev != nil {
			moved = true
			prev = prev.prev
			n++
		}

		if n == c || moved == false {
			break
		}
	}

	ret := make(entries, 0, n)
	t := next
	for n > 0 {
		ret = append(ret, *t)
		t = t.prev
		n--
	}
	return ret
}

func entryFromMeta(fs http.FileSystem, path string) (entry, error) {
	ret := entry{}
	f, err := fs.Open(filepath.Join(path))
	if err != nil {
		return ret, errors.Wrapf(err, "file open failed: %q", path)
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&ret.meta)
	if err != nil {
		err = errors.Wrapf(err, "Parsing json in: %q", path)
	}
	return ret, err
}

func entryFromDir(fs http.FileSystem, path, activepath string) (ret entry, err error) {
	ret, err = entryFromMeta(fs, filepath.Join(path, "meta.json"))
	if err != nil {
		err = errors.Wrapf(err, "Cannot open meta.json in: %q", path)
		return ret, err
	}

	// skip hidden folders, unless directly asked for
	if ret.meta.Hidden && activepath != path {
		return ret, ErrHidden
	}

	ret.link = url.URL{Path: path}
	// .Dir() removes trailing slashes, this prevents
	// /a/b/ beeing seen as prefix from /a/bc/
	if strings.HasPrefix(activepath+PSep, path+PSep) {
		ret.active = true
	}

	md_path := filepath.Join(path, "article.md")
	md, err := fs.Open(md_path)
	// Decide wether this is menu or article
	// TODO add entry to metadata and remove implicit menus
	if err != nil {
		if !os.IsNotExist(err) {
			return ret, errors.Wrapf(err, "Cannot open article.md in: %q", path)
		}
		return ret, nil
	}
	md.Close()

	ret.isarticle = true
	ret.renderHTML = articleRenderer{
		fs:      fs,
		md_path: md_path,
		unsafe:  ret.meta.Unsafe,
	}

	return ret, nil
}

type articleRenderer struct {
	fs      http.FileSystem
	md_path string
	unsafe  bool
}

func (a articleRenderer) Render() ([]byte, error) {
	md, err := a.fs.Open(a.md_path)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot open markdown file: %q", a.md_path)
	}
	defer md.Close()

	b, err := ioutil.ReadAll(md)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read markdown file: %q", a.md_path)
	}
	html := bf.Markdown(b,
		CorrectHeadingLevel{
			ImageAltTitleCopy{
				bf.HtmlRenderer(0, "", ""),
			},
		}, bf.EXTENSION_TABLES)
	if !a.unsafe {
		html = bm.UGCPolicy().SanitizeBytes(html)
	}
	return html, nil
}

func entriesFromDir(fs http.FileSystem, path, activepath string) (entries, error) {
	var ret entries

	dir, err := fs.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot open directory: %q", path)
	}

	dirlist, err := dir.Readdir(0)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read directory: %q", path)
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
	return ret, nil
}

func latestOf(modtime time.Time, e *entry) time.Time {
	if e != nil && modtime.After(e.Date()) {
		return modtime
	}
	return modtime
}

type Page struct {
	Menu     []entries
	Articles entries
	Content  *entry
	Index    *entry
	ModTime  time.Time
}

func PageFromDir(fs http.FileSystem, path string) (Page, error) {
	var p Page
	path = filepath.Clean(path)
	activepath := path

	for {
		es, err := entriesFromDir(fs, path, activepath)
		if err != nil {
			return Page{}, errors.Wrap(err, "entriesFromDir")
		}

		menu, articles := es.Split()
		if len(menu) > 0 {
			sort.Sort(menu)
			p.Menu = append(p.Menu, menu)
		}

		if p.Articles == nil && len(articles) > 0 {
			sort.Sort(articles)
			p.Articles = articles

			// don't list the index page as article
			for p.Articles[len(p.Articles)-1].meta.IsIndex {
				p.Index = &p.Articles[len(p.Articles)-1]
				p.Articles = p.Articles[:len(p.Articles)-1]
			}

			// link next/prev pointers
			for i := 1; i < len(p.Articles); i++ {
				p.Articles[i].next = &p.Articles[i-1]
				p.Articles[i-1].prev = &p.Articles[i]
			}

			// set the active article as content
			for i := range articles {
				if articles[i].link.Path == activepath {
					p.Content = &articles[i]
					break
				}
			}

			// if no exact match, default to first article
			if p.Content == nil {
				p.Content = &p.Articles[0]
				p.Content.active = true
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

	// set ModTime to the latest of all values

	p.ModTime = latestOf(p.ModTime, p.Index)
	p.ModTime = latestOf(p.ModTime, p.Content)
	for _, es := range p.Menu {
		for i := range es {
			p.ModTime = latestOf(p.ModTime, &es[i])
		}
	}
	for i := range p.Articles {
		p.ModTime = latestOf(p.ModTime, &p.Articles[i])
	}

	return p, nil
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
