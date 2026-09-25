package core

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
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

	// Below this size net/http buffers the whole response and computes
	// Content-Length itself; above it, responses would go chunked unless
	// the length is set explicitly.
	chunkingThreshold = 2048
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

// Bind decodes the JSON request body into v with encoding/json/v2 defaults:
// member names match case-sensitively, unknown members are ignored, and
// duplicate names, invalid UTF-8 and trailing data are errors.
func (c *Context) Bind(v any) error {
	return json.UnmarshalRead(c.request.Body, v)
}

// BindWith is Bind with encoding/json/v2 options, e.g.
// json.RejectUnknownMembers(true) to fail on members v does not declare.
func (c *Context) BindWith(v any, opts ...json.Options) error {
	return json.UnmarshalRead(c.request.Body, v, opts...)
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
	if len(b) > chunkingThreshold {
		c.response.Header().Set(HeaderContentLength, strconv.Itoa(len(b)))
	}
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

// Error writes err as a JSON error response. Deliberate *errs.Error values
// keep their message and status; anything else is logged server-side and
// redacted to a generic 500 so internal details never reach the client.
func (c *Context) Error(err error) {
	if c.response.Committed {
		return
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			e = errs.NewErrorRequestEntityTooLarge("Request body too large")
		} else {
			c.core.Resources.Log.Error("Unhandled error in handler", "controller", c.controller, "action", c.action, "error", err)
			e = errs.NewErrorInternal("Internal Server Error")
		}
	}
	c.writeError(e, err)
}

// errorFallbackBody is sent as-is when even a message-only error cannot be
// encoded, e.g. a Message holding invalid UTF-8.
var errorFallbackBody = []byte(`{"code":500,"message":"Internal Server Error"}`)

// writeError always writes a response: one left untouched is finished by
// net/http as an empty 200, which a client reads as success. Encoding runs
// before anything is written, so each attempt starts from a clean response.
// The retry drops attrs and replaces invalid UTF-8 in the message (which
// often echoes the request path), so the status survives.
func (c *Context) writeError(e *errs.Error, original error) {
	retry := &errs.Error{Code: e.Code, Message: strings.ToValidUTF8(e.Message, "\uFFFD")}
	for _, candidate := range []*errs.Error{e, retry} {
		err := c.Data(candidate, candidate.Code)
		if err == nil {
			return
		}
		if c.response.Committed {
			// The write itself failed (client gone); retrying cannot help.
			c.core.Resources.Log.Error("Failed to write error response", "error", err, "original", original)
			return
		}
		c.core.Resources.Log.Error("Failed to encode error response", "error", err, "original", original, "attrs_dropped", candidate != e)
	}
	if err := c.Blob(http.StatusInternalServerError, MIMEApplicationJSON, errorFallbackBody); err != nil {
		c.core.Resources.Log.Error("Failed to write error response", "error", err, "original", original)
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
