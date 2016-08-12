package glubcms

type ContentRenderer interface {
	Render(fs http.FileSystem, meta Meta, path string)
	// need context / page options -> extra css -> derive templates ...

}
