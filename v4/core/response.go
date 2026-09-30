package core

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

type Response struct {
	Writer     http.ResponseWriter
	Status     int
	Size       int64
	Committed  bool
	controller *http.ResponseController
}

func NewResponse(w http.ResponseWriter) *Response {
	return &Response{Writer: w}
}

func (r *Response) rc() *http.ResponseController {
	if r.controller == nil {
		r.controller = http.NewResponseController(r.Writer)
	}
	return r.controller
}

func (r *Response) Header() http.Header {
	return r.Writer.Header()
}

func (r *Response) WriteHeader(code int) {
	if r.Committed {
		return
	}
	r.Status = code
	r.Writer.WriteHeader(code)
	r.Committed = true
}

func (r *Response) Write(b []byte) (n int, err error) {
	if !r.Committed {
		if r.Status == 0 {
			r.Status = http.StatusOK
		}
		r.WriteHeader(r.Status)
	}
	n, err = r.Writer.Write(b)
	r.Size += int64(n)
	return
}

var _ io.ReaderFrom = (*Response)(nil)

// ReadFrom lets io.Copy, and with it http.ServeContent, reach the writer's
// own ReadFrom, which net/http implements with sendfile. It forwards to
// Writer and never past it: a writer a middleware substituted (gzip, say)
// stays in the path, and without a ReadFrom of its own it gets plain writes.
func (r *Response) ReadFrom(src io.Reader) (n int64, err error) {
	if !r.Committed {
		if r.Status == 0 {
			r.Status = http.StatusOK
		}
		r.WriteHeader(r.Status)
	}
	n, err = io.Copy(r.Writer, src)
	r.Size += n
	return
}

func (r *Response) Flush() {
	r.rc().Flush()
}

func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.rc().Hijack()
}

func (r *Response) Unwrap() http.ResponseWriter {
	return r.Writer
}

func (r *Response) init(w http.ResponseWriter) {
	r.Writer = w
	r.Size = 0
	r.Status = http.StatusOK
	r.Committed = false
	r.controller = nil
}
