package glubcms

type Meta struct {
	Author string
	Date   GCTime
	Title  string
	Desc   string `json:",omitempt"`

	Priority int
	Hidden   bool      `json:",omitempty"`
	Unsafe   bool      `json:",omitempty"`
	IsIndex  bool      `json:",omitempty"`
	IsMenu   bool      `json:",omitempty"`
	Option   []Content `json:",omitempty"`
	Content  []Content `json:",omitempty"`
}

type Content struct {
	Type   string
	Inline string `json:",omitempty"`
	Path   string `json:",omitempty"`
}

const (
	ContentGallery  = "gallery"
	ContentHTML     = "html"
	ContentMarkdown = "markdown"
	ContentIndex    = "index"
	ContentSitemap  = "sitemap"

	OptionCSS      = "css"
	OptionJS       = "js"
	OptionTemplate = "template"
	OptionRedirect = "redirect"
	OptionStatic   = "static"
)
