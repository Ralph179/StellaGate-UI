package controller

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/web/session"
)

// StellaController exposes a compact, stable API for the single-VPS product
// surface. Advanced 3x-ui APIs remain untouched under their current paths.
type StellaController struct {
	service  service.StellaService
	server   service.ServerService
	inbounds service.InboundService
	settings service.SettingService
}

func NewStellaController(g *gin.RouterGroup) *StellaController {
	a := &StellaController{}
	a.routes(g)
	return a
}
func (a *StellaController) routes(g *gin.RouterGroup) {
	g.GET("/vps/status", a.status)
	g.GET("/subscription", a.subscription)
	g.POST("/subscription/reset", a.resetSubscription)
	g.POST("/node/restart", a.restart)
	g.POST("/node/reset", a.reset)
	g.POST("/protocol/switch", a.switchProtocol)
	g.GET("/traffic/summary", a.traffic)
}
func stellaUser(c *gin.Context) (int, bool) {
	u := session.GetLoginUser(c)
	if u == nil {
		jsonMsg(c, "authentication required", fmt.Errorf("no user"))
		return 0, false
	}
	return u.Id, true
}
func stellaProtocol(ibProtocol string) string {
	if ibProtocol == "hysteria" {
		return "hysteria2"
	}
	if ibProtocol == "vless" {
		return "vless-reality"
	}
	return ""
}
func (a *StellaController) status(c *gin.Context) {
	uid, ok := stellaUser(c)
	if !ok {
		return
	}
	ib, err := a.service.StellaInbound(uid)
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	status := a.server.RefreshStatus()
	protocol, port := "", 0
	if ib != nil {
		protocol, port = stellaProtocol(string(ib.Protocol)), ib.Port
	}
	subscriptionLink := ""
	if ib != nil {
		subscriptionLink, _, _ = a.stellaSubscriptionLink(c, ib)
	}
	jsonObj(c, gin.H{"name": "StellaGate VPS", "ip": status.PublicIP.IPv4, "system": runtime.GOOS + "/" + runtime.GOARCH, "online": status.Xray.State == service.Running, "protocol": protocol, "port": port, "xrayStatus": status.Xray.State, "singBoxStatus": "not-managed", "monthTraffic": gin.H{"up": sumUp(ib), "down": sumDown(ib), "total": sumUp(ib) + sumDown(ib)}, "access": stellaInstallAccess(c, subscriptionLink), "checkedAt": time.Now().UnixMilli()}, nil)
}
func sumUp(ib *model.Inbound) int64 {
	if ib != nil {
		return ib.Up
	}
	return 0
}
func sumDown(ib *model.Inbound) int64 {
	if ib != nil {
		return ib.Down
	}
	return 0
}
func (a *StellaController) subscription(c *gin.Context) {
	uid, ok := stellaUser(c)
	if !ok {
		return
	}
	ib, err := a.service.StellaInbound(uid)
	if err != nil || ib == nil {
		if err == nil {
			err = fmt.Errorf("no StellaGate node exists")
		}
		jsonObj(c, nil, err)
		return
	}
	link, token, err := a.stellaSubscriptionLink(c, ib)
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, gin.H{"link": link, "token": token, "qrData": link}, nil)
}

func (a *StellaController) stellaSubscriptionLink(c *gin.Context, ib *model.Inbound) (string, string, error) {
	clients, err := a.inbounds.GetClients(ib)
	if err != nil {
		return "", "", err
	}
	if len(clients) == 0 {
		return "", "", fmt.Errorf("subscription client not found")
	}
	token := clients[0].SubID
	base, baseErr := a.settings.GetSubURI()
	if baseErr != nil || strings.TrimSpace(base) == "" {
		// Do not derive a subscription URL from the panel listener.  The
		// subscription server normally has its own port, so reuse the existing
		// SettingService rule that selects its public scheme, host and port.
		// The one-click installer invokes this endpoint locally; in that case a
		// loopback Host must never leak into the link handed to the user.
		host := c.Request.Host
		if stellaLoopbackHost(host) {
			if publicIP := a.server.RefreshStatus().PublicIP.IPv4; publicIP != "" {
				host = publicIP
			}
		}
		subPath, pathErr := a.settings.GetSubPath()
		if pathErr != nil || strings.TrimSpace(subPath) == "" {
			subPath = "/sub/"
		}
		base = strings.TrimRight(a.settings.BuildSubURIBase(host), "/") + "/" + strings.TrimLeft(subPath, "/")
	}
	link := strings.TrimSuffix(base, "/") + "/" + token
	return link, token, nil
}

func stellaLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if candidate, _, err := net.SplitHostPort(host); err == nil {
		host = candidate
	}
	host = strings.Trim(host, "[]")
	return strings.EqualFold(host, "localhost") || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}

// stellaInstallAccess exposes the credentials generated during one-click
// installation to an already authenticated panel session. They are read from
// the root-only result file; when the panel was installed manually, only the
// current username and an empty password are returned.
func stellaInstallAccess(c *gin.Context, subscriptionLink string) gin.H {
	username, password, panelURL := "", "", ""
	if raw, err := os.ReadFile("/etc/x-ui/install-result.env"); err == nil {
		username = stellaInstallValue(string(raw), "XUI_USERNAME")
		password = stellaInstallValue(string(raw), "XUI_PASSWORD")
		panelURL = stellaInstallValue(string(raw), "XUI_ACCESS_URL")
	}
	if username == "" {
		if user := session.GetLoginUser(c); user != nil {
			username = user.Username
		}
	}
	if panelURL == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		panelURL = scheme + "://" + c.Request.Host
	}
	return gin.H{"panelUrl": panelURL, "username": username, "password": password, "subscriptionLink": subscriptionLink}
}

func stellaInstallValue(raw, key string) string {
	prefix := key + "="
	for _, line := range strings.Split(raw, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if decoded, err := strconv.Unquote(value); err == nil {
			return decoded
		}
		return strings.Trim(value, "'\"")
	}
	return ""
}
func (a *StellaController) resetSubscription(c *gin.Context) {
	uid, ok := stellaUser(c)
	if !ok {
		return
	}
	ib, err := a.service.ResetSubscription(uid)
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, gin.H{"inboundId": ib.Id}, nil)
}
func (a *StellaController) restart(c *gin.Context) {
	if err := a.service.Restart(); err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, gin.H{"restarted": true}, nil)
}
func (a *StellaController) reset(c *gin.Context) {
	uid, ok := stellaUser(c)
	if !ok {
		return
	}
	var req struct {
		ResetType string `json:"resetType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonObj(c, nil, err)
		return
	}
	ib, err := a.service.Reset(uid, req.ResetType)
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, gin.H{"inboundId": ib.Id}, nil)
}
func (a *StellaController) switchProtocol(c *gin.Context) {
	uid, ok := stellaUser(c)
	if !ok {
		return
	}
	var req struct {
		Protocol string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonObj(c, nil, err)
		return
	}
	ib, err := a.service.SwitchProtocol(uid, req.Protocol)
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, gin.H{"protocol": stellaProtocol(string(ib.Protocol)), "port": ib.Port}, nil)
}
func (a *StellaController) traffic(c *gin.Context) {
	uid, ok := stellaUser(c)
	if !ok {
		return
	}
	ib, err := a.service.StellaInbound(uid)
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	up, down := sumUp(ib), sumDown(ib)
	online := 0
	if ib != nil {
		for _, email := range a.inbounds.GetOnlineClients() {
			if email == "stellagate" {
				online++
			}
		}
	}
	jsonObj(c, gin.H{"today": gin.H{"up": int64(0), "down": int64(0), "total": int64(0)}, "month": gin.H{"up": up, "down": down, "total": up + down}, "total": gin.H{"up": up, "down": down, "total": up + down}, "onlineClients": online}, nil)
}
