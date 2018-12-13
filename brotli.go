package brotli

import (
	"net/http"
	"strings"

	"gopkg.in/kothar/brotli-go.v0/enc"
)

type middleware struct {
	*enc.BrotliParams
}

func New(quality int) *middleware {
	params := enc.NewBrotliParams()

	params.SetQuality(quality)

	return &middleware{params}
}

func (m *middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Skip compression if the client doesn't accept br encoding.
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
		next(w, r)
		return
	}

	brWriter := enc.NewBrotliWriter(m.BrotliParams, w)
	defer brWriter.Close()

	next(&writer{brWriter: brWriter, ResponseWriter: w}, r)
}

type writer struct {
	http.ResponseWriter
	brWriter *enc.BrotliWriter
	compress bool
}

// WriteHeader checks if encoding makes sense and if so changes some headers.
func (w *writer) WriteHeader(code int) {
	headers := w.Header()

	contentType := headers.Get("Content-Type")
	if i := strings.IndexByte(contentType, ';'); i != -1 {
		contentType = contentType[:i]
	}

	// Only compress content types that make sense and aren't already encoded.
	switch contentType {
	case "application/json", "image/svg+xml", "text/css", "text/html", "text/plain":
		w.compress = headers.Get("Content-Encoding") == ""
	}

	if w.compress {
		headers.Del("Content-Length")
		headers.Set("Content-Encoding", "br")
		headers.Set("Vary", "Accept-Encoding")
	}

	w.ResponseWriter.WriteHeader(code)
}

// Write encodes on not depending on what WriteHeader decided. This means this
// middleware only supports encoding with explicit WriteHeader calls.
func (w *writer) Write(b []byte) (int, error) {
	if w.compress {
		return w.brWriter.Write(b)
	}

	return w.ResponseWriter.Write(b)
}
