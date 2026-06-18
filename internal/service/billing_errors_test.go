package service

import (
	"errors"
	"net/http"
	"testing"
)

func TestPaymentRequiredErrorStatus(t *testing.T) {
	err := PaymentRequiredError("insufficient credits", map[string]any{"balance": 0})
	if err.Status != http.StatusPaymentRequired {
		t.Fatalf("status=%d want 402", err.Status)
	}
	if err.Code != BillingErrorPaymentRequired {
		t.Fatalf("code=%s", err.Code)
	}
}

func TestMapBillingErrorInsufficientCredits(t *testing.T) {
	mapped := MapBillingError(errors.New("insufficient credits for action"))
	billingErr, ok := AsBillingError(mapped)
	if !ok {
		t.Fatalf("expected billing error")
	}
	if billingErr.Status != http.StatusPaymentRequired {
		t.Fatalf("status=%d want 402", billingErr.Status)
	}
}

func TestMapBillingErrorConflict(t *testing.T) {
	mapped := MapBillingError(errors.New("purchase already processed conflict"))
	billingErr, ok := AsBillingError(mapped)
	if !ok {
		t.Fatalf("expected billing error")
	}
	if billingErr.Status != http.StatusConflict {
		t.Fatalf("status=%d want 409", billingErr.Status)
	}
}