package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-raptor/raptor/v4/errs"
)

type Context struct {
	core     *Core
	request  *http.Request
	response *Response
	path     string
	query    url.Values

	store      map[string]any
	routeStore map[string]any

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
		core:     c,
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
	return c.core.IPExtractor(c.request)
}

func (c *Context) Path() string {
	return c.path
}

func (c *Context) Param(name string) string {
	return c.request.PathValue(name)
}

func (c *Context) Bind(v any) error {
	body := c.request.Body
	if max := c.core.Resources.Config.ServerConfig.MaxBodyBytes; max > 0 {
		body = http.MaxBytesReader(c.response.Writer, body, max)
	}
	return json.NewDecoder(body).Decode(v)
}

func (c *Context) Query() url.Values {
	if c.query == nil {
		c.query = c.request.URL.Query()
	}
	return c.query
}

func (c *Context) QueryParam(name string) string {
	return c.Query().Get(name)
}

func (c *Context) QueryParams() url.Values {
	return c.Query()
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

func (c *Context) Get(key string) any {
	if v, ok := c.store[key]; ok {
		return v
	}
	return c.routeStore[key]
}

func (c *Context) Set(key string, val any) {
	if c.store == nil {
		c.store = make(map[string]any)
	}
	c.store[key] = val
}

func (c *Context) String(code int, s string) (err error) {
	return c.Blob(code, MIMETextPlainCharsetUTF8, []byte(s))
}

func (c *Context) JSON(code int, i any) error {
	if b, ok := i.([]byte); ok {
		return c.JSONBlob(code, b)
	}
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	return c.Blob(code, MIMEApplicationJSON, b)
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
	if code < http.StatusMultipleChoices || code > http.StatusPermanentRedirect {
		return errs.ErrInvalidRedirectCode
	}
	c.response.Header().Set(HeaderLocation, url)
	c.response.WriteHeader(code)
	return nil
}

func (c *Context) Data(data any, status ...int) error {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}
	return c.JSON(code, data)
}

func (c *Context) Error(err error) {
	var e *errs.Error
	if !errors.As(err, &e) {
		e = errs.NewErrorInternal(err.Error())
	}
	if writeErr := c.Data(e, e.Code); writeErr != nil {
		c.core.Resources.Log.Error("Failed to write error response", "error", writeErr, "original", err)
	}
}

func (c *Context) Handler() HandlerFunc {
	return c.handler
}

func (c *Context) ResetAndInit(r *http.Request, w http.ResponseWriter, controller, action, path string, store map[string]any) {
	c.controller = controller
	c.action = action
	c.path = path
	c.request = r
	c.response.init(w)
	c.query = nil
	c.handler = nil
	c.routeStore = store
	if len(c.store) > 0 {
		clear(c.store)
	}
}
