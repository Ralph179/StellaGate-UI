package controller

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/web/session"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
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
	g.POST("/node/random-port", a.randomPort)
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
	up, down, _, _ := a.stellaTrafficTotals(ib)
	jsonObj(c, gin.H{"name": "StellaGate VPS", "ip": status.PublicIP.IPv4, "system": runtime.GOOS + "/" + runtime.GOARCH, "online": status.Xray.State == service.Running, "protocol": protocol, "port": port, "xrayStatus": status.Xray.State, "singBoxStatus": "not-managed", "monthTraffic": gin.H{"up": up, "down": down, "total": up + down}, "checkedAt": time.Now().UnixMilli()}, nil)
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
func (a *StellaController) randomPort(c *gin.Context) {
	uid, ok := stellaUser(c)
	if !ok {
		return
	}
	ib, err := a.service.RandomizePort(uid)
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, gin.H{"protocol": stellaProtocol(string(ib.Protocol)), "port": ib.Port}, nil)
}
func (a *StellaController) reset(c *gin.Context) {
	uid, ok := stellaUser(c)
	if !ok {
		return
	}
	var req struct {
		ResetType string `json:"resetType"`
	}
	if err := c.ShouldBind(&req); err != nil {
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
	if err := c.ShouldBind(&req); err != nil {
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
	up, down, lastOnline, trafficErr := a.stellaTrafficTotals(ib)
	if trafficErr != nil {
		jsonObj(c, nil, trafficErr)
		return
	}
	online := a.stellaOnlineClients(ib, lastOnline)
	// 3x-ui stores StellaGate's first-version accounting as cumulative client
	// counters. Until a persisted daily baseline is added, surface the same
	// real counters in the "today" bucket so the simplified panel never looks
	// broken or blank; clients still get accurate month/total usage.
	bucket := gin.H{"up": up, "down": down, "total": up + down}
	jsonObj(c, gin.H{"today": bucket, "month": bucket, "total": bucket, "onlineClients": online, "estimatedToday": true}, nil)
}

func (a *StellaController) stellaTrafficTotals(ib *model.Inbound) (up int64, down int64, lastOnline int64, err error) {
	up, down = sumUp(ib), sumDown(ib)
	if ib == nil {
		return up, down, 0, nil
	}
	emailSet, err := a.stellaClientEmailSet(ib)
	if err != nil || len(emailSet) == 0 {
		return up, down, 0, err
	}
	emails := make([]string, 0, len(emailSet))
	for email := range emailSet {
		emails = append(emails, email)
	}
	var rows []xray.ClientTraffic
	if err = database.GetDB().Model(xray.ClientTraffic{}).Where("email IN ?", emails).Find(&rows).Error; err != nil {
		return up, down, 0, err
	}
	var clientUp, clientDown int64
	for _, row := range rows {
		clientUp += row.Up
		clientDown += row.Down
		if row.LastOnline > lastOnline {
			lastOnline = row.LastOnline
		}
	}
	if clientUp+clientDown > up+down {
		up, down = clientUp, clientDown
	}
	return up, down, lastOnline, nil
}

func (a *StellaController) stellaClientEmailSet(ib *model.Inbound) (map[string]struct{}, error) {
	clients, err := a.inbounds.GetClients(ib)
	if err != nil {
		return nil, err
	}
	emailSet := make(map[string]struct{}, len(clients))
	for _, client := range clients {
		email := strings.TrimSpace(client.Email)
		if email != "" {
			emailSet[email] = struct{}{}
		}
	}
	return emailSet, nil
}

func (a *StellaController) stellaOnlineClients(ib *model.Inbound, lastOnline int64) int {
	if ib == nil {
		return 0
	}
	emailSet, err := a.stellaClientEmailSet(ib)
	if err != nil || len(emailSet) == 0 {
		return 0
	}
	seen := map[string]struct{}{}
	for _, email := range a.inbounds.GetOnlineClients() {
		if _, ok := emailSet[email]; ok {
			seen[email] = struct{}{}
		}
	}
	if len(seen) > 0 {
		return len(seen)
	}
	// Hysteria2 can be harder to attribute as a named online client in some
	// Xray snapshots. If the managed client's traffic was updated recently,
	// show one online client instead of a misleading zero.
	if lastOnline > 0 && time.Now().UnixMilli()-lastOnline <= int64(2*time.Minute/time.Millisecond) {
		return 1
	}
	return 0
}
