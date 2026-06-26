package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

// StellaActivationMiddleware enforces the local StellaGate activation gate.
// StellaGate-UI is locked by default: a configured StellaGate Cloud URL and a
// successful invite claim are required before core panel APIs are reachable.
// Only activation endpoints stay open while the local panel is locked.
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
