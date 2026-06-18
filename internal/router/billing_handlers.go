package router

import (
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"
	"cgwm/battle/internal/service"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type billingEstimateRequest struct {
	ActionType   string         `json:"actionType"`
	Feature      string         `json:"feature"`
	PromptChars  int            `json:"promptChars"`
	ModelName    string         `json:"modelName"`
	ProviderName string         `json:"providerName"`
	Metadata     map[string]any `json:"metadata"`
}

type billingPurchaseRequest struct {
	ProductID   string `json:"productId"`
	ProductSlug string `json:"productSlug"`
	ReceiptID   string `json:"receiptId"`
	Platform    string `json:"platform"`
}

type billingSubscribeRequest struct {
	ProductID   string `json:"productId"`
	ProductSlug string `json:"productSlug"`
	ReceiptID   string `json:"receiptId"`
	Platform    string `json:"platform"`
}

type billingRestoreRequest struct {
	ReceiptID string `json:"receiptId"`
	Platform  string `json:"platform"`
}

type billingModeRequest struct {
	BillingMode string `json:"billingMode"`
}

type billingCancelSubscriptionRequest struct {
	AtPeriodEnd bool `json:"atPeriodEnd"`
}

func registerBillingRoutes(private *gin.RouterGroup, database *gorm.DB) {
	billing := private.Group("/billing")
	billing.GET("/products", listBillingProducts(database))
	billing.GET("/wallet", getBillingWallet(database))
	billing.GET("/ledger", getBillingLedger(database))
	billing.GET("/entitlements", getBillingEntitlements(database))
	billing.GET("/subscription", getBillingSubscription(database))
	billing.POST("/subscription/cancel", cancelBillingSubscription(database))
	billing.GET("/access/:tier", getBillingTierAccess(database))
	billing.POST("/mock/purchase", mockBillingPurchase(database))
	billing.POST("/mock/subscribe", mockBillingSubscribe(database))
	billing.POST("/mock/restore", mockBillingRestore(database))
	billing.PATCH("/mode", updateBillingMode(database))

	private.POST("/ai/estimate", estimateAIUsage(database))
}

func newBillingService(database *gorm.DB) *service.BillingService {
	wallets := service.NewWalletService(repository.NewWalletRepository(database), service.NewAICreditEstimator())
	entitlements := service.NewEntitlementService(repository.NewEntitlementRepository(database))
	return service.NewBillingService(
		wallets,
		service.NewStoreProductService(repository.NewStoreProductRepository(database)),
		entitlements,
		repository.NewStoreTransactionRepository(database),
		repository.NewSubscriptionRepository(database),
		service.NewStoreVerifierFromEnv(),
	)
}

func newAIOrchestrator(database *gorm.DB) *service.AIOrchestrator {
	wallets := service.NewWalletService(repository.NewWalletRepository(database), service.NewAICreditEstimator())
	entitlements := service.NewEntitlementService(repository.NewEntitlementRepository(database))
	return service.NewAIOrchestrator(wallets, service.NewAICreditEstimator(), entitlements)
}

func listBillingProducts(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		billing := newBillingService(database)
		productType := strings.TrimSpace(c.Query("type"))
		products, err := billing.ListProducts(c.Request.Context(), productType)
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"products": serializeStoreProducts(products)})
	}
}

func getBillingWallet(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		billing := newBillingService(database)
		wallet, err := billing.GetWallet(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"wallet": wallet})
	}
}

func getBillingLedger(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		billing := newBillingService(database)
		items, total, err := billing.ListLedger(c.Request.Context(), currentUserID(c), limit, offset)
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"items":   serializeLedgerEntries(items),
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": int64(offset+len(items)) < total,
		})
	}
}

func getBillingEntitlements(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		billing := newBillingService(database)
		entitlements, err := billing.ListEntitlements(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"entitlements": serializeEntitlements(entitlements)})
	}
}

func getBillingTierAccess(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tier := strings.TrimSpace(c.Param("tier"))
		billing := newBillingService(database)
		allowed, currentTier, err := billing.CheckTierAccess(c.Request.Context(), currentUserID(c), tier)
		if err != nil {
			writeBillingError(c, err)
			return
		}
		if !allowed {
			writeBillingError(c, service.BillingForbiddenError("tier access denied", map[string]any{
				"requiredTier": tier,
				"currentTier":  currentTier,
			}))
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"allowed":      true,
			"requiredTier": tier,
			"currentTier":  currentTier,
		})
	}
}

func getBillingSubscription(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		subscription, err := newBillingService(database).GetSubscription(
			c.Request.Context(),
			currentUserID(c),
		)
		if err != nil {
			writeBillingError(c, err)
			return
		}
		if subscription == nil || !subscription.Active {
			c.JSON(http.StatusOK, gin.H{
				"subscription": gin.H{
					"active": false,
				},
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"subscription": subscription})
	}
}

func cancelBillingSubscription(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req billingCancelSubscriptionRequest
		if err := bindPayload(c, &req); err != nil {
			req.AtPeriodEnd = true
		}
		result, err := newBillingService(database).CancelSubscription(
			c.Request.Context(),
			currentUserID(c),
			req.AtPeriodEnd,
		)
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func mockBillingPurchase(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req billingPurchaseRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid purchase payload"})
			return
		}
		productID := strings.TrimSpace(req.ProductID)
		if productID == "" {
			productID = strings.TrimSpace(req.ProductSlug)
		}
		result, err := newBillingService(database).MockPurchase(c.Request.Context(), service.MockPurchaseInput{
			UserID:    currentUserID(c),
			ProductID: productID,
			ReceiptID: req.ReceiptID,
			Platform:  req.Platform,
		})
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func mockBillingSubscribe(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req billingSubscribeRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscribe payload"})
			return
		}
		productID := strings.TrimSpace(req.ProductID)
		if productID == "" {
			productID = strings.TrimSpace(req.ProductSlug)
		}
		result, err := newBillingService(database).MockSubscribe(c.Request.Context(), service.MockSubscribeInput{
			UserID:    currentUserID(c),
			ProductID: productID,
			ReceiptID: req.ReceiptID,
			Platform:  req.Platform,
		})
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func mockBillingRestore(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req billingRestoreRequest
		_ = bindPayload(c, &req)
		result, err := newBillingService(database).MockRestore(c.Request.Context(), service.MockRestoreInput{
			UserID:    currentUserID(c),
			ReceiptID: req.ReceiptID,
			Platform:  req.Platform,
		})
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func updateBillingMode(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req billingModeRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid billing mode payload"})
			return
		}
		wallet, err := newBillingService(database).UpdateBillingMode(c.Request.Context(), currentUserID(c), req.BillingMode)
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"wallet": wallet})
	}
}

func estimateAIUsage(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req billingEstimateRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid estimate payload"})
			return
		}
		payload, err := newBillingService(database).EstimateUsage(
			c.Request.Context(),
			currentUserID(c),
			req.ActionType,
			req.Feature,
			req.PromptChars,
			req.ProviderName,
			req.ModelName,
			req.Metadata,
		)
		if err != nil {
			writeBillingError(c, err)
			return
		}
		c.JSON(http.StatusOK, payload)
	}
}

func writeBillingError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	mapped := service.MapBillingError(err)
	if billingErr, ok := service.AsBillingError(mapped); ok {
		c.JSON(billingErr.Status, gin.H{
			"error":   billingErr.Code,
			"message": billingErr.Message,
			"details": billingErr.Details,
		})
		return
	}
	status := http.StatusBadRequest
	if errors.Is(err, gorm.ErrRecordNotFound) {
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func serializeStoreProducts(products []models.StoreProduct) []gin.H {
	items := make([]gin.H, 0, len(products))
	for _, product := range products {
		items = append(items, gin.H{
			"id":          product.Slug,
			"type":        product.ProductType,
			"name":        product.Name,
			"description": product.Description,
			"credits":     product.NexusCoinsGrant,
			"priceCents":  product.PriceCents,
			"currency":    product.Currency,
			"badge":       product.Badge,
			"popular":     product.Popular,
			"interval":    product.Interval,
			"tier":        product.Tier,
		})
	}
	return items
}

func serializeLedgerEntries(entries []models.AIWalletLedger) []gin.H {
	items := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		items = append(items, gin.H{
			"id":           strconv.FormatUint(uint64(entry.Id), 10),
			"type":         entry.EntryType,
			"amount":       entry.Amount,
			"balanceAfter": entry.BalanceAfter,
			"description":  entry.Description,
			"feature":      entry.Feature,
			"createdAt":    entry.CreatedAt,
		})
	}
	return items
}

func serializeEntitlements(entitlements []models.UserEntitlement) []gin.H {
	items := make([]gin.H, 0, len(entitlements))
	for _, entitlement := range entitlements {
		items = append(items, gin.H{
			"key":       entitlement.Key,
			"value":     entitlement.Value,
			"source":    entitlement.Source,
			"sourceRef": entitlement.SourceRef,
			"active":    entitlement.Active,
			"expiresAt": entitlement.ExpiresAt,
		})
	}
	return items
}
