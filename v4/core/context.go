package core

import (
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/go-raptor/raptor/v4/errs"
)

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
	defaultIndent = "  "
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
	// Fall back to legacy behavior
	if ip := c.request.Header.Get(HeaderXForwardedFor); ip != "" {
		i := strings.IndexAny(ip, ",")
		if i > 0 {
			xffip := strings.TrimSpace(ip[:i])
			xffip = strings.TrimPrefix(xffip, "[")
			xffip = strings.TrimSuffix(xffip, "]")
			return xffip
		}
		return ip
	}
	if ip := c.request.Header.Get(HeaderXRealIP); ip != "" {
		ip = strings.TrimPrefix(ip, "[")
		ip = strings.TrimSuffix(ip, "]")
		return ip
	}
	ra, _, _ := net.SplitHostPort(c.request.RemoteAddr)
	return ra
}

func (c *Context) Path() string {
	return c.path
}

func (c *Context) Param(name string) string {
	return c.request.PathValue(name)
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

func (c *Context) Bind(i interface{}) error {
	return c.core.Binder.Bind(i, c)
}

// TODO:
/*
func (c *Context) Validate(i interface{}) error {
	if c.echo.Validator == nil {
		return ErrValidatorNotRegistered
	}
	return c.echo.Validator.Validate(i)
}
*/

func (c *Context) String(code int, s string) (err error) {
	return c.Blob(code, MIMETextPlainCharsetUTF8, []byte(s))
}

func (c *Context) jsonPBlob(code int, callback string, i interface{}) (err error) {
	indent := ""
	if _, pretty := c.QueryParams()["pretty"]; pretty {
		indent = defaultIndent
	}
	c.writeContentType(MIMEApplicationJavaScriptCharsetUTF8)
	c.response.WriteHeader(code)
	if _, err = c.response.Write([]byte(callback + "(")); err != nil {
		return
	}
	if err = JSONSerialize(c, i, indent); err != nil {
		return
	}
	if _, err = c.response.Write([]byte(");")); err != nil {
		return
	}
	return
}

func (c *Context) json(code int, i interface{}, indent string) error {
	c.writeContentType(MIMEApplicationJSON)
	c.response.Status = code
	return JSONSerialize(c, i, indent)
}

func (c *Context) JSON(code int, i interface{}) (err error) {
	indent := ""
	if _, pretty := c.QueryParams()["pretty"]; pretty {
		indent = defaultIndent
	}
	return c.json(code, i, indent)
}

func (c *Context) JSONPretty(code int, i interface{}, indent string) (err error) {
	return c.json(code, i, indent)
}

func (c *Context) JSONBlob(code int, b []byte) (err error) {
	return c.Blob(code, MIMEApplicationJSON, b)
}

func (c *Context) JSONP(code int, callback string, i interface{}) (err error) {
	return c.jsonPBlob(code, callback, i)
}

func (c *Context) JSONPBlob(code int, callback string, b []byte) (err error) {
	c.writeContentType(MIMEApplicationJavaScriptCharsetUTF8)
	c.response.WriteHeader(code)
	if _, err = c.response.Write([]byte(callback + "(")); err != nil {
		return
	}
	if _, err = c.response.Write(b); err != nil {
		return
	}
	_, err = c.response.Write([]byte(");"))
	return
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
	c.response.WriteHeader(http.StatusNoContent)
	return nil
}

func (c *Context) NotFound() error {
	c.response.WriteHeader(http.StatusNotFound)
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
	if len(status) == 0 {
		status = append(status, http.StatusOK)
	}
	c.JSON(status[0], data)
	return nil
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

func (c *Context) Init(r *http.Request, w http.ResponseWriter, controller, action, path string, store map[string]interface{}) {
	c.controller = controller
	c.action = action
	c.path = path
	c.request = r
	c.response.init(w)
	c.query = nil
	c.handler = nil
	maps.Copy(c.store, store)
}

func (c *Context) Reset() {
	clear(c.store)
}
