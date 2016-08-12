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
	Content  []Content `json:",omitempty"`
}

const (
	ContentArticle  = "article"
	ContentGallery  = "gallery"
	ContentSitemap  = "sitemap"
	ContentOverview = "overview"
	ContentRedirect = "redirect"
)

type Content struct {
	Type   string
	Inline string `json:",omitempty"`
	Path   string `json:",omitempty"`
}
