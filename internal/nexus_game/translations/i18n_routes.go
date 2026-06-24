package translations

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// registerI18nRoutes expose les endpoints unifiés /api/i18n/* pour le client Flutter.
func registerI18nRoutes(r *gin.Engine, svc TranslationService) {
	r.GET("/api/i18n/locales", func(c *gin.Context) {
		catalog, err := svc.GetSupportedLocaleCatalog(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		locales := make([]gin.H, 0, len(catalog))
		for _, item := range catalog {
			locales = append(locales, gin.H{
				"locale":       item.Code,
				"languageCode": languageFromLocale(item.Code),
				"countryCode":  countryFromLocale(item.Code),
				"label":        item.Label,
				"flagEmoji":    flagForLocale(item.Code),
				"enabled":      true,
				"isDefault":    item.Code == "fr" || item.Code == "fr-FR",
			})
		}
		c.JSON(http.StatusOK, locales)
	})

	r.GET("/api/i18n/bundle", func(c *gin.Context) {
		locale := c.DefaultQuery("locale", "fr")
		version := c.DefaultQuery("version", "app")
		lang := languageFromLocale(locale)
		data, err := svc.GetTranslations(c.Request.Context(), lang, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("ETag", version+"-"+locale+"-"+time.Now().UTC().Format("2006-01-02"))
		c.JSON(http.StatusOK, gin.H{
			"locale":         locale,
			"version":        time.Now().UTC().Format(time.RFC3339),
			"fallbackLocale": "fr-FR",
			"appVersion":     version,
			"translations":   data,
		})
	})

	r.GET("/api/i18n/changed", func(c *gin.Context) {
		locale := c.DefaultQuery("locale", "fr")
		since := c.Query("since")
		lang := languageFromLocale(locale)
		data, err := svc.GetTranslations(c.Request.Context(), lang, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"locale":       locale,
			"since":        since,
			"changed":      true,
			"translations": data,
		})
	})
}

func languageFromLocale(locale string) string {
	if len(locale) >= 2 {
		return locale[:2]
	}
	return locale
}

func countryFromLocale(locale string) string {
	parts := splitLocale(locale)
	if len(parts) > 1 {
		return parts[1]
	}
	switch languageFromLocale(locale) {
	case "en":
		return "US"
	case "de":
		return "DE"
	default:
		return "FR"
	}
}

func splitLocale(locale string) []string {
	out := make([]string, 0, 2)
	cur := ""
	for _, ch := range locale {
		if ch == '-' || ch == '_' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(ch)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func flagForLocale(locale string) string {
	switch languageFromLocale(locale) {
	case "en":
		return "🇬🇧"
	case "de":
		return "🇩🇪"
	case "es":
		return "🇪🇸"
	default:
		return "🇫🇷"
	}
}