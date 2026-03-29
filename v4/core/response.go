package core

import (
	"bufio"
	"net"
	"net/http"
)

type Response struct {
	Writer    http.ResponseWriter
	Status    int
	Size      int64
	Committed bool
}

func NewResponse(w http.ResponseWriter) *Response {
	return &Response{Writer: w}
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

func (r *Response) Flush() {
	http.NewResponseController(r.Writer).Flush()
}

func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(r.Writer).Hijack()
}

func (r *Response) Unwrap() http.ResponseWriter {
	return r.Writer
}

func (r *Response) init(w http.ResponseWriter) {
	r.Writer = w
	r.Size = 0
	r.Status = http.StatusOK
	r.Committed = false
}
