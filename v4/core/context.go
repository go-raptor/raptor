package core

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/go-raptor/raptor/v4/errs"
)

// Context carries one request through its handler chain. Raptor pools
// Contexts: when the handler returns, the Context is cleared and soon
// serves another request, so never keep it or hand it to a goroutine that
// outlives the handler. Copy out what the goroutine needs instead, such as
// ctx.Request().Context() or a parsed param.
type Context struct {
	core     *Core
	request  *http.Request
	response *Response
	// ownResponse is the Response Raptor created for this Context; response
	// differs from it while a middleware's substituted writer is in use.
	ownResponse *Response
	path        string
	query       url.Values

	realIP    string
	realIPSet bool

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
	response := NewResponse(w)
	return &Context{
		request:     r,
		response:    response,
		ownResponse: response,
		core:        c,
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
	c.realIPSet = false // the new request may carry another peer or headers
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

// RealIP returns the client address per the configured ip_extractor. It
// is computed once per request: the logger, the rate limiter and handlers
// all ask for it, and behind a proxy each extraction parses the
// X-Forwarded-For chain.
func (c *Context) RealIP() string {
	if !c.realIPSet {
		c.realIP = c.core.IPExtractor(c.request)
		c.realIPSet = true
	}
	return c.realIP
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
//
// A body must be declared as JSON (application/json or application/*+json),
// or Bind returns a 415. That check is a CSRF defense: a cross-site form can
// post text/plain that happens to be valid JSON, but not application/json
// without a CORS preflight. Malformed JSON is a 400 and an oversized body a
// 413; the decode error stays reachable through errors.Is and errors.As.
func (c *Context) Bind(v any) error {
	return c.BindWith(v)
}

// BindWith is Bind with encoding/json/v2 options, e.g.
// json.RejectUnknownMembers(true) to fail on members v does not declare.
func (c *Context) BindWith(v any, opts ...json.Options) error {
	// A request without a body has no payload to smuggle, so it skips the
	// check and decodes as empty, keeping optional-body handlers working.
	hasBody := c.request.Body != nil && c.request.Body != http.NoBody
	if hasBody && !isJSONContentType(c.request.Header.Get(HeaderContentType)) {
		return errs.NewErrorUnsupportedMediaType("Expected an application/json body")
	}
	if err := json.UnmarshalRead(c.request.Body, v, opts...); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return err // Error renders it as a 413
		}
		return errs.NewErrorBadRequest("Invalid JSON body").WithCause(err)
	}
	return nil
}

func isJSONContentType(value string) bool {
	// What clients send almost always; skips ParseMediaType's allocations.
	if strings.EqualFold(value, MIMEApplicationJSON) || strings.EqualFold(value, "application/json; charset=utf-8") {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	return mediaType == MIMEApplicationJSON ||
		strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json")
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

// jsonBuffers reuse encoding space across responses; buffers that grew
// past maxPooledJSONBuffer are dropped so one huge response doesn't pin
// its memory in the pool.
var jsonBuffers = sync.Pool{New: func() any { return new(bytes.Buffer) }}

const maxPooledJSONBuffer = 64 << 10

func (c *Context) JSON(code int, i any) error {
	if b, ok := i.([]byte); ok {
		return c.JSONBlob(code, b)
	}
	buf := jsonBuffers.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		if buf.Cap() <= maxPooledJSONBuffer {
			jsonBuffers.Put(buf)
		}
	}()
	if err := json.MarshalWrite(buf, i); err != nil {
		return err
	}
	return c.Blob(code, MIMEApplicationJSON, buf.Bytes())
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

// contentDisposition sets the header per RFC 6266: mime.FormatMediaType
// quotes plain names and switches to filename* (RFC 2231) for non-ASCII or
// control characters, so the header stays ASCII and cannot be split.
func (c *Context) contentDisposition(file, name, dispositionType string) error {
	c.response.Header().Set(HeaderContentDisposition, mime.FormatMediaType(dispositionType, map[string]string{"filename": name}))
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

// staleErrorHeaders describe the successful response a handler was
// preparing. Left on an error they let a CDN cache it, or make a browser
// save it as a download. Cache-Control is replaced, not just removed: a
// 404 without it is heuristically cacheable.
var staleErrorHeaders = []string{
	HeaderExpires, HeaderCDNCacheControl, HeaderSurrogateControl,
	HeaderETag, HeaderLastModified, HeaderContentLength, HeaderContentDisposition,
}

// writeError always writes a response: one left untouched is finished by
// net/http as an empty 200, which a client reads as success. Encoding runs
// before anything is written, so each attempt starts from a clean response.
// The retry drops attrs and replaces invalid UTF-8 in the message (which
// often echoes the request path), so the status survives.
func (c *Context) writeError(e *errs.Error, original error) {
	if e.Code < 400 || e.Code > 599 {
		// net/http panics on codes outside 100–999, and a 2xx or 3xx would
		// tell the client the request worked.
		c.core.Resources.Log.Error("Error response with a non-error status, sending 500", "code", e.Code, "original", original)
		e = errs.NewErrorInternal("Internal Server Error")
	}
	c.resetErrorHeaders()

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

// resetErrorHeaders removes the success-path headers, marks the error
// no-store and forces a JSON content type: writeContentType keeps a type the handler already set, and
// an error message may echo request input, so a preset text/html would
// turn it into markup.
func (c *Context) resetErrorHeaders() {
	h := c.response.Header()
	for _, key := range staleErrorHeaders {
		h.Del(key)
	}
	// A writer substituted downstream, such as a compressing middleware,
	// encodes the error body too, so its Content-Encoding must stay; net/http's
	// http.Error keeps it for the same reason. Without one, the encoding was
	// set for content the error replaces.
	if c.response == c.ownResponse {
		h.Del(HeaderContentEncoding)
	}
	h.Set(HeaderCacheControl, "no-store")
	h.Set(HeaderContentType, MIMEApplicationJSON)
	h.Set(HeaderXContentTypeOptions, "nosniff")
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
	c.realIP, c.realIPSet = "", false
}

// release drops the finished request's references, so a Context idle in
// the pool keeps no body, writer or stored values alive. A Context used
// after its handler returned sees a nil Request rather than another
// request's data.
func (c *Context) release() {
	c.request = nil
	c.response.init(nil)
	c.query = nil
	c.handler = nil
	c.routeStore = nil
	clear(c.store)
	c.realIP, c.realIPSet = "", false
}
