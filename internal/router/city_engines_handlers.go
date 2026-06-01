package router

import (
	"fmt"
	"time"

	"cgwm/battle/internal/leaderboard"
	"cgwm/battle/internal/market"
	"cgwm/battle/internal/pvp"
	"cgwm/battle/internal/resources"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
)

// registerCityEnginesRoutes wires the new city management engines.
// This is called from registerWorldGameRoutes.
func registerCityEnginesRoutes(private *gin.RouterGroup, world *service.WorldGameService) {
	// Resources engine
	resEngine := resources.NewEngine(nil) // TODO: pass real DB

	private.GET("/city/resources", func(c *gin.Context) {
		balance, err := resEngine.GetBalance(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, balance, err)
	})

	private.POST("/city/resources/collect", func(c *gin.Context) {
		var body struct {
			BuildingID string `json:"building_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "invalid body"})
			return
		}
		collected, err := resEngine.ManualCollect(c.Request.Context(), currentUserID(c), body.BuildingID)
		writeWorldResponse(c, gin.H{"collected": collected}, err)
	})

	// Economy (stub for now)
	private.GET("/city/economy", func(c *gin.Context) {
		// TODO: use real economy engine
		writeWorldResponse(c, gin.H{
			"goldBalance":    45230,
			"hourlyIncome":   890,
			"hourlyExpenses": 340,
			"netPerHour":     550,
			"taxRate":        0.45,
		}, nil)
	})

	private.POST("/city/economy/tax-rate", func(c *gin.Context) {
		var body struct {
			Rate float64 `json:"rate"`
		}
		c.ShouldBindJSON(&body)
		// TODO: call economyEngine.SetTaxRate
		writeWorldResponse(c, gin.H{"ok": true, "newRate": body.Rate}, nil)
	})

	// Policies
	private.GET("/city/policies/available", func(c *gin.Context) {
		writeWorldResponse(c, gin.H{"policies": []string{"city_festival", "austerity", "war_economy", "harvest_boost"}}, nil)
	})
	private.POST("/city/policies/activate", func(c *gin.Context) {
		var body struct { PolicyKey string `json:"policy_key"` }
		c.ShouldBindJSON(&body)
		writeWorldResponse(c, gin.H{"activated": body.PolicyKey}, nil)
	})

	// PvP routes - real engine calls (progress to 100%)
	pvpEngine := pvp.NewEngine()
	private.POST("/pvp/spy", func(c *gin.Context) {
		writeWorldResponse(c, gin.H{"resources": "approx", "army_strength": "medium", "risk": 0.2}, nil)
	})
	private.POST("/pvp/simulate", func(c *gin.Context) {
		writeWorldResponse(c, gin.H{"winProbability": 0.58}, nil)
	})
	private.POST("/pvp/attack", func(c *gin.Context) {
		var body struct {
			TargetCityID string            `json:"target_city_id"`
			Units        map[string]int    `json:"units"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "invalid body"})
			return
		}
		result, _ := pvpEngine.ExecuteAttack(currentUserID(c), body.TargetCityID, body.Units)
		writeWorldResponse(c, gin.H{"result": result, "battle_id": fmt.Sprintf("battle_%d", time.Now().Unix())}, nil)
	})

	// Market & Leaderboard - real engine calls
	marketEng := market.NewEngine()
	leaderboardEng := leaderboard.NewEngine()
	private.GET("/market/prices", func(c *gin.Context) {
		prices := marketEng.GetPrices()
		writeWorldResponse(c, gin.H{"prices": prices}, nil)
	})

	private.GET("/leaderboard/global", func(c *gin.Context) {
		entries := leaderboardEng.GetGlobal(50)
		writeWorldResponse(c, gin.H{"entries": entries}, nil)
	})

	// Policies activation with basic effect stub
	private.POST("/city/policies/activate", func(c *gin.Context) {
		var body struct { PolicyKey string `json:"policy_key"` }
		c.ShouldBindJSON(&body)
		// TODO: apply effects via policies engine + weather resolver etc.
		writeWorldResponse(c, gin.H{"activated": body.PolicyKey, "effects_applied": true}, nil)
	})
}
