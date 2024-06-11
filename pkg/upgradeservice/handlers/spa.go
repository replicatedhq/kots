package handlers

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/replicatedhq/kots/web"
)

// SPAHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type SPAHandler struct {
}

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
func (h SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get the absolute path to prevent directory traversal
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		// if we failed to get the absolute path respond with a 400 bad request
		// and stop
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// prepend the path with the path to the static directory

	// check whether a file exists at the given path
	fsys, err := fs.Sub(web.Content, "dist")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = fs.Stat(fsys, filepath.Join(".", path)) // because ... fs.Sub seems to require this

	rr := r /// because the docs say to not modify request, and we might need to, so lets clone
	if os.IsNotExist(err) {
		rr = r.Clone(context.Background())
		// file does not exist, serve index.html
		rr.URL.Path = "/"
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static dir
	http.FileServer(http.FS(fsys)).ServeHTTP(w, rr)
}
