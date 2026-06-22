package controller

import (
	"fmt"
	"runtime"
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
	jsonObj(c, gin.H{"name": "StellaGate VPS", "ip": status.PublicIP.IPv4, "system": runtime.GOOS + "/" + runtime.GOARCH, "online": status.Xray.State == service.Running, "protocol": protocol, "port": port, "xrayStatus": status.Xray.State, "singBoxStatus": "not-managed", "monthTraffic": gin.H{"up": sumUp(ib), "down": sumDown(ib), "total": sumUp(ib) + sumDown(ib)}, "checkedAt": time.Now().UnixMilli()}, nil)
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
	clients, err := a.inbounds.GetClients(ib)
	if err != nil || len(clients) == 0 {
		if err == nil {
			err = fmt.Errorf("subscription client not found")
		}
		jsonObj(c, nil, err)
		return
	}
	token := clients[0].SubID
	base, baseErr := a.settings.GetSubURI()
	if baseErr != nil || strings.TrimSpace(base) == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		base = fmt.Sprintf("%s/sub/", scheme+"://"+c.Request.Host)
	}
	link := strings.TrimSuffix(base, "/") + "/" + token
	jsonObj(c, gin.H{"link": link, "token": token, "qrData": link}, nil)
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
