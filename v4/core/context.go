package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/go-raptor/raptor/v4/errs"
)

var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024))
	},
}

type Context struct {
	core     *Core
	request  *http.Request
	response *Response
	path     string
	query    url.Values

	store map[string]interface{}
	lock  sync.RWMutex

	controller string
	action     string
	handler    HandlerFunc
}

const (
	defaultMemory = 32 << 20 // 32 MB
)

func NewContext(c *Core, r *http.Request, w http.ResponseWriter) *Context {
	return &Context{
		request:  r,
		response: NewResponse(w),
		store:    make(map[string]interface{}),
		core:     c,
		handler:  nil,
	}
}

func (c *Context) writeContentType(value string) {
	header := c.Response().Header()
	if header.Get(HeaderContentType) == "" {
		header.Set(HeaderContentType, value)
	}
}

func (c *Context) Controller() string {
	return c.controller
}

func (c *Context) Action() string {
	return c.action
}

func (c *Context) Core() *Core {
	return c.core
}

func (c *Context) Request() *http.Request {
	return c.request
}

func (c *Context) SetRequest(r *http.Request) {
	c.request = r
}

func (c *Context) Response() *Response {
	return c.response
}

func (c *Context) SetResponse(r *Response) {
	c.response = r
}

func (c *Context) IsWebSocket() bool {
	upgrade := c.request.Header.Get(HeaderUpgrade)
	return strings.EqualFold(upgrade, "websocket")
}

func (c *Context) RealIP() string {
	if c.core != nil && c.core.IPExtractor != nil {
		return c.core.IPExtractor(c.request)
	}

	for _, header := range []string{HeaderXForwardedFor, HeaderXRealIP} {
		if ip := c.request.Header.Get(header); ip != "" {
			ip = strings.TrimSpace(strings.SplitN(ip, ",", 2)[0])
			ip = strings.Trim(ip, "[]")
			if ip != "" {
				return ip
			}
		}
	}

	host, _, _ := net.SplitHostPort(c.request.RemoteAddr)
	return host
}

func (c *Context) Path() string {
	return c.path
}

func (c *Context) Param(name string) string {
	return c.request.PathValue(name)
}

func (c *Context) Bind(v any) error {
	return json.NewDecoder(c.request.Body).Decode(v)
}

func (c *Context) Query() url.Values {
	if c.query == nil {
		c.query = c.request.URL.Query()
	}
	return c.query
}

func (c *Context) QueryParam(name string) string {
	if c.query == nil {
		c.query = c.request.URL.Query()
	}
	return c.query.Get(name)
}

func (c *Context) QueryParams() url.Values {
	if c.query == nil {
		c.query = c.request.URL.Query()
	}
	return c.query
}

func (c *Context) QueryString() string {
	return c.request.URL.RawQuery
}

func (c *Context) FormValue(name string) string {
	return c.request.FormValue(name)
}

func (c *Context) FormParams() (url.Values, error) {
	if strings.HasPrefix(c.request.Header.Get(HeaderContentType), MIMEMultipartForm) {
		if err := c.request.ParseMultipartForm(defaultMemory); err != nil {
			return nil, err
		}
	} else {
		if err := c.request.ParseForm(); err != nil {
			return nil, err
		}
	}
	return c.request.Form, nil
}

func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	f, fh, err := c.request.FormFile(name)
	if err != nil {
		return nil, err
	}
	f.Close()
	return fh, nil
}

func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.request.ParseMultipartForm(defaultMemory)
	return c.request.MultipartForm, err
}

func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
}

func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response(), cookie)
}

func (c *Context) Cookies() []*http.Cookie {
	return c.request.Cookies()
}

func (c *Context) Get(key string) interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.store[key]
}

func (c *Context) Set(key string, val interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.store[key] = val
}

func (c *Context) String(code int, s string) (err error) {
	return c.Blob(code, MIMETextPlainCharsetUTF8, []byte(s))
}

func (c *Context) JSON(code int, i interface{}) error {
	c.writeContentType(MIMEApplicationJSON)
	c.response.Status = code

	if b, ok := i.([]byte); ok {
		c.response.Header().Set(HeaderContentLength, strconv.Itoa(len(b)))
		_, err := c.response.Write(b)
		return err
	}

	buf := jsonBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer jsonBufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(i); err != nil {
		return err
	}

	c.response.Header().Set(HeaderContentLength, strconv.Itoa(buf.Len()))
	_, err := c.response.Write(buf.Bytes())
	return err
}

func (c *Context) JSONBlob(code int, b []byte) error {
	return c.Blob(code, MIMEApplicationJSON, b)
}

func (c *Context) Blob(code int, contentType string, b []byte) (err error) {
	c.writeContentType(contentType)
	c.response.WriteHeader(code)
	_, err = c.response.Write(b)
	return
}

func (c *Context) Stream(code int, contentType string, r io.Reader) (err error) {
	c.writeContentType(contentType)
	c.response.WriteHeader(code)
	_, err = io.Copy(c.response, r)
	return
}

func (c *Context) Attachment(file, name string) error {
	return c.contentDisposition(file, name, "attachment")
}

func (c *Context) Inline(file, name string) error {
	return c.contentDisposition(file, name, "inline")
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func (c *Context) contentDisposition(file, name, dispositionType string) error {
	c.response.Header().Set(HeaderContentDisposition, fmt.Sprintf(`%s; filename="%s"`, dispositionType, quoteEscaper.Replace(name)))
	return c.File(file)
}

func (c *Context) NoContent() error {
	return c.Status(http.StatusNoContent)
}

func (c *Context) NotFound() error {
	return c.Status(http.StatusNotFound)
}

func (c *Context) Status(code int) error {
	c.response.WriteHeader(code)
	return nil
}

func (c *Context) Redirect(code int, url string) error {
	if code < 300 || code > 308 {
		return errs.ErrInvalidRedirectCode
	}
	c.response.Header().Set(HeaderLocation, url)
	c.response.WriteHeader(code)
	return nil
}

func (c *Context) Data(data interface{}, status ...int) error {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}
	return c.JSON(code, data)
}

func (c *Context) Error(err error) {
	if e, ok := err.(*errs.Error); ok {
		c.Data(e, e.Code)
		return
	}
	c.Data(errs.NewErrorInternal(err.Error()), http.StatusInternalServerError)
}

func (c *Context) Handler() HandlerFunc {
	return c.handler
}

func (c *Context) ResetAndInit(r *http.Request, w http.ResponseWriter, controller, action, path string, store map[string]interface{}) {
	c.controller = controller
	c.action = action
	c.path = path
	c.request = r
	c.response.init(w)
	c.query = nil
	c.handler = nil

	clear(c.store)
	if len(store) > 0 {
		maps.Copy(c.store, store)
	}
}
