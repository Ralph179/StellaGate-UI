package service

import (
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	"gorm.io/gorm"
)

const (
	StellaCloudConfigPath = "/etc/x-ui/stellagate-cloud.json"
	StellaActivationPath  = "/etc/x-ui/stellagate-activation.json"
	StellaAgentVersion    = "stellagate-ui-local-1"
)

const stellaActivationCheckTTL = 30 * time.Second

var stellaActivationCheckMu sync.Mutex

type StellaCloudConfig struct {
	CloudURL string `json:"cloud_url"`
}

type StellaActivationState struct {
	CloudURL        string `json:"cloud_url"`
	ServerID        string `json:"server_id"`
	DeviceID        string `json:"device_id"`
	ActivationToken string `json:"activation_token"`
	ActivatedAt     int64  `json:"activated_at"`
	LastCheckedAt   int64  `json:"last_checked_at"`
	Revoked         bool   `json:"revoked,omitempty"`
	RevokedReason   string `json:"revoked_reason,omitempty"`
}

type StellaActivationStatus struct {
	Activated bool   `json:"activated"`
	CloudURL  string `json:"cloud_url"`
	ServerID  string `json:"server_id,omitempty"`
	CheckedAt int64  `json:"checked_at,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type StellaLocalActivationService struct {
	server ServerService
	stella StellaService
}

func (s *StellaLocalActivationService) ReadCloudConfig() (*StellaCloudConfig, error) {
	b, err := os.ReadFile(StellaCloudConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		return &StellaCloudConfig{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg StellaCloudConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	cfg.CloudURL = strings.TrimSpace(cfg.CloudURL)
	_ = os.Chmod(StellaCloudConfigPath, 0600)
	return &cfg, nil
}

func (s *StellaLocalActivationService) ReadActivation() (*StellaActivationState, error) {
	b, err := os.ReadFile(StellaActivationPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state StellaActivationState
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, err
	}
	_ = os.Chmod(StellaActivationPath, 0600)
	return &state, nil
}

func (s *StellaLocalActivationService) Status() (*StellaActivationStatus, error) {
	cfg, err := s.ReadCloudConfig()
	if err != nil {
		return nil, err
	}
	if cfg.CloudURL == "" {
		return &StellaActivationStatus{Activated: false, CloudURL: "", Reason: "cloud_not_configured"}, nil
	}
	state, err := s.ReadActivation()
	if err != nil {
		return nil, err
	}
	if state == nil || state.ActivationToken == "" || state.ServerID == "" {
		return &StellaActivationStatus{Activated: false, CloudURL: cfg.CloudURL, Reason: "not_activated"}, nil
	}
	if state.Revoked {
		reason := state.RevokedReason
		if reason == "" {
			reason = "revoked"
		}
		return &StellaActivationStatus{Activated: false, CloudURL: cfg.CloudURL, Reason: reason, CheckedAt: state.LastCheckedAt}, nil
	}
	if s.shouldRefreshActivation(state) {
		if status, _, err := s.Check(); err == nil && status != nil {
			return status, nil
		} else if err != nil {
			// Temporary Cloud/network outages should not instantly brick an
			// already-activated self-hosted VPS. Definitive invalidation errors
			// are handled inside Check() and persisted as Revoked; only transient
			// failures fall back to the last known local state.
			logger.Warning("StellaGate activation check failed:", activationErrorCode(err))
		}
	}
	return &StellaActivationStatus{Activated: true, CloudURL: cfg.CloudURL, ServerID: state.ServerID, CheckedAt: state.LastCheckedAt}, nil
}

func (s *StellaLocalActivationService) IsUnlocked() bool {
	status, err := s.Status()
	return err == nil && status.Activated
}

func (s *StellaLocalActivationService) Claim(inviteCode string) (*StellaActivationStatus, string, error) {
	inviteCode = strings.TrimSpace(inviteCode)
	if inviteCode == "" {
		return nil, "invite_invalid", common.NewError("invite_invalid")
	}
	cfg, err := s.ReadCloudConfig()
	if err != nil {
		return nil, "cloud_not_configured", err
	}
	if cfg.CloudURL == "" {
		return nil, "cloud_not_configured", common.NewError("cloud_not_configured")
	}
	client, err := NewCloudClient(cfg.CloudURL)
	if err != nil {
		return nil, activationErrorCode(err), err
	}
	deviceID, err := StellaDeviceID()
	if err != nil {
		return nil, "device_id_failed", err
	}
	info := s.localInfo()
	claimResp, err := client.Claim(CloudActivationClaimRequest{
		InviteCode:   inviteCode,
		DeviceID:     deviceID,
		Hostname:     info.Hostname,
		PublicIP:     info.PublicIP,
		OS:           info.OS,
		Arch:         info.Arch,
		PanelVersion: info.PanelVersion,
		AgentVersion: info.AgentVersion,
	})
	if err != nil {
		return nil, activationErrorCode(err), err
	}
	now := time.Now().Unix()
	state := StellaActivationState{
		CloudURL:        cfg.CloudURL,
		ServerID:        claimResp.ServerID,
		DeviceID:        deviceID,
		ActivationToken: claimResp.ActivationToken,
		ActivatedAt:     now,
		LastCheckedAt:   now,
	}
	if err := writeStellaJSON0600(StellaActivationPath, state); err != nil {
		return nil, "activation_save_failed", err
	}
	if err := s.setManagedNodeEnabled(true); err != nil {
		logger.Warning("StellaGate activation succeeded but node re-enable failed:", err)
	}
	return &StellaActivationStatus{Activated: true, CloudURL: cfg.CloudURL, ServerID: state.ServerID, CheckedAt: state.LastCheckedAt}, "", nil
}

func (s *StellaLocalActivationService) Check() (*StellaActivationStatus, string, error) {
	cfg, err := s.ReadCloudConfig()
	if err != nil {
		return nil, "cloud_not_configured", err
	}
	if cfg.CloudURL == "" {
		return &StellaActivationStatus{Activated: false, CloudURL: "", Reason: "cloud_not_configured"}, "cloud_not_configured", nil
	}
	state, err := s.ReadActivation()
	if err != nil {
		return nil, "not_activated", err
	}
	if state == nil || state.ActivationToken == "" {
		return &StellaActivationStatus{Activated: false, CloudURL: cfg.CloudURL, Reason: "not_activated"}, "not_activated", nil
	}
	client, err := NewCloudClient(cfg.CloudURL)
	if err != nil {
		return nil, activationErrorCode(err), err
	}
	info := s.localInfo()
	resp, err := client.Check(state.ActivationToken, CloudActivationCheckRequest{
		ServerID:     state.ServerID,
		DeviceID:     state.DeviceID,
		PanelVersion: info.PanelVersion,
		PublicIP:     info.PublicIP,
	})
	if err != nil {
		code := activationErrorCode(err)
		if reason := activationInvalidationReason(code); reason != "" {
			state.LastCheckedAt = time.Now().Unix()
			_ = s.markRevokedAndLock(state, reason)
			return &StellaActivationStatus{Activated: false, CloudURL: cfg.CloudURL, Reason: reason, CheckedAt: state.LastCheckedAt}, reason, nil
		}
		return nil, code, err
	}
	state.LastCheckedAt = time.Now().Unix()
	if resp.Active != nil && !*resp.Active {
		reason := resp.Reason
		if reason == "" {
			reason = "revoked"
		}
		_ = s.markRevokedAndLock(state, reason)
		return &StellaActivationStatus{Activated: false, CloudURL: cfg.CloudURL, Reason: reason, CheckedAt: state.LastCheckedAt}, reason, nil
	}
	if err := writeStellaJSON0600(StellaActivationPath, state); err != nil {
		return nil, "activation_save_failed", err
	}
	return &StellaActivationStatus{Activated: true, CloudURL: cfg.CloudURL, ServerID: state.ServerID, CheckedAt: state.LastCheckedAt}, "", nil
}

func (s *StellaLocalActivationService) Heartbeat() error {
	state, err := s.ReadActivation()
	if err != nil || state == nil || state.ActivationToken == "" || state.Revoked {
		return err
	}
	cfg, err := s.ReadCloudConfig()
	if err != nil || cfg.CloudURL == "" {
		return err
	}
	client, err := NewCloudClient(cfg.CloudURL)
	if err != nil {
		return err
	}
	payload := s.heartbeatPayload(state)
	resp, err := client.Heartbeat(state.ActivationToken, payload)
	if err != nil {
		code := activationErrorCode(err)
		if reason := activationInvalidationReason(code); reason != "" {
			return s.markRevokedAndLock(state, reason)
		}
		// Temporary Cloud outages must not interrupt the local proxy.
		logger.Warning("StellaGate heartbeat failed:", code)
		return nil
	}
	state.LastCheckedAt = time.Now().Unix()
	if resp.Active != nil && !*resp.Active {
		reason := resp.Reason
		if reason == "" {
			reason = "revoked"
		}
		return s.markRevokedAndLock(state, reason)
	}
	return writeStellaJSON0600(StellaActivationPath, state)
}

func (s *StellaLocalActivationService) shouldRefreshActivation(state *StellaActivationState) bool {
	if state == nil || state.LastCheckedAt <= 0 {
		return true
	}
	return time.Since(time.Unix(state.LastCheckedAt, 0)) >= stellaActivationCheckTTL
}

func (s *StellaLocalActivationService) markRevoked(state *StellaActivationState, reason string) error {
	state.Revoked = true
	state.RevokedReason = reason
	state.LastCheckedAt = time.Now().Unix()
	return writeStellaJSON0600(StellaActivationPath, state)
}

func (s *StellaLocalActivationService) markRevokedAndLock(state *StellaActivationState, reason string) error {
	stellaActivationCheckMu.Lock()
	defer stellaActivationCheckMu.Unlock()

	err := s.markRevoked(state, reason)
	if nodeErr := s.setManagedNodeEnabled(false); nodeErr != nil {
		logger.Warning("StellaGate activation revoked but managed node lock failed:", nodeErr)
	}
	return err
}

func (s *StellaLocalActivationService) setManagedNodeEnabled(enabled bool) error {
	var inbound model.Inbound
	err := database.GetDB().Where("tag LIKE ?", stellaTagPrefix+"%").Order("id ASC").First(&inbound).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if inbound.Enable == enabled {
		return nil
	}
	if err := database.GetDB().Model(&model.Inbound{}).Where("id = ?", inbound.Id).Update("enable", enabled).Error; err != nil {
		return err
	}
	return s.server.RestartXrayService()
}

type stellaLocalInfo struct {
	Hostname     string
	PublicIP     string
	OS           string
	Arch         string
	PanelVersion string
	AgentVersion string
}

func (s *StellaLocalActivationService) localInfo() stellaLocalInfo {
	hostname, _ := os.Hostname()
	status := s.server.RefreshStatus()
	publicIP := ""
	panelVersion := config.GetVersion()
	if status != nil {
		publicIP = status.PublicIP.IPv4
		if publicIP == "" || publicIP == "N/A" {
			publicIP = status.PublicIP.IPv6
		}
		if status.PanelVersion != "" {
			panelVersion = status.PanelVersion
		}
	}
	return stellaLocalInfo{
		Hostname:     hostname,
		PublicIP:     publicIP,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		PanelVersion: panelVersion,
		AgentVersion: StellaAgentVersion,
	}
}

func (s *StellaLocalActivationService) heartbeatPayload(state *StellaActivationState) map[string]any {
	info := s.localInfo()
	status := s.server.RefreshStatus()
	var cpuPct, memPct, diskPct float64
	xrayStatus := ""
	if status != nil {
		cpuPct = status.Cpu
		if status.Mem.Total > 0 {
			memPct = float64(status.Mem.Current) / float64(status.Mem.Total) * 100
		}
		if status.Disk.Total > 0 {
			diskPct = float64(status.Disk.Current) / float64(status.Disk.Total) * 100
		}
		xrayStatus = string(status.Xray.State)
	}
	protocol, port, up, down := s.currentNodeSummary()
	return map[string]any{
		"server_id":        state.ServerID,
		"device_id":        state.DeviceID,
		"hostname":         info.Hostname,
		"public_ip":        info.PublicIP,
		"os":               info.OS,
		"arch":             info.Arch,
		"cpu_pct":          cpuPct,
		"mem_pct":          memPct,
		"disk_pct":         diskPct,
		"xray_status":      xrayStatus,
		"current_protocol": protocol,
		"current_port":     port,
		"upload":           up,
		"download":         down,
		"panel_version":    info.PanelVersion,
		"agent_version":    info.AgentVersion,
	}
}

func (s *StellaLocalActivationService) currentNodeSummary() (protocol string, port int, up int64, down int64) {
	var inbound model.Inbound
	err := database.GetDB().Where("tag LIKE ?", stellaTagPrefix+"%").Order("id ASC").First(&inbound).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || err != nil {
		return "", 0, 0, 0
	}
	switch inbound.Protocol {
	case model.VLESS:
		protocol = "vless-reality"
	case model.Hysteria:
		protocol = "hysteria2"
	default:
		protocol = string(inbound.Protocol)
	}
	return protocol, inbound.Port, inbound.Up, inbound.Down
}

func writeStellaJSON0600(path string, v any) error {
	if err := os.MkdirAll("/etc/x-ui", 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}

func activationErrorCode(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	msg = strings.TrimPrefix(msg, "Error:")
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "cloud_unreachable"
	}
	allowed := map[string]bool{
		"invite_invalid":          true,
		"invite_expired":          true,
		"invite_disabled":         true,
		"invite_used_up":          true,
		"device_already_bound":    true,
		"rate_limited":            true,
		"invalid_request":         true,
		"invalid_token":           true,
		"device_mismatch":         true,
		"expired":                 true,
		"cloud_unreachable":       true,
		"cloud_not_configured":    true,
		"cloud_url_must_be_https": true,
		"invalid_cloud_url":       true,
		"revoked":                 true,
	}
	if allowed[msg] {
		return msg
	}
	return msg
}

func activationInvalidationReason(code string) string {
	switch strings.TrimSpace(code) {
	case "revoked", "expired", "invalid_token", "device_mismatch", "invalid_request":
		return code
	case "cloud_http_400":
		return "invalid_request"
	case "cloud_http_401", "cloud_http_403", "cloud_http_404":
		return "invalid_token"
	default:
		return ""
	}
}
