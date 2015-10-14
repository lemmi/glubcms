package glubcms

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"testing"
)

func TestArticlePage(t *testing.T) {
	p := PageFromDir(http.Dir("example_page/pages"), "menu1/article1")
	fmt.Println(p.Outline())
}
func TestMenuPage(t *testing.T) {
	p := PageFromDir(http.Dir("example_page/pages"), "menu1/")
	fmt.Println(p.Outline())
}
func TestLandingPage(t *testing.T) {
	p := PageFromDir(http.Dir("example_page/pages"), "")
	fmt.Println(p.Outline())
}
func TestEntry(t *testing.T) {
	e := entryFromDir(http.Dir("example_page/pages"), "menu1/article1", "menu1/article1").(*entry)
	fmt.Printf("%+v", e)
	fmt.Printf("%+v", string(e.HTML()))
}
func TestTemplate(t *testing.T) {
	tmpl := template.Must(template.ParseGlob("example_page/templates/*tmpl"))
	p := PageFromDir(http.Dir("example_page/pages"), "menu1/article1")
	fmt.Println(tmpl.Execute(os.Stdout, p))
}
