package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

// StellaActivationMiddleware enforces the local StellaGate activation gate.
// When no Cloud URL is configured the middleware is inert, preserving normal
// self-hosted 3x-ui compatibility. When Cloud is configured, only activation
// endpoints are reachable until this VPS has claimed an invite successfully.
func StellaActivationMiddleware() gin.HandlerFunc {
	activation := &service.StellaLocalActivationService{}
	return func(c *gin.Context) {
		if activationPathAllowedWhileLocked(c.Request.URL.Path) || activation.IsUnlocked() {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "StellaGate-UI is not activated",
			"error":   "not_activated",
		})
	}
}

func activationPathAllowedWhileLocked(path string) bool {
	return strings.Contains(path, "/panel/api/stella/activation/status") ||
		strings.Contains(path, "/panel/api/stella/activation/claim") ||
		strings.Contains(path, "/panel/api/stella/activation/check")
}
