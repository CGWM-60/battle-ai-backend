package router

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	contextUserIDKey     = "auth.user_id"
	contextUserEmailKey  = "auth.email"
	contextUserPseudoKey = "auth.pseudo"
)

type requestQueue struct {
	slots   chan struct{}
	timeout time.Duration
}

func newRequestQueue(maxConcurrent int, timeout time.Duration) *requestQueue {
	if maxConcurrent <= 0 {
		maxConcurrent = 8
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &requestQueue{
		slots:   make(chan struct{}, maxConcurrent),
		timeout: timeout,
	}
}

func (q *requestQueue) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		timer := time.NewTimer(q.timeout)
		defer timer.Stop()

		select {
		case q.slots <- struct{}{}:
			defer func() { <-q.slots }()
			c.Header("X-Queue-Status", "accepted")
			c.Next()
		case <-timer.C:
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "server busy, retry later",
			})
		case <-c.Request.Context().Done():
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{
				"error": "request cancelled while waiting",
			})
		}
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(c.Request.URL.Path, "/admin/_next/static/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	}
}

func requestBodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestLimit := requestBodyLimitForPath(c.Request.Method, c.Request.URL.Path, limit)
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, requestLimit)
		c.Next()
	}
}

func requestBodyLimitForPath(method, requestPath string, defaultLimit int64) int64 {
	if method == http.MethodPost &&
		strings.HasPrefix(requestPath, "/admin/api/roleplay/quests/") &&
		strings.HasSuffix(requestPath, "/images") {
		if uploadLimit := service.RolePlaySceneMaxUploadRequestBytes(); uploadLimit > defaultLimit {
			return uploadLimit
		}
	}
	return defaultLimit
}

func jwtAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}

			return []byte(getEnv("JWT_SECRET", "dev-secret-change-me")), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			return
		}

		userID, err := strconv.ParseUint(sub, 10, 64)
		if err != nil || userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			return
		}

		c.Set(contextUserIDKey, uint(userID))
		if email, ok := claims["email"].(string); ok {
			c.Set(contextUserEmailKey, email)
		}
		if pseudo, ok := claims["pseudo"].(string); ok {
			c.Set(contextUserPseudoKey, pseudo)
		}

		c.Next()
	}
}

func currentUserID(c *gin.Context) uint {
	userID, _ := c.Get(contextUserIDKey)
	if value, ok := userID.(uint); ok {
		return value
	}

	return 0
}

func adminAPIAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		expectedSecret := getEnv("ADMIN_API_SECRET", "")
		if expectedSecret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "ADMIN_API_SECRET is not configured"})
			return
		}

		providedSecret := c.GetHeader("X-Admin-Secret")
		if providedSecret == "" || subtle.ConstantTimeCompare([]byte(providedSecret), []byte(expectedSecret)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin secret required"})
			return
		}

		c.Next()
	}
}

func maxConcurrentRequests() int {
	value, err := strconv.Atoi(getEnv("APP_MAX_CONCURRENT_REQUESTS", "8"))
	if err != nil || value <= 0 {
		return 8
	}

	return value
}

func queueTimeout() time.Duration {
	value, err := strconv.Atoi(getEnv("APP_QUEUE_TIMEOUT_SECONDS", "10"))
	if err != nil || value <= 0 {
		return 10 * time.Second
	}

	return time.Duration(value) * time.Second
}

func maxBodyBytes() int64 {
	value, err := strconv.ParseInt(getEnv("APP_MAX_BODY_BYTES", "10485760"), 10, 64)
	if err != nil || value <= 0 {
		return 10485760
	}

	return value
}
