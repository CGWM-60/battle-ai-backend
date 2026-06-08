package translations

import (
	"fmt"
	"net/http"

	"cgwm/battle/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes enregistre les endpoints publics et semi-publics pour les traductions.
// Suivant la spec AGENTS.md POINT 02.
func RegisterRoutes(r *gin.Engine, database *gorm.DB) {
	svc := NewTranslationService(database)

	// Public bootstrap pour le client Flutter au démarrage
	r.GET("/api/translations/bootstrap", func(c *gin.Context) {
		locale := c.DefaultQuery("locale", "fr")
		// Pour l'instant on retourne tout, domaines optionnels via query si besoin plus tard
		data, err := svc.GetTranslations(c.Request.Context(), locale, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"locale":       locale,
			"translations": data,
		})
	})

	// Liste des domaines
	r.GET("/api/translations/domains", func(c *gin.Context) {
		domains, err := svc.GetDomains(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"domains": domains})
	})

	// Traductions pour un domaine spécifique
	r.GET("/api/translations/domain/:domain", func(c *gin.Context) {
		domain := c.Param("domain")
		locale := c.DefaultQuery("locale", "fr")
		data, err := svc.GetDomainTranslations(c.Request.Context(), domain, locale)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"domain":       domain,
			"locale":       locale,
			"translations": data,
		})
	})

	// Auth requis pour mettre à jour la locale utilisateur
	// On utilise un middleware simple ici (le vrai jwtAuth est dans le router principal)
	r.PUT("/api/user/locale", func(c *gin.Context) {
		// Pour le point 02, on accepte user_id dans le body ou query pour simplicité.
		// En vrai, on extrairait du JWT.
		var req struct {
			UserID uint   `json:"user_id" form:"user_id"`
			Locale string `json:"locale" form:"locale" binding:"required"`
		}
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "locale requis"})
			return
		}
		if req.UserID == 0 {
			// Fallback pour test sans auth complète
			req.UserID = 1
		}
		if err := svc.SetUserLocale(c.Request.Context(), req.UserID, req.Locale); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "locale": req.Locale})
	})

	// Log de clé manquante (appelé par le client quand une clé n'est pas trouvée)
	r.POST("/api/translations/missing", func(c *gin.Context) {
		var req struct {
			Key    string `json:"key" binding:"required"`
			Locale string `json:"locale" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := svc.LogMissingKey(c.Request.Context(), req.Key, req.Locale); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "logged"})
	})
}

// RegisterAdminRoutes enregistre les endpoints admin pour les traductions.
// Doit être appelé sur un groupe protégé par admin middleware.
func RegisterAdminRoutes(adminGroup *gin.RouterGroup, database *gorm.DB) {
	svc := NewTranslationService(database)

	// Domains admin
	adminGroup.GET("/translations/domains", func(c *gin.Context) {
		domains, err := svc.GetAllDomains(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"domains": domains})
	})
	adminGroup.POST("/translations/domains", func(c *gin.Context) {
		var d models.TranslationDomain
		if err := c.ShouldBindJSON(&d); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := svc.CreateDomain(c.Request.Context(), &d); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, d)
	})

	// Keys admin
	adminGroup.GET("/translations/keys", func(c *gin.Context) {
		keys, err := svc.GetAllKeys(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"keys": keys})
	})
	adminGroup.POST("/translations/keys", func(c *gin.Context) {
		var k models.TranslationKey
		if err := c.ShouldBindJSON(&k); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := svc.CreateKey(c.Request.Context(), &k); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, k)
	})
	adminGroup.PUT("/translations/keys/:id", func(c *gin.Context) {
		id := c.Param("id")
		var k models.TranslationKey
		if err := c.ShouldBindJSON(&k); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// parse id if needed, assume uint for simplicity
		var idUint uint
		fmt.Sscanf(id, "%d", &idUint)
		if err := svc.UpdateKey(c.Request.Context(), idUint, &k); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "updated"})
	})
	adminGroup.DELETE("/translations/keys/:id", func(c *gin.Context) {
		id := c.Param("id")
		var idUint uint
		fmt.Sscanf(id, "%d", &idUint)
		if err := svc.DeleteKey(c.Request.Context(), idUint); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	})

	// Values admin
	adminGroup.GET("/translations/values", func(c *gin.Context) {
		values, err := svc.GetAllValues(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"values": values})
	})
	adminGroup.PUT("/translations/values/:id", func(c *gin.Context) {
		id := c.Param("id")
		var v models.TranslationValue
		if err := c.ShouldBindJSON(&v); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var idUint uint
		fmt.Sscanf(id, "%d", &idUint)
		if err := svc.UpdateValue(c.Request.Context(), idUint, &v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "updated"})
	})

	// Import
	adminGroup.POST("/translations/import/preview", func(c *gin.Context) {
		var rows []models.TranslationImportRow
		if err := c.ShouldBindJSON(&rows); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		previewed, err := svc.PreviewImport(c.Request.Context(), rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"preview": previewed})
	})
	adminGroup.POST("/translations/import/commit", func(c *gin.Context) {
		var req struct {
			ImportID uint `json:"import_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := svc.CommitImport(c.Request.Context(), req.ImportID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "committed"})
	})

	// Other admin
	adminGroup.GET("/translations/imports", func(c *gin.Context) {
		imports, err := svc.GetImports(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"imports": imports})
	})
	adminGroup.GET("/translations/imports/:id", func(c *gin.Context) {
		id := c.Param("id")
		var idUint uint
		fmt.Sscanf(id, "%d", &idUint)
		imp, err := svc.GetImportByID(c.Request.Context(), idUint)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, imp)
	})
	adminGroup.GET("/translations/missing", func(c *gin.Context) {
		logs, err := svc.GetMissing(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"missing": logs})
	})
	adminGroup.GET("/translations/export", func(c *gin.Context) {
		lang := c.DefaultQuery("locale", "fr")
		data, err := svc.ExportTranslations(c.Request.Context(), lang)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, data)
	})
	adminGroup.POST("/translations/batch-update", func(c *gin.Context) {
		var entries []TranslationEntry
		if err := c.ShouldBindJSON(&entries); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := svc.BatchUpdate(c.Request.Context(), entries); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "updated"})
	})
}
