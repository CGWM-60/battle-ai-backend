package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAllowedNegotiationActions(t *testing.T) {
	valid := []string{"send_emissary", "offer_resources", "threaten", "propose_treaty", "request_alliance", "request_ceasefire", "negotiate_trade_route", "request_military_support"}
	for _, action := range valid {
		if !isAllowedNegotiationAction(action) {
			t.Fatalf("expected action %q to be allowed", action)
		}
	}
	if isAllowedNegotiationAction("invalid_action") {
		t.Fatalf("unexpected allowed invalid action")
	}
}

func TestAllowedEmissaryMissions(t *testing.T) {
	valid := []string{"commerce", "paix", "intimidation", "alliance", "espionnage", "crise", "guilde"}
	for _, mission := range valid {
		if !isAllowedEmissaryMission(mission) {
			t.Fatalf("expected mission %q to be allowed", mission)
		}
	}
	if isAllowedEmissaryMission("unknown") {
		t.Fatalf("unexpected allowed invalid mission")
	}
}

func TestNewAPIErrorDefaults(t *testing.T) {
	err, ok := newAPIError(0, "", "boom", nil).(*apiError)
	if !ok {
		t.Fatalf("expected *apiError")
	}
	if err.status != http.StatusBadRequest {
		t.Fatalf("expected default status 400, got %d", err.status)
	}
	if err.code != "BAD_REQUEST" {
		t.Fatalf("expected default code BAD_REQUEST, got %s", err.code)
	}
	if err.details == nil {
		t.Fatalf("expected details map initialized")
	}
}

func TestWriteWorldResponseWithAPIError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writeWorldResponse(c, nil, conflictError("COOLDOWN_ACTIVE", "Wait", map[string]any{"cooldownSeconds": 60}))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if success, _ := body["success"].(bool); success {
		t.Fatalf("expected success=false")
	}
	errorObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object")
	}
	if code, _ := errorObj["code"].(string); code != "COOLDOWN_ACTIVE" {
		t.Fatalf("expected code COOLDOWN_ACTIVE, got %q", code)
	}
}
