package glubcms

import (
	"net/http"
	"path/filepath"
)

// The StaticHandler behaves like http.ServeContent without directoy listings.
// It also implements the http.Filesystem interface.
type StaticHandler struct {
	fs     http.FileSystem
	prefix string
}

// Serve the file requestet by r. Error 404 on directory access.
func (sh StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, err := sh.Open(r.URL.Path)
	if err != nil {
		http.Error(w, r.URL.Path, http.StatusNotFound)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		http.Error(w, r.URL.Path, http.StatusInternalServerError)
		return
	}
	if stat.IsDir() {
		http.Error(w, r.URL.Path, http.StatusNotFound)
		return
	}
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
}

// Return a new StaticHandler with new root directory.
func (sh StaticHandler) Cd(path string) StaticHandler {
	path = filepath.Clean(path)
	sh.prefix = filepath.Join(sh.prefix, path)
	return sh
}

// Implement the http.Filesystem interface.
func (sh StaticHandler) Open(name string) (http.File, error) {
	name = filepath.Clean(name)
	return sh.fs.Open(filepath.Clean(filepath.Join(sh.prefix, name)))
}

// Serves all files from fs.
func NewStaticHandler(fs http.FileSystem) StaticHandler {
	return StaticHandler{fs: fs}
}
