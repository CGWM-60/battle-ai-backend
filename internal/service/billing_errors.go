package service

import (
	"errors"
	"net/http"
	"strings"
)

const (
	BillingErrorPaymentRequired = "PAYMENT_REQUIRED"
	BillingErrorForbidden       = "FORBIDDEN"
	BillingErrorConflict        = "CONFLICT"
)

// BillingError = erreur HTTP standardisee pour le module billing (402/403/409).
type BillingError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *BillingError) Error() string {
	if e == nil {
		return "billing error"
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func NewBillingError(status int, code string, message string, details map[string]any) *BillingError {
	if details == nil {
		details = map[string]any{}
	}
	return &BillingError{
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
	}
}

func PaymentRequiredError(message string, details map[string]any) *BillingError {
	if message == "" {
		message = "insufficient credits"
	}
	return NewBillingError(http.StatusPaymentRequired, BillingErrorPaymentRequired, message, details)
}

func BillingForbiddenError(message string, details map[string]any) *BillingError {
	if message == "" {
		message = "access denied"
	}
	return NewBillingError(http.StatusForbidden, BillingErrorForbidden, message, details)
}

func BillingConflictError(message string, details map[string]any) *BillingError {
	if message == "" {
		message = "billing conflict"
	}
	return NewBillingError(http.StatusConflict, BillingErrorConflict, message, details)
}

func AsBillingError(err error) (*BillingError, bool) {
	if err == nil {
		return nil, false
	}
	var billingErr *BillingError
	if errors.As(err, &billingErr) {
		return billingErr, true
	}
	return nil, false
}

func MapBillingError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := AsBillingError(err); ok {
		return err
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "insufficient") || strings.Contains(message, "not enough credits") || strings.Contains(message, "insufficient_credits"):
		return PaymentRequiredError(err.Error(), nil)
	case strings.Contains(message, "access denied") || strings.Contains(message, "entitlement") || strings.Contains(message, "tier required"):
		return BillingForbiddenError(err.Error(), nil)
	case strings.Contains(message, "already") || strings.Contains(message, "duplicate") || strings.Contains(message, "conflict"):
		return BillingConflictError(err.Error(), nil)
	default:
		return err
	}
}