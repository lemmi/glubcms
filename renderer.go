package glubcms

import (
	"io/ioutil"

	"github.com/lemmi/glubcms/backend"
	"github.com/pkg/errors"

	bm "github.com/microcosm-cc/bluemonday"
	bf "github.com/russross/blackfriday"
)

type ContentRenderer interface {
	Render() ([]byte, error)
}

type articleRenderer struct {
	fs      backend.Backend
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
		NewMdModifier(
			bf.HtmlRenderer(0, "", ""),
		), bf.EXTENSION_TABLES)
	if !a.unsafe {
		html = bm.UGCPolicy().SanitizeBytes(html)
	}
	return html, nil
}
