package core

import (
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type Context struct {
	controller string
	action     string
	request    *http.Request
	response   *Response
	query      url.Values

	store map[string]interface{}
	lock  sync.RWMutex

	binder Binder
	// following fields are set by Router
	handler HandlerFunc

	// path is route path that Router matched. It is empty string where there is no route match.
	// Route registered with RouteNotFound is considered as a match and path therefore is not empty.
	path string

	// Usually echo.Echo is sizing pvalues but there could be user created middlewares that decide to
	// overwrite parameter by calling SetParamNames + SetParamValues.
	// When echo.Echo allocated that slice it length/capacity is tied to echo.Echo.maxParam value.
	//
	// It is important that pvalues size is always equal or bigger to pnames length.
	pvalues []string

	// pnames length is tied to param count for the matched route
	pnames []string
}

const (
	// ContextKeyHeaderAllow is set by Router for getting value for `Allow` header in later stages of handler call chain.
	// Allow header is mandatory for status 405 (method not found) and useful for OPTIONS method requests.
	// It is added to context only when Router does not find matching method handler for request.
	ContextKeyHeaderAllow = "echo_header_allow"
)

const (
	defaultMemory = 32 << 20 // 32 MB
	indexPage     = "index.html"
	defaultIndent = "  "
)

func NewContext(r *http.Request, w http.ResponseWriter, binder Binder) *Context {
	return &Context{
		request:  r,
		response: NewResponse(w),
		store:    make(map[string]interface{}),
		pvalues:  make([]string, 0),
		binder:   binder,
		handler:  nil,
	}
}

func (c *Context) writeContentType(value string) {
	header := c.Response().Header()
	if header.Get(HeaderContentType) == "" {
		header.Set(HeaderContentType, value)
	}
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

func (c *Context) IsTLS() bool {
	return c.request.TLS != nil
}

func (c *Context) IsWebSocket() bool {
	upgrade := c.request.Header.Get(HeaderUpgrade)
	return strings.EqualFold(upgrade, "websocket")
}

func (c *Context) Scheme() string {
	// Can't use `r.Request.URL.Scheme`
	// See: https://groups.google.com/forum/#!topic/golang-nuts/pMUkBlQBDF0
	if c.IsTLS() {
		return "https"
	}
	if scheme := c.request.Header.Get(HeaderXForwardedProto); scheme != "" {
		return scheme
	}
	if scheme := c.request.Header.Get(HeaderXForwardedProtocol); scheme != "" {
		return scheme
	}
	if ssl := c.request.Header.Get(HeaderXForwardedSsl); ssl == "on" {
		return "https"
	}
	if scheme := c.request.Header.Get(HeaderXUrlScheme); scheme != "" {
		return scheme
	}
	return "http"
}

func (c *Context) RealIP() string {
	// TODO:
	/*if c.echo != nil && c.echo.IPExtractor != nil {
		return c.echo.IPExtractor(c.request)
	}*/
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

func (c *Context) SetPath(p string) {
	c.path = p
}

func (c *Context) Param(name string) string {
	return c.request.PathValue("id")
}

func (c *Context) ParamNames() []string {
	return c.pnames
}

func (c *Context) SetParamNames(names ...string) {
	c.pnames = names

	l := len(names)
	if len(c.pvalues) < l {
		// Keeping the old pvalues just for backward compatibility, but it sounds that doesn't make sense to keep them,
		// probably those values will be overridden in a Context#SetParamValues
		newPvalues := make([]string, l)
		copy(newPvalues, c.pvalues)
		c.pvalues = newPvalues
	}
}

func (c *Context) ParamValues() []string {
	return c.pvalues[:len(c.pnames)]
}

func (c *Context) SetParamValues(values ...string) {
	// NOTE: Don't just set c.pvalues = values, because it has to have length c.echo.maxParam (or bigger) at all times
	// It will brake the Router#Find code
	limit := len(values)
	if limit > len(c.pvalues) {
		c.pvalues = make([]string, limit)
	}
	for i := 0; i < limit; i++ {
		c.pvalues[i] = values[i]
	}
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

	if c.store == nil {
		c.store = make(map[string]interface{})
	}
	c.store[key] = val
}

func (c *Context) Bind(i interface{}) error {
	return c.binder.Bind(i, c)
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

func (c *Context) NoContent(code int) error {
	c.response.WriteHeader(code)
	return nil
}

func (c *Context) Redirect(code int, url string) error {
	if code < 300 || code > 308 {
		return ErrInvalidRedirectCode
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
	NewError(http.StatusInternalServerError, err.Error(), err)
}

func (c *Context) Handler() HandlerFunc {
	return c.handler
}

func (c *Context) SetHandler(h HandlerFunc) {
	c.handler = h
}

func (c *Context) Reset(r *http.Request, w http.ResponseWriter, controller, action string) {
	c.controller = controller
	c.action = action
	c.request = r
	c.response.reset(w)
	c.query = nil
	c.handler = nil
	c.store = nil
	c.path = ""
	c.pnames = nil
	// NOTE: Don't reset because it has to have length c.echo.maxParam (or bigger) at all times
	for i := 0; i < len(c.pvalues); i++ {
		c.pvalues[i] = ""
	}
}
