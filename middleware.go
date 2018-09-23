package roxy

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	loggerLogFormat  = "%s [%s] \"%s %s %s\" %d %d - %.3f ms\n"
	loggerTimeFormat = "02/Jan/2006 15:04:05 -0700"
)

// loggerResponseWriter is a wrapper on of a http.ResponseWriter,
// which captures HTTP Status Code and Content-Length.
//
// Note that loggerResponseWriter does not implement http.Flusher and / or
// http.Hijacker and even, if the underlying http.ResponseWriter does implement
// these interface, Flush() and Hijack() cannot not be used with it.
type loggerResponseWriter struct {
	http.ResponseWriter

	status int
	size   int
}

func (w *loggerResponseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.size += n

	return n, err
}

func (w *loggerResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

type middleware func(http.Handler) http.Handler

// chain creates a middleware chain applied to a http.HandlerFunc
func chain(f http.Handler, middleware ...middleware) http.Handler {
	for _, m := range middleware {
		f = m(f)
	}

	return f
}

// method middleware ensures that url can only be requested with a specific method,
// else returns a 405 method Not Allowed.
func method(m string) middleware {
	return func(f http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != m {
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}

			f.ServeHTTP(w, r)
		})
	}
}

// logger middleware logs all requests with its path and the time it took
// to process.
func logger(out io.Writer) middleware {
	if out == nil {
		out = os.Stdout
	}

	return func(f http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := &loggerResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			now := time.Now()

			f.ServeHTTP(l, r)

			d := float64(time.Since(now) / time.Millisecond)

			fmt.Fprintf(out, loggerLogFormat,
				r.RemoteAddr,                 // :remote-addr
				now.Format(loggerTimeFormat), // :datetime
				r.Method,                     // :method
				r.RequestURI,                 // :url
				r.Proto,                      // :http-version
				l.status,                     // :status
				l.size,                       // :content-length
				d) // :response-time ms
		})
	}
}
