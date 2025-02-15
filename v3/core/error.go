package core

import (
	"fmt"
	"net/http"
)

func (e *Error) Error() string {
	return fmt.Sprintf("(%d) %s", e.Code, e.Message)
}

func NewError(code int, descriptions ...string) *Error {
	var description string
	if len(descriptions) > 0 {
		description = descriptions[0]
	}

	return &Error{
		Code:        code,
		Message:     http.StatusText(code),
		Description: description,
	}
}

// 4xx
func NewErrorBadRequest(descriptions ...string) *Error {
	return NewError(http.StatusBadRequest, descriptions...)
}

func NewErrorUnauthorized(descriptions ...string) *Error {
	return NewError(http.StatusUnauthorized, descriptions...)
}

func NewErrorPaymentRequired(descriptions ...string) *Error {
	return NewError(http.StatusPaymentRequired, descriptions...)
}

func NewErrorForbidden(descriptions ...string) *Error {
	return NewError(http.StatusForbidden, descriptions...)
}

func NewErrorNotFound(descriptions ...string) *Error {
	return NewError(http.StatusNotFound, descriptions...)
}

func NewErrorMethodNotAllowed(descriptions ...string) *Error {
	return NewError(http.StatusMethodNotAllowed, descriptions...)
}

func NewErrorNotAcceptable(descriptions ...string) *Error {
	return NewError(http.StatusNotAcceptable, descriptions...)
}

func NewErrorProxyAuthRequired(descriptions ...string) *Error {
	return NewError(http.StatusProxyAuthRequired, descriptions...)
}

func NewErrorRequestTimeout(descriptions ...string) *Error {
	return NewError(http.StatusRequestTimeout, descriptions...)
}

func NewErrorConflict(descriptions ...string) *Error {
	return NewError(http.StatusConflict, descriptions...)
}

func NewErrorGone(descriptions ...string) *Error {
	return NewError(http.StatusGone, descriptions...)
}

func NewErrorLengthRequired(descriptions ...string) *Error {
	return NewError(http.StatusLengthRequired, descriptions...)
}

func NewErrorPreconditionFailed(descriptions ...string) *Error {
	return NewError(http.StatusPreconditionFailed, descriptions...)
}

func NewErrorRequestEntityTooLarge(descriptions ...string) *Error {
	return NewError(http.StatusRequestEntityTooLarge, descriptions...)
}

func NewErrorRequestURITooLong(descriptions ...string) *Error {
	return NewError(http.StatusRequestURITooLong, descriptions...)
}

func NewErrorUnsupportedMediaType(descriptions ...string) *Error {
	return NewError(http.StatusUnsupportedMediaType, descriptions...)
}

func NewErrorRequestedRangeNotSatisfiable(descriptions ...string) *Error {
	return NewError(http.StatusRequestedRangeNotSatisfiable, descriptions...)
}

func NewErrorExpectationFailed(descriptions ...string) *Error {
	return NewError(http.StatusExpectationFailed, descriptions...)
}

func NewErrorTeapot(descriptions ...string) *Error {
	return NewError(http.StatusTeapot, descriptions...)
}

func NewErrorMisdirectedRequest(descriptions ...string) *Error {
	return NewError(http.StatusMisdirectedRequest, descriptions...)
}

func NewErrorUnprocessableEntity(descriptions ...string) *Error {
	return NewError(http.StatusUnprocessableEntity, descriptions...)
}

func NewErrorLocked(descriptions ...string) *Error {
	return NewError(http.StatusLocked, descriptions...)
}

func NewErrorFailedDependency(descriptions ...string) *Error {
	return NewError(http.StatusFailedDependency, descriptions...)
}

func NewErrorTooEarly(descriptions ...string) *Error {
	return NewError(http.StatusTooEarly, descriptions...)
}

func NewErrorUpgradeRequired(descriptions ...string) *Error {
	return NewError(http.StatusUpgradeRequired, descriptions...)
}

func NewErrorPreconditionRequired(descriptions ...string) *Error {
	return NewError(http.StatusPreconditionRequired, descriptions...)
}

func NewErrorTooManyRequests(descriptions ...string) *Error {
	return NewError(http.StatusTooManyRequests, descriptions...)
}

func NewErrorRequestHeaderFieldsTooLarge(descriptions ...string) *Error {
	return NewError(http.StatusRequestHeaderFieldsTooLarge, descriptions...)
}

func NewErrorUnavailableForLegalReasons(descriptions ...string) *Error {
	return NewError(http.StatusUnavailableForLegalReasons, descriptions...)
}

// 5xx
func NewErrorInternal(descriptions ...string) *Error {
	return NewError(http.StatusInternalServerError, descriptions...)
}

func NewErrorNotImplemented(descriptions ...string) *Error {
	return NewError(http.StatusNotImplemented, descriptions...)
}

func NewErrorBadGateway(descriptions ...string) *Error {
	return NewError(http.StatusBadGateway, descriptions...)
}

func NewErrorServiceUnavailable(descriptions ...string) *Error {
	return NewError(http.StatusServiceUnavailable, descriptions...)
}

func NewErrorGatewayTimeout(descriptions ...string) *Error {
	return NewError(http.StatusGatewayTimeout, descriptions...)
}

func NewErrorHTTPVersionNotSupported(descriptions ...string) *Error {
	return NewError(http.StatusHTTPVersionNotSupported, descriptions...)
}

func NewErrorVariantAlsoNegotiates(descriptions ...string) *Error {
	return NewError(http.StatusVariantAlsoNegotiates, descriptions...)
}

func NewErrorInsufficientStorage(descriptions ...string) *Error {
	return NewError(http.StatusInsufficientStorage, descriptions...)
}

func NewErrorLoopDetected(descriptions ...string) *Error {
	return NewError(http.StatusLoopDetected, descriptions...)
}

func NewErrorNotExtended(descriptions ...string) *Error {
	return NewError(http.StatusNotExtended, descriptions...)
}

func NewErrorNetworkAuthenticationRequired(descriptions ...string) *Error {
	return NewError(http.StatusNetworkAuthenticationRequired, descriptions...)
}
