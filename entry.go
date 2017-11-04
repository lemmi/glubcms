package glubcms

import (
	"html/template"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/lemmi/glubcms/backend"
)

type Entries []Entry

func (e Entries) Less(i, j int) bool {
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

type Entry struct {
	meta       Meta
	active     bool
	html       []byte
	isarticle  bool
	link       url.URL
	next       *Entry
	prev       *Entry
	fs         backend.Backend
	md_path    string
	once       sync.Once
	renderHTML ContentRenderer
}

func (e *Entry) Active() bool {
	return e.active
}
func (a *Entry) Author() string {
	return a.meta.Author
}
func (e *Entry) Date() time.Time {
	return time.Time(e.meta.Date)
}
func (e *Entry) ExtraStyle() []string {
	return e.meta.ExtraStyle
}
func (e *Entry) ExtraScript() []string {
	return e.meta.ExtraScript
}
func (e *Entry) HTML() template.HTML {
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
func (e *Entry) IsArticle() bool {
	return e.isarticle
}
func (e *Entry) Link() string {
	return e.link.String()
}
func (e *Entry) Priority() int {
	return e.meta.Priority
}
func (e *Entry) Title() string {
	return e.meta.Title
}
func (e *Entry) Next() *Entry {
	if e.next != nil {
		return e.next
	}
	return nil
}
func (e *Entry) Prev() *Entry {
	if e.prev != nil {
		return e.prev
	}
	return nil
}
func (e *Entry) IsIndex() bool {
	return e.meta.IsIndex
}
func (e *Entry) Context(c int) Entries {
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

	ret := make(Entries, 0, n)
	t := next
	for n > 0 {
		ret = append(ret, *t)
		t = t.prev
		n--
	}
	return ret
}

func SplitEntries(e Entries) (Menu, Articles Entries) {
	for _, v := range e {
		if v.IsArticle() {
			Articles = append(Articles, v)
		} else {
			Menu = append(Menu, v)
		}
	}

	return
}
