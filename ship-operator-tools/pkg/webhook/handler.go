package webhook

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type Handler struct {
	// must already exist
	dir    string
	logger log.Logger
}

func NewHandler(dir string, logger log.Logger) *Handler {
	return &Handler{dir, logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	errs := level.Error(log.With(h.logger))

	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		errs.Log("event", "parse.mediatype", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		http.Error(w, "Unsupported Content Type", http.StatusUnsupportedMediaType)
		return
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			errs.Log("event", "next.part", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		switch p.FormName() {
		case tarFormName:
			f, err := os.Create(filepath.Join(h.dir, p.FileName()))
			if err != nil {
				errs.Log("event", "create.tar.file", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if _, err := io.Copy(f, p); err != nil {
				if err == io.EOF {
					return
				}
				errs.Log("event", "copy.tar.part", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		case jsonFormName:
			f, err := os.Create(filepath.Join(h.dir, p.FileName()))
			if err != nil {
				errs.Log("event", "create.json.file", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if _, err := io.Copy(f, p); err != nil {
				if err == io.EOF {
					return
				}
				errs.Log("event", "copy.json.part", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		case stateFormName:
			return
		default:
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	}
}
