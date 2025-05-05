package errs

import (
	"fmt"
	"net/http"
)

type Error struct {
	Code    int            `json:"code"`
	Message string         `json:"message,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("(%d) %s - %v", e.Code, e.Message, e.Attrs)
}

func NewError(code int, message string, attr ...any) *Error {
	var attrs map[string]any
	if len(attr) > 0 {
		attrs = make(map[string]any)

		for i := 0; i < len(attr); i += 2 {
			if i+1 < len(attr) {
				if key, ok := attr[i].(string); ok {
					attrs[key] = attr[i+1]
				}
			}
		}
	}

	if message == "" {
		message = http.StatusText(code)
	}

	return &Error{
		Code:    code,
		Message: message,
		Attrs:   attrs,
	}
}

func (e *Error) AttrsToSlice() []any {
	if e.Attrs == nil {
		return nil
	}

	args := make([]any, 0, len(e.Attrs)*2)
	for k, v := range e.Attrs {
		args = append(args, k, v)
	}
	return args
}

// 4xx
var (
	ErrBadRequest                   = NewErrorBadRequest("Bad Request")
	ErrUnauthorized                 = NewErrorUnauthorized("Unauthorized")
	ErrPaymentRequired              = NewErrorPaymentRequired("Payment Required")
	ErrForbidden                    = NewErrorForbidden("Forbidden")
	ErrNotFound                     = NewErrorNotFound("Not Found")
	ErrMethodNotAllowed             = NewErrorMethodNotAllowed("Method Not Allowed")
	ErrNotAcceptable                = NewErrorNotAcceptable("Not Acceptable")
	ErrProxyAuthRequired            = NewErrorProxyAuthRequired("Proxy Authentication Required")
	ErrRequestTimeout               = NewErrorRequestTimeout("Request Timeout")
	ErrConflict                     = NewErrorConflict("Conflict")
	ErrGone                         = NewErrorGone("Gone")
	ErrLengthRequired               = NewErrorLengthRequired("Length Required")
	ErrPreconditionFailed           = NewErrorPreconditionFailed("Precondition Failed")
	ErrRequestEntityTooLarge        = NewErrorRequestEntityTooLarge("Request Entity Too Large")
	ErrRequestURITooLong            = NewErrorRequestURITooLong("Request URI Too Long")
	ErrUnsupportedMediaType         = NewErrorUnsupportedMediaType("Unsupported Media Type")
	ErrRequestedRangeNotSatisfiable = NewErrorRequestedRangeNotSatisfiable("Requested Range Not Satisfiable")
	ErrExpectationFailed            = NewErrorExpectationFailed("Expectation Failed")
	ErrTeapot                       = NewErrorTeapot("I'm a teapot")
	ErrMisdirectedRequest           = NewErrorMisdirectedRequest("Misdirected Request")
	ErrUnprocessableEntity          = NewErrorUnprocessableEntity("Unprocessable Entity")
	ErrLocked                       = NewErrorLocked("Locked")
	ErrFailedDependency             = NewErrorFailedDependency("Failed Dependency")
	ErrTooEarly                     = NewErrorTooEarly("Too Early")
	ErrUpgradeRequired              = NewErrorUpgradeRequired("Upgrade Required")
	ErrPreconditionRequired         = NewErrorPreconditionRequired("Precondition Required")
	ErrTooManyRequests              = NewErrorTooManyRequests("Too Many Requests")
	ErrRequestHeaderFieldsTooLarge  = NewErrorRequestHeaderFieldsTooLarge("Request Header Fields Too Large")
	ErrUnavailableForLegalReasons   = NewErrorUnavailableForLegalReasons("Unavailable For Legal Reasons")

	ErrValidatorNotRegistered = NewError(500, "Validator not registered")
	ErrInvalidRedirectCode    = NewError(500, "Invalid redirect status code")
	ErrCookieNotFound         = NewError(500, "Cookie not found")
	ErrInvalidCertOrKeyType   = NewError(500, "Invalid cert or key type, must be string or []byte")
	ErrInvalidListenerNetwork = NewError(500, "Invalid listener network")
)

func NewErrorBadRequest(message string, attr ...any) *Error {
	return NewError(http.StatusBadRequest, message, attr...)
}

func NewErrorUnauthorized(message string, attr ...any) *Error {
	return NewError(http.StatusUnauthorized, message, attr...)
}

func NewErrorPaymentRequired(message string, attr ...any) *Error {
	return NewError(http.StatusPaymentRequired, message, attr...)
}

func NewErrorForbidden(message string, attr ...any) *Error {
	return NewError(http.StatusForbidden, message, attr...)
}

func NewErrorNotFound(message string, attr ...any) *Error {
	return NewError(http.StatusNotFound, message, attr...)
}

func NewErrorMethodNotAllowed(message string, attr ...any) *Error {
	return NewError(http.StatusMethodNotAllowed, message, attr...)
}

func NewErrorNotAcceptable(message string, attr ...any) *Error {
	return NewError(http.StatusNotAcceptable, message, attr...)
}

func NewErrorProxyAuthRequired(message string, attr ...any) *Error {
	return NewError(http.StatusProxyAuthRequired, message, attr...)
}

func NewErrorRequestTimeout(message string, attr ...any) *Error {
	return NewError(http.StatusRequestTimeout, message, attr...)
}

func NewErrorConflict(message string, attr ...any) *Error {
	return NewError(http.StatusConflict, message, attr...)
}

func NewErrorGone(message string, attr ...any) *Error {
	return NewError(http.StatusGone, message, attr...)
}

func NewErrorLengthRequired(message string, attr ...any) *Error {
	return NewError(http.StatusLengthRequired, message, attr...)
}

func NewErrorPreconditionFailed(message string, attr ...any) *Error {
	return NewError(http.StatusPreconditionFailed, message, attr...)
}

func NewErrorRequestEntityTooLarge(message string, attr ...any) *Error {
	return NewError(http.StatusRequestEntityTooLarge, message, attr...)
}

func NewErrorRequestURITooLong(message string, attr ...any) *Error {
	return NewError(http.StatusRequestURITooLong, message, attr...)
}

func NewErrorUnsupportedMediaType(message string, attr ...any) *Error {
	return NewError(http.StatusUnsupportedMediaType, message, attr...)
}

func NewErrorRequestedRangeNotSatisfiable(message string, attr ...any) *Error {
	return NewError(http.StatusRequestedRangeNotSatisfiable, message, attr...)
}

func NewErrorExpectationFailed(message string, attr ...any) *Error {
	return NewError(http.StatusExpectationFailed, message, attr...)
}

func NewErrorTeapot(message string, attr ...any) *Error {
	return NewError(http.StatusTeapot, message, attr...)
}

func NewErrorMisdirectedRequest(message string, attr ...any) *Error {
	return NewError(http.StatusMisdirectedRequest, message, attr...)
}

func NewErrorUnprocessableEntity(message string, attr ...any) *Error {
	return NewError(http.StatusUnprocessableEntity, message, attr...)
}

func NewErrorLocked(message string, attr ...any) *Error {
	return NewError(http.StatusLocked, message, attr...)
}

func NewErrorFailedDependency(message string, attr ...any) *Error {
	return NewError(http.StatusFailedDependency, message, attr...)
}

func NewErrorTooEarly(message string, attr ...any) *Error {
	return NewError(http.StatusTooEarly, message, attr...)
}

func NewErrorUpgradeRequired(message string, attr ...any) *Error {
	return NewError(http.StatusUpgradeRequired, message, attr...)
}

func NewErrorPreconditionRequired(message string, attr ...any) *Error {
	return NewError(http.StatusPreconditionRequired, message, attr...)
}

func NewErrorTooManyRequests(message string, attr ...any) *Error {
	return NewError(http.StatusTooManyRequests, message, attr...)
}

func NewErrorRequestHeaderFieldsTooLarge(message string, attr ...any) *Error {
	return NewError(http.StatusRequestHeaderFieldsTooLarge, message, attr...)
}

func NewErrorUnavailableForLegalReasons(message string, attr ...any) *Error {
	return NewError(http.StatusUnavailableForLegalReasons, message, attr...)
}

// 5xx
func NewErrorInternal(message string, attr ...any) *Error {
	return NewError(http.StatusInternalServerError, message, attr...)
}

func NewErrorNotImplemented(message string, attr ...any) *Error {
	return NewError(http.StatusNotImplemented, message, attr...)
}

func NewErrorBadGateway(message string, attr ...any) *Error {
	return NewError(http.StatusBadGateway, message, attr...)
}

func NewErrorServiceUnavailable(message string, attr ...any) *Error {
	return NewError(http.StatusServiceUnavailable, message, attr...)
}

func NewErrorGatewayTimeout(message string, attr ...any) *Error {
	return NewError(http.StatusGatewayTimeout, message, attr...)
}

func NewErrorHTTPVersionNotSupported(message string, attr ...any) *Error {
	return NewError(http.StatusHTTPVersionNotSupported, message, attr...)
}

func NewErrorVariantAlsoNegotiates(message string, attr ...any) *Error {
	return NewError(http.StatusVariantAlsoNegotiates, message, attr...)
}

func NewErrorInsufficientStorage(message string, attr ...any) *Error {
	return NewError(http.StatusInsufficientStorage, message, attr...)
}

func NewErrorLoopDetected(message string, attr ...any) *Error {
	return NewError(http.StatusLoopDetected, message, attr...)
}

func NewErrorNotExtended(message string, attr ...any) *Error {
	return NewError(http.StatusNotExtended, message, attr...)
}

func NewErrorNetworkAuthenticationRequired(message string, attr ...any) *Error {
	return NewError(http.StatusNetworkAuthenticationRequired, message, attr...)
}
