package glubcms

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	active bool
	author string
	date   time.Time
	html   []byte
	link   url.URL
	title  string
}

func (e Entry) Title() string {
	return e.title
}
func (e Entry) Link() string {
	return e.link.String()
}
func (e Entry) IsActive() bool {
	return e.active
}
func (a Entry) Author() string {
	return a.author
}
func (e Entry) Date() time.Time {
	return e.date
}
func (e Entry) HTML() []byte {
	return e.html
}
func (e Entry) IsArticle() bool {
	return e.HTML() != nil
}

type PTreeList []*PTree
type PTree struct {
	Entry
	Children PTreeList
}

func (ptl PTreeList) Filter(filterfunc func(*PTree) bool) PTreeList {
	var ret PTreeList
	for _, e := range ptl {
		if filterfunc(e) {
			ret = append(ret, e)
		}
	}
	return ret
}

func (pt *PTree) Articles() PTreeList {
	return pt.Children.Filter((*PTree).IsArticle)
}

//func (pt *PTree) MenuEntries() []*PTree {
//	return pt.Children.Filter((*PTree).IsMenu)
//}
func (pt *PTree) Active() *PTree {
	ret := pt.Children.Filter((*PTree).IsActive)
	switch len(ret) {
	case 1:
		return ret[0]
	case 0:
		return nil
	default:
		log.Fatalf("Multiple active children. %+v", ret)
		return nil
	}
}
func (pt *PTree) IsLeaf() bool {
	return pt.Children == nil
}

func splitfirstdir(s string) (string, string) {
	if len(s) == 0 {
		return "", ""
	}
	if s[0] == os.PathSeparator {
		return splitfirstdir(s[1:])
	}
	split := strings.SplitN(s, string(os.PathSeparator), 2)
	switch len(split) {
	case 2:
		return split[0], split[1]
	case 1:
		return split[0], ""
	default:
		return "", ""
	}
}

func PTreeFromDir(prefix string, path string) *PTree {
	var ret *PTree
	var dirname string
	path = filepath.Clean(path)
	for {
		dirname, path = splitfirstdir(path)
		if dirname == "" {
			break
		}

	}
	return ret
}

func pTreeFromDir(prefix string, path string) *PTree {
	var ret PTree

	metafile, err := os.Open(filepath.Join(path, "meta.json"))
	if err != nil {
		log.Fatal(err)
	}
	err = json.NewDecoder(metafile).Decode(&ret)
	metafile.Close()
	if err != nil {
		log.Fatal(err)
	}
	ret.link = url.URL{}

	paths, err := filepath.Glob(filepath.Join(path, "*", "meta.json"))
	if err != nil {
		log.Fatal(err)
	}
	for _, path := range paths {
		path = filepath.Dir(path)
		newChild := pTreeFromDir(prefix, path)
		if newChild != nil {
			ret.Children = append(ret.Children, newChild)
		}
	}

	if ret.Children != nil {
	}

	return &ret
}

//type Page struct {
//	Menu     [][]MenuEntry
//	Articles []Article
//}
//
//func (p *Page) RenderHTML(t *template.Template) []byte {
//	b := bytes.Buffer{}
//	fmt.Println(t.Execute(&b, p))
//	return b.Bytes()
//}
//
//func (p *Page) Path() []MenuEntry {
//	var ret []MenuEntry
//level:
//	for _, menuLevel := range p.Menu {
//		for _, m := range menuLevel {
//			if m.Active() {
//				ret = append(ret, m)
//				continue level
//			}
//		}
//	}
//
//	return ret
//}
//
//func (p *Page) Title(sep ...string) string {
//	for _, a := range p.Articles {
//		if a.Active() {
//			return a.Title()
//		}
//	}
//
//	var menutitles []string
//	for _, m := range p.Path() {
//		menutitles = append(menutitles, m.Title())
//	}
//
//	if len(sep) == 0 {
//		sep = " - "
//	}
//	return strings.Join(menutitles, sep)
//}
