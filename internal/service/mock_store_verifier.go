package service

import (
	"context"
	"fmt"
	"strings"
)

// StoreVerificationInput = donnees minimales pour verifier un achat mock/dev.
type StoreVerificationInput struct {
	Platform       string
	ReceiptID      string
	StoreProductID string
	UserID         uint
}

// StoreVerificationResult = resultat de verification store.
type StoreVerificationResult struct {
	Valid          bool
	Platform       string
	ReceiptID      string
	StoreProductID string
	Message        string
}

// StoreVerifier verifie un recu d'achat store.
type StoreVerifier interface {
	VerifyPurchase(ctx context.Context, input StoreVerificationInput) (StoreVerificationResult, error)
}

// MockStoreVerifier accepte les recus mock pour le dev/local.
type MockStoreVerifier struct{}

func NewMockStoreVerifier() *MockStoreVerifier {
	return &MockStoreVerifier{}
}

func (v *MockStoreVerifier) VerifyPurchase(_ context.Context, input StoreVerificationInput) (StoreVerificationResult, error) {
	receiptID := strings.TrimSpace(input.ReceiptID)
	storeProductID := strings.TrimSpace(input.StoreProductID)
	platform := strings.TrimSpace(input.Platform)
	if platform == "" {
		platform = "mock"
	}
	if receiptID == "" {
		return StoreVerificationResult{Valid: false, Message: "receipt id is required"}, fmt.Errorf("receipt id is required")
	}
	if storeProductID == "" {
		return StoreVerificationResult{Valid: false, Message: "store product id is required"}, fmt.Errorf("store product id is required")
	}
	if input.UserID == 0 {
		return StoreVerificationResult{Valid: false, Message: "user id is required"}, fmt.Errorf("user id is required")
	}
	return StoreVerificationResult{
		Valid:          true,
		Platform:       platform,
		ReceiptID:      receiptID,
		StoreProductID: storeProductID,
		Message:        "mock receipt accepted",
	}, nil
}