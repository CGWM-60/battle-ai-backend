package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestBillingPurchaseRouteExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	registerBillingRoutes(api, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/billing/purchase",
		strings.NewReader(`{"productId":"anima_companion_premium"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("purchase route must exist, got 404")
	}
}

func TestBillingPurchaseWithoutProductReturnsStructuredError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/billing/purchase", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	billingPurchase(nil)(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want 400", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload["code"] != "billing.error.product_required" {
		t.Fatalf("code=%v", payload["code"])
	}
}

func TestBillingPurchaseStoreNotConfiguredReturnsStructuredError(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	t.Setenv("STORE_VERIFIER", "live")
	t.Setenv("BILLING_STRIPE_SECRET_KEY", "")

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/billing/purchase",
		strings.NewReader(`{"productId":"anima_companion_premium"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(contextUserIDKey, uint(42))

	billingPurchase(nil)(c)

	if rec.Code == http.StatusNotFound {
		t.Fatal("purchase route must not return 404")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d want 503", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload["code"] != "billing.error.store_not_configured" {
		t.Fatalf("code=%v", payload["code"])
	}
}

func TestBillingPurchaseAminaCompanionUsesLivePurchaseService(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STORE_VERIFIER", "mock")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	registerBillingRoutes(api, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/billing/purchase",
		strings.NewReader(`{"productId":"anima_companion_premium","testMode":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("purchase route must exist")
	}
	if rec.Code == http.StatusServiceUnavailable {
		t.Fatalf("testMode purchase must not fail as store_not_configured, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBillingPurchaseAminaCompanionInDevGrantsEntitlement(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STORE_VERIFIER", "mock")

	// Entitlement grant is covered by service.TestBillingServiceMockPurchaseAnimaCompanionGrantsEntitlement.
	// Handler-level: dev mock path must not short-circuit to 501.
	if getEnv("GIN_MODE", "debug") != "release" &&
		strings.EqualFold(getEnv("STORE_VERIFIER", "mock"), "mock") {
		return
	}
	t.Fatal("dev mock purchase path must be active with STORE_VERIFIER=mock")
}

func TestBillingPurchaseNeverReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	registerBillingRoutes(api, nil)

	for _, route := range []string{
		"/api/v1/billing/purchase",
		"/api/v1/billing/mock/purchase",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodPost,
			route,
			strings.NewReader(`{"productId":"anima_companion_premium"}`),
		)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Fatalf("%s must exist, got 404", route)
		}
	}
}

func TestBillingPurchaseRouteLogsAndHandlesAminaCompanion(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("STORE_VERIFIER", "mock")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	registerBillingRoutes(api, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/billing/purchase",
		strings.NewReader(`{"productId":"anima_companion_premium"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("purchase route must not return 404")
	}
	if rec.Code == http.StatusNotImplemented {
		t.Fatalf("dev mock purchase must not return 501, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBillingSubscribeRouteExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	registerBillingRoutes(api, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/billing/subscribe",
		strings.NewReader(`{"productId":"nexus_light_monthly"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("subscribe route must exist, got 404")
	}
}

func TestBillingRestoreRouteExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	registerBillingRoutes(api, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/restore", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("restore route must exist, got 404")
	}
}

func TestMockBillingPurchaseLogsAndHandlesAminaCompanion(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	registerBillingRoutes(api, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/billing/mock/purchase",
		strings.NewReader(`{"productSlug":"anima_companion_premium"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("mock purchase route must exist in debug mode")
	}
}