package brotli

import (
	"net/http"
	"strings"

	"gopkg.in/kothar/brotli-go.v0/enc"
)

type middleware struct{ *enc.BrotliParams }

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

	wr := &writer{brParams: m.BrotliParams, ResponseWriter: w}
	defer wr.Close()

	next(wr, r)
}

type writer struct {
	http.ResponseWriter
	brParams      *enc.BrotliParams
	brWriter      *enc.BrotliWriter
	mungedHeaders bool
}

func (w *writer) Close() error {
	if w.brWriter != nil {
		return w.brWriter.Close()
	}

	return nil
}

func (w *writer) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *writer) Push(target string, opts *http.PushOptions) error {
	return w.ResponseWriter.(http.Pusher).Push(target, opts)
}

func (w *writer) Write(b []byte) (int, error) {
	w.mungeHeaders()

	if w.brWriter != nil {
		return w.brWriter.Write(b)
	}

	return w.ResponseWriter.Write(b)
}

func (w *writer) WriteHeader(code int) {
	w.mungeHeaders()
	w.ResponseWriter.WriteHeader(code)
}

func (w *writer) mungeHeaders() {
	if w.mungedHeaders {
		return
	}

	w.mungedHeaders = true

	headers := w.Header()

	contentType := headers.Get("Content-Type")
	if i := strings.IndexByte(contentType, ';'); i != -1 {
		contentType = contentType[:i]
	}

	// Only compress content types that make sense and aren't already encoded.
	switch contentType {
	case "application/json", "image/svg+xml", "text/css", "text/html", "text/plain":
		if headers.Get("Content-Encoding") == "" {
			w.brWriter = enc.NewBrotliWriter(w.brParams, w.ResponseWriter)

			headers.Del("Content-Length")
			headers.Set("Content-Encoding", "br")
			headers.Set("Vary", "Accept-Encoding")
		}
	}
}
