package service

// StellaService is a deliberately small product layer over the existing
// inbound/client/Xray services.  It owns no second configuration format: the
// generated templates are ordinary 3x-ui inbounds and remain editable from
// the advanced UI.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	"gorm.io/gorm"
)

const stellaTagPrefix = "stellagate-"

type StellaService struct {
	inbounds InboundService
	server   ServerService
}

// StellaInbound returns the one inbound managed by the simplified panel. The
// StellaGate product is deliberately single-VPS/single-node: if the initial
// installer created that node under a different panel session, the logged-in
// owner must still see and control it instead of seeing a false "no node"
// state.
func (s *StellaService) StellaInbound(userID int) (*model.Inbound, error) {
	inbounds, err := s.inbounds.GetInbounds(userID)
	if err != nil {
		return nil, err
	}
	for _, inbound := range inbounds {
		if strings.HasPrefix(inbound.Tag, stellaTagPrefix) {
			return inbound, nil
		}
	}

	var managed model.Inbound
	err = database.GetDB().Where("tag LIKE ?", stellaTagPrefix+"%").Order("id ASC").First(&managed).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &managed, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func freePort(network string, preferred int) (int, error) {
	if preferred > 0 {
		addr := fmt.Sprintf(":%d", preferred)
		if network == "udp" {
			c, err := net.ListenPacket("udp", addr)
			if err == nil {
				_ = c.Close()
				return preferred, nil
			}
		} else {
			l, err := net.Listen("tcp", addr)
			if err == nil {
				_ = l.Close()
				return preferred, nil
			}
		}
	}
	if network == "udp" {
		c, err := net.ListenPacket("udp", ":0")
		if err != nil {
			return 0, common.NewError("UDP is unavailable on this VPS: ", err)
		}
		defer c.Close()
		return c.LocalAddr().(*net.UDPAddr).Port, nil
	}
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func (s *StellaService) template(protocol string, userID int, existing *model.Inbound) (*model.Inbound, error) {
	client := model.Client{ID: uuid.NewString(), Auth: uuid.NewString(), Email: "stellagate", SubID: uuid.NewString(), Enable: true}
	if existing != nil {
		clients, err := s.inbounds.GetClients(existing)
		if err != nil {
			return nil, err
		}
		if len(clients) > 0 {
			client = clients[0]
		}
	}

	ib := &model.Inbound{UserId: userID, Enable: true, Listen: "", ShareAddrStrategy: "node", Sniffing: `{"enabled":true,"destOverride":["http","tls","quic"]}`}
	switch protocol {
	case "vless-reality":
		port, err := freePort("tcp", 443)
		if err != nil {
			return nil, err
		}
		keyPair, err := s.server.GetNewX25519Cert()
		if err != nil {
			return nil, common.NewError("could not generate Reality key pair: ", err)
		}
		keys := keyPair.(map[string]any)
		shortID, err := randomHex(8)
		if err != nil {
			return nil, err
		}
		client.ID, client.Auth, client.Flow = uuid.NewString(), "", "xtls-rprx-vision"
		ib.Port, ib.Protocol, ib.Remark, ib.Tag = port, model.VLESS, "StellaGate · VLESS Reality", stellaTagPrefix+"vless-reality"
		ib.Settings = mustJSON(map[string]any{"clients": []model.Client{client}, "decryption": "none"})
		ib.StreamSettings = mustJSON(map[string]any{"network": "tcp", "security": "reality", "realitySettings": map[string]any{
			"show": false, "dest": "www.cloudflare.com:443", "xver": 0, "serverNames": []string{"www.cloudflare.com"},
			"privateKey": keys["privateKey"], "publicKey": keys["publicKey"], "shortIds": []string{shortID},
		}})
	case "hysteria2":
		port, err := freePort("udp", 0)
		if err != nil {
			return nil, err
		}
		client.ID, client.Flow, client.Auth = "", "", uuid.NewString()
		ib.Port, ib.Protocol, ib.Remark, ib.Tag = port, model.Hysteria, "StellaGate · Hysteria2", stellaTagPrefix+"hysteria2"
		ib.Settings = mustJSON(map[string]any{"version": 2, "clients": []model.Client{client}})
		// Xray obtains the certificate through the normal panel TLS settings.
		// The handler reports any missing/invalid TLS material when it restarts.
		ib.StreamSettings = mustJSON(map[string]any{"network": "hysteria", "security": "tls", "tlsSettings": map[string]any{"serverName": "stellagate"}})
	default:
		return nil, common.NewError("unsupported protocol")
	}
	return ib, nil
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func (s *StellaService) SwitchProtocol(userID int, protocol string) (*model.Inbound, error) {
	existing, err := s.StellaInbound(userID)
	if err != nil {
		return nil, err
	}
	ownerID := userID
	if existing != nil {
		ownerID = existing.UserId
	}
	ib, err := s.template(protocol, ownerID, existing)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		created, _, err := s.inbounds.AddInbound(ib)
		if err != nil {
			return nil, err
		}
		ib = created
	} else {
		ib.Id = existing.Id
		updated, _, err := s.inbounds.UpdateInbound(ib)
		if err != nil {
			return nil, err
		}
		ib = updated
	}
	if err := s.server.RestartXrayService(); err != nil {
		return nil, common.NewError("configuration saved but proxy service could not start: ", err)
	}
	return ib, nil
}

func (s *StellaService) Restart() error { return s.server.RestartXrayService() }

func (s *StellaService) Reset(userID int, resetType string) (*model.Inbound, error) {
	if resetType == "light" {
		return s.StellaInbound(userID)
	}
	ib, err := s.StellaInbound(userID)
	if err != nil {
		return nil, err
	}
	if ib == nil {
		return nil, common.NewError("no StellaGate node exists; choose a protocol first")
	}
	if resetType == "deep" {
		protocol := "vless-reality"
		if ib.Protocol == model.Hysteria {
			protocol = "hysteria2"
		}
		return s.SwitchProtocol(userID, protocol)
	}
	if resetType != "normal" {
		return nil, common.NewError("resetType must be light, normal, or deep")
	}
	clients, err := s.inbounds.GetClients(ib)
	if err != nil {
		return nil, err
	}
	if len(clients) == 0 {
		return nil, common.NewError("StellaGate node has no client")
	}
	if ib.Protocol == model.VLESS {
		clients[0].ID = uuid.NewString()
	} else {
		clients[0].Auth = uuid.NewString()
	}
	var settings map[string]any
	_ = json.Unmarshal([]byte(ib.Settings), &settings)
	settings["clients"] = clients
	ib.Settings = mustJSON(settings)
	updated, _, err := s.inbounds.UpdateInbound(ib)
	if err != nil {
		return nil, err
	}
	if err := s.server.RestartXrayService(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *StellaService) ResetSubscription(userID int) (*model.Inbound, error) {
	ib, err := s.StellaInbound(userID)
	if err != nil {
		return nil, err
	}
	if ib == nil {
		return nil, common.NewError("no StellaGate node exists")
	}
	clients, err := s.inbounds.GetClients(ib)
	if err != nil {
		return nil, err
	}
	if len(clients) == 0 {
		return nil, common.NewError("StellaGate node has no client")
	}
	clients[0].SubID = uuid.NewString()
	var settings map[string]any
	_ = json.Unmarshal([]byte(ib.Settings), &settings)
	settings["clients"] = clients
	ib.Settings = mustJSON(settings)
	updated, _, err := s.inbounds.UpdateInbound(ib)
	if err != nil {
		return nil, err
	}
	return updated, nil
}
