package glubcms

import (
	"html/template"
	"io/ioutil"
	"os"
	"testing"
)

func TestPage(t *testing.T) {
	p := Page{
		Menu: [][]MenuEntry{
			{
				{Entry{title: "Menu 11"}},
				{Entry{title: "Menu 12"}},
				{Entry{title: "Menu 13", active: true}},
			},
			{
				{Entry{title: "Menu 11", active: true}},
				{Entry{title: "Menu 12"}},
			},
		},
		Articles: []Article{
			{Entry: Entry{title: "Article 1", active: true}, html: []byte{}},
			{Entry: Entry{title: "Article 2"}, html: []byte{}},
			{Entry: Entry{title: "Article 3"}, html: []byte{}},
		},
	}

	tmpl := template.Must(template.ParseFiles("page_template.html"))
	page := p.RenderHTML(tmpl)
	os.Stdout.Write(page)
	ioutil.WriteFile("/tmp/page.html", page, 0600)
}
