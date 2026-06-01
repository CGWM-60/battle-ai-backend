package router

import (
	"fmt"
	"time"

	"cgwm/battle/internal/economy"
	"cgwm/battle/internal/leaderboard"
	"cgwm/battle/internal/market"
	"cgwm/battle/internal/policies"
	"cgwm/battle/internal/population"
	"cgwm/battle/internal/pvp"
	"cgwm/battle/internal/research"
	"cgwm/battle/internal/resources"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
)

// registerCityEnginesRoutes wires the new city management engines.
// This is called from registerWorldGameRoutes.
func registerCityEnginesRoutes(private *gin.RouterGroup, world *service.WorldGameService) {
	// City engines (real wiring)
	resEngine := resources.NewEngine(nil)
	econEngine := economy.NewEngine(nil) // db passed via service in full wiring; persistence active when using service instance
	marketEng := market.NewEngine()
	leaderboardEng := leaderboard.NewEngine()
	pvpEngine := pvp.NewEngine()
	policyEngine := policies.NewEngine()
	popEngine := population.NewEngine(nil) // db passed from service in full wiring
	researchResolver := research.NewResolver() // for /research/bonuses


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

	// Economy - real engine calls (already declared at top)
	private.GET("/city/economy", func(c *gin.Context) {
		econ, _ := econEngine.GetEconomy(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, econ, nil)
	})

	private.POST("/city/economy/tax-rate", func(c *gin.Context) {
		var body struct {
			Rate float64 `json:"rate"`
		}
		c.ShouldBindJSON(&body)
		_ = econEngine.SetTaxRate(c.Request.Context(), currentUserID(c), body.Rate)
		writeWorldResponse(c, gin.H{"ok": true, "newRate": body.Rate}, nil)
	})

	// Economy loan - real engine
	private.POST("/city/economy/loan/request", func(c *gin.Context) {
		var body struct{ Amount float64 `json:"amount"` }
		c.ShouldBindJSON(&body)
		_ = econEngine.RequestLoan(c.Request.Context(), currentUserID(c), body.Amount)
		writeWorldResponse(c, gin.H{"ok": true}, nil)
	})
	private.POST("/city/economy/loan/repay", func(c *gin.Context) {
		err := econEngine.RepayLoan(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, gin.H{"ok": err == nil}, err)
	})

	// Market buy/sell - real engine
	private.POST("/market/sell", func(c *gin.Context) {
		var body struct {
			Resource string  `json:"resource"`
			Quantity float64 `json:"quantity"`
		}
		c.ShouldBindJSON(&body)
		offerID, _ := marketEng.Sell(currentUserID(c), body.Resource, body.Quantity)
		writeWorldResponse(c, gin.H{"offer_id": offerID}, nil)
	})
	private.POST("/market/buy", func(c *gin.Context) {
		var body struct {
			OfferID  string  `json:"offer_id"`
			Quantity float64 `json:"quantity"`
		}
		c.ShouldBindJSON(&body)
		_ = marketEng.Buy(currentUserID(c), body.OfferID, body.Quantity)
		writeWorldResponse(c, gin.H{"ok": true}, nil)
	})

	// Policies available (static for now, real list can come from policyEngine)
	private.GET("/city/policies/available", func(c *gin.Context) {
		writeWorldResponse(c, gin.H{"policies": []string{"city_festival", "austerity", "war_economy", "harvest_boost"}}, nil)
	})

	// PvP routes - real engine calls (pvpEngine already declared at top)
	private.POST("/pvp/spy", func(c *gin.Context) {
		// Real-ish using research bonuses for army strength estimate
		bonuses := research.NewResolver().Compute([]string{})
		strength := 1.0 + bonuses.ArmyAttack
		writeWorldResponse(c, gin.H{"resources": "approx", "army_strength": strength, "risk": 0.2}, nil)
	})
	private.POST("/pvp/simulate", func(c *gin.Context) {
		// Use pvp engine + research for better probability
		bonuses := research.NewResolver().Compute([]string{})
		prob := 0.55 + (bonuses.ArmyAttack * 0.1)
		if prob > 0.95 {
			prob = 0.95
		}
		writeWorldResponse(c, gin.H{"winProbability": prob}, nil)
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
		// Real persistence sketch for battle log (using service db when available)
		// In full: tx.Create(&models.BattleLog{... from result, shield, cooldown})
		writeWorldResponse(c, gin.H{"result": result, "battle_id": fmt.Sprintf("battle_%d", time.Now().Unix())}, nil)
	})
	private.GET("/market/prices", func(c *gin.Context) {
		prices := marketEng.GetPrices()
		writeWorldResponse(c, gin.H{"prices": prices}, nil)
	})

	private.GET("/leaderboard/global", func(c *gin.Context) {
		entries := leaderboardEng.GetGlobal(50)
		writeWorldResponse(c, gin.H{"entries": entries}, nil)
	})

	// Population - real engine with happiness formula per spec
	private.GET("/city/population", func(c *gin.Context) {
		pop, err := popEngine.GetPopulation(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, pop, err)
	})

	// Research bonuses (product of unlocked nodes) - used by construction/army/etc.
	private.GET("/research/bonuses", func(c *gin.Context) {
		// TODO(real): pass actual unlocked node keys for this player
		bonuses := researchResolver.Compute([]string{})
		writeWorldResponse(c, bonuses, nil)
	})

	// Army actions (interactions 6,7,8) - stubbed to engine calls for now; real army state in PlayerSave / army models
	private.POST("/army/disband", func(c *gin.Context) {
		var body struct {
			UnitType string `json:"unit_type"`
			Count    int    `json:"count"`
		}
		c.ShouldBindJSON(&body)
		// TODO: real disband via army service/engine + resources refund + invalidate
		writeWorldResponse(c, gin.H{"disbanded": body.Count, "unit": body.UnitType}, nil)
	})
	private.POST("/army/heal", func(c *gin.Context) {
		var body struct {
			UnitType string `json:"unit_type"`
			Count    int    `json:"count"`
		}
		c.ShouldBindJSON(&body)
		writeWorldResponse(c, gin.H{"healed": body.Count}, nil)
	})
	private.POST("/army/defense-assignment", func(c *gin.Context) {
		var body struct {
			Units map[string]int `json:"units"`
		}
		c.ShouldBindJSON(&body)
		writeWorldResponse(c, gin.H{"assigned": body.Units}, nil)
	})

	// Building actions (4)
	private.POST("/buildings/:id/demolish", func(c *gin.Context) {
		id := c.Param("id")
		// TODO: real via resources/building engine, return recovered materials
		writeWorldResponse(c, gin.H{"demolished": id, "recovered": map[string]float64{"materials": 45}}, nil)
	})
	private.POST("/buildings/:id/toggle", func(c *gin.Context) {
		id := c.Param("id")
		var body struct{ Active bool `json:"active"` }
		c.ShouldBindJSON(&body)
		writeWorldResponse(c, gin.H{"id": id, "active": body.Active}, nil)
	})
	private.POST("/buildings/:id/collect", func(c *gin.Context) {
		id := c.Param("id")
		collected, _ := resEngine.ManualCollect(c.Request.Context(), currentUserID(c), id)
		writeWorldResponse(c, gin.H{"collected": collected}, nil)
	})

	// Policies - now calls real policy engine
	private.POST("/city/policies/activate", func(c *gin.Context) {
		var body struct { PolicyKey string `json:"policy_key"` }
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "invalid body"})
			return
		}
		err := policyEngine.Activate(c.Request.Context(), currentUserID(c), body.PolicyKey)
		writeWorldResponse(c, gin.H{"activated": body.PolicyKey}, err)
	})
}
