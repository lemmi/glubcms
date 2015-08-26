package glubcms

import (
	"fmt"
	"html/template"
	"os"
	"testing"
)

func TestArticlePage(t *testing.T) {
	p := PageFromDir("example_page", "menu1/article1")
	fmt.Println(p.Outline())
}
func TestMenuPage(t *testing.T) {
	p := PageFromDir("example_page", "menu1/")
	fmt.Println(p.Outline())
}
func TestLandingPage(t *testing.T) {
	p := PageFromDir("example_page", "")
	fmt.Println(p.Outline())
}
func TestEntry(t *testing.T) {
	e := entryFromDir("example_page/", "menu1/article1", "menu1/article1").(*entry)
	fmt.Printf("%+v", e)
	fmt.Printf("%+v", string(e.HTML()))
}
func TestTemplate(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("page_template.html"))
	p := PageFromDir("example_page", "menu1/article1")
	fmt.Println(tmpl.Execute(os.Stdout, p))
}
