package glubcms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/lemmi/glubcms/backend"
)

const (
	PSep = string(os.PathSeparator)
)

var (
	ErrHidden = errors.New("Hidden")
)

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
	Author      string
	Date        GCTime
	Title       string
	Priority    int
	Hidden      bool `json:",omitempty"`
	Unsafe      bool `json:",omitempty"`
	IsIndex     bool `json:",omitempty"`
	ExtraStyle  []string
	ExtraScript []string
}

func entryFromMeta(fs backend.Backend, path string) (Entry, error) {
	ret := Entry{}
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

func entryFromDir(fs backend.Backend, path, activepath string) (ret Entry, err error) {
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

func entriesFromDir(fs backend.Backend, path, activepath string) (Entries, error) {
	var ret Entries

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

	sort.Slice(ret, ret.Less)
	return ret, nil
}

func latestOf(modtime time.Time, e *Entry) time.Time {
	if e != nil && modtime.After(e.Date()) {
		return modtime
	}
	return modtime
}

type Page struct {
	Menu     []Entries
	Articles Entries
	Content  *Entry
	Index    *Entry
	ModTime  time.Time
}

func PageFromDir(fs backend.Backend, path string) (Page, error) {
	var p Page
	path = filepath.Clean(path)
	activepath := path

	for {
		es, err := entriesFromDir(fs, path, activepath)
		if err != nil {
			return Page{}, errors.Wrap(err, "entriesFromDir")
		}

		menu, articles := SplitEntries(es)
		if len(menu) > 0 {
			sort.Slice(menu, menu.Less)
			p.Menu = append(p.Menu, menu)
		}

		if p.Articles == nil && len(articles) > 0 {
			sort.Slice(articles, articles.Less)
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
