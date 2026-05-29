package router

import "net/http"

type apiError struct {
	status  int
	code    string
	message string
	details map[string]any
}

func (e *apiError) Error() string {
	if e == nil {
		return "BAD_REQUEST: unknown error"
	}
	return e.code + ": " + e.message
}

func newAPIError(status int, code string, message string, details map[string]any) error {
	if status <= 0 {
		status = http.StatusBadRequest
	}
	if code == "" {
		code = "BAD_REQUEST"
	}
	if details == nil {
		details = map[string]any{}
	}
	return &apiError{status: status, code: code, message: message, details: details}
}

func badRequestError(code string, message string, details map[string]any) error {
	return newAPIError(http.StatusBadRequest, code, message, details)
}

func conflictError(code string, message string, details map[string]any) error {
	return newAPIError(http.StatusConflict, code, message, details)
}

func forbiddenError(code string, message string, details map[string]any) error {
	return newAPIError(http.StatusForbidden, code, message, details)
}

func notFoundError(code string, message string, details map[string]any) error {
	return newAPIError(http.StatusNotFound, code, message, details)
}
