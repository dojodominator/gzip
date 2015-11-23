package gzip

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-martini/martini"
)

const (
	HeaderAcceptEncoding  = "Accept-Encoding"
	HeaderContentEncoding = "Content-Encoding"
	HeaderContentLength   = "Content-Length"
	HeaderContentType     = "Content-Type"
	HeaderVary            = "Vary"
)

var serveGzip = func(w http.ResponseWriter, r *http.Request, c martini.Context) {
	if !strings.Contains(r.Header.Get(HeaderAcceptEncoding), "gzip") {
		return
	}

	gzw := gzipResponseWriter{
		ResponseWriter: w.(martini.ResponseWriter),
		wroteHeader:    false,
	}
	c.MapTo(gzw, (*http.ResponseWriter)(nil))

	c.Next()

	// delete content length after we know we have been written to
	if gzw.wroteHeader {
		gzw.Header().Del("Content-Length")
		gzw.w.Close()
	}
}

// All returns a Handler that adds gzip compression to all requests
func All() martini.Handler {
	return serveGzip
}

type gzipResponseWriter struct {
	w *gzip.Writer
	martini.ResponseWriter
	wroteHeader bool
}

func (grw gzipResponseWriter) Write(p []byte) (int, error) {
	//Don't do anything if this write attempt has 0 bytes
	if p == nil || len(p) == 0 {
		return 0, nil
	} else if !grw.wroteHeader {
		//Write the content headers before the first write
		grw.Header().Set(HeaderContentEncoding, "gzip")
		grw.Header().Set(HeaderVary, HeaderAcceptEncoding)

		//Instantiate gzip writer on first write
		//This ensures that the response body is empty if nothing is ever
		//written to it
		grw.w = gzip.NewWriter(grw.ResponseWriter)

		grw.wroteHeader = true
	}
	if len(grw.Header().Get(HeaderContentType)) == 0 {
		grw.Header().Set(HeaderContentType, http.DetectContentType(p))
	}

	return grw.w.Write(p)
}

func (grw gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := grw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the ResponseWriter doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}
