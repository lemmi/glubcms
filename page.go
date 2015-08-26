package glubcms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	bf "github.com/russross/blackfriday"
)

type Entry interface {
	Active() bool
	Author() string
	Date() time.Time
	HTML() template.HTML
	IsArticle() bool
	Link() string
	Title() string
}

type Entries []Entry

func (e Entries) Len() int {
	return len(e)
}
func (e Entries) Less(i, j int) bool {
	return e[i].Date().After(e[j].Date())
}
func (e Entries) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
func (e Entries) Split() (Menu, Articles Entries) {
	for _, v := range e {
		if v.IsArticle() {
			Articles = append(Articles, v)
		} else {
			Menu = append(Menu, v)
		}
	}

	return
}

type mtime time.Time

func (t *mtime) UnmarshalJSON(b []byte) error {
	layout := "2006-01-02 15:04"
	tmp, err := time.Parse(layout, strings.Trim(string(b), "\""))
	*t = mtime(tmp)
	return err
}

type Meta struct {
	Author string
	Date   mtime
	Title  string
}

type entry struct {
	meta      Meta
	active    bool
	html      []byte
	isarticle bool
	link      url.URL
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
func (e entry) Title() string {
	return e.meta.Title
}

func entryFromMeta(path string) (*entry, error) {
	ret := entry{}
	f, err := os.Open(filepath.Join(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&ret.meta)
	return &ret, err
}

func entryFromDir(prefix, path, activepath string) Entry {
	ret, err := entryFromMeta(filepath.Join(prefix, path, "meta.json"))
	if err != nil {
		log.Println(err)
		return nil
	}

	ret.link = url.URL{Path: path}
	if strings.HasPrefix(activepath, path) {
		ret.active = true
	}

	md, err := os.Open(filepath.Join(prefix, path, "article.md"))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println(err)
		}
		return ret
	}
	defer md.Close()

	// only attempt to convert the markdown if it's the active path
	if activepath == path {
		b, err := ioutil.ReadAll(md)
		if err != nil {
			log.Println(err)
			return ret
		}
		ret.html = bf.MarkdownBasic(b)
	}

	ret.isarticle = true

	return ret
}

func entriesFromDir(prefix, path, activepath string) Entries {
	var ret Entries

	dir, err := os.Open(filepath.Join(prefix, path))
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
		entry := entryFromDir(prefix, filepath.Join(path, fi.Name()), activepath)
		if entry != nil {
			ret = append(ret, entry)
		}
	}

	sort.Sort(ret)
	return ret
}

type Page struct {
	Menu     []Entries
	Articles Entries
	Content  Entry
}

func PageFromDir(prefix, path string) Page {
	var p Page
	path = filepath.Clean(path)
	activepath := path

	// look for an article in current path
	if c := entryFromDir(prefix, path, activepath); c.IsArticle() {
		p.Content = c
	}

	for {
		menu, articles := entriesFromDir(prefix, path, activepath).Split()
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
				p.Content = entryFromDir(prefix, cpath, cpath)
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
