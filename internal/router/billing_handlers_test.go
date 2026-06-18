package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
)

func TestWriteBillingErrorPaymentRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writeBillingError(c, service.PaymentRequiredError("insufficient credits", nil))

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("code=%d want 402", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload["error"] != "PAYMENT_REQUIRED" {
		t.Fatalf("error=%v", payload["error"])
	}
}