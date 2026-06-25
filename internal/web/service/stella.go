package service

// StellaService is a deliberately small product layer over the existing
// inbound/client/Xray services.  It owns no second configuration format: the
// generated templates are ordinary 3x-ui inbounds and remain editable from
// the advanced UI.

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	"gorm.io/gorm"
)

const stellaTagPrefix = "stellagate-"

const (
	stellaHysteriaCertDir  = "/etc/x-ui/stellagate"
	stellaHysteriaCertFile = stellaHysteriaCertDir + "/hysteria.crt"
	stellaHysteriaKeyFile  = stellaHysteriaCertDir + "/hysteria.key"
)

var stellaHysteriaCertMu sync.Mutex

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

// ensureHysteriaCertificate creates one stable, local certificate for the
// single-VPS Hysteria2 template. A fresh certificate is only generated when
// the stored pair is absent or unreadable; its SHA-256 pin is sent in the
// subscription so clients can validate the self-managed node without asking
// ordinary users to provision a domain and ACME certificate first.
func ensureHysteriaCertificate() (certFile, keyFile, pin string, err error) {
	stellaHysteriaCertMu.Lock()
	defer stellaHysteriaCertMu.Unlock()

	loadPin := func() (string, error) {
		pair, err := tls.LoadX509KeyPair(stellaHysteriaCertFile, stellaHysteriaKeyFile)
		if err != nil || len(pair.Certificate) == 0 {
			return "", err
		}
		sum := sha256.Sum256(pair.Certificate[0])
		return hex.EncodeToString(sum[:]), nil
	}
	if pin, err = loadPin(); err == nil {
		return stellaHysteriaCertFile, stellaHysteriaKeyFile, pin, nil
	}

	if err = os.MkdirAll(stellaHysteriaCertDir, 0700); err != nil {
		return "", "", "", common.NewError("could not create Hysteria certificate directory: ", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", "", common.NewError("could not generate Hysteria private key: ", err)
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return "", "", "", common.NewError("could not generate Hysteria certificate serial: ", err)
	}
	now := time.Now()
	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "stellagate"},
		DNSNames:              []string{"stellagate"},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.AddDate(5, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", "", "", common.NewError("could not create Hysteria certificate: ", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", "", common.NewError("could not encode Hysteria private key: ", err)
	}
	if err = os.WriteFile(stellaHysteriaCertFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
		return "", "", "", common.NewError("could not write Hysteria certificate: ", err)
	}
	if err = os.WriteFile(stellaHysteriaKeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		_ = os.Remove(stellaHysteriaCertFile)
		return "", "", "", common.NewError("could not write Hysteria private key: ", err)
	}
	_ = os.Chmod(stellaHysteriaCertFile, 0600)
	_ = os.Chmod(stellaHysteriaKeyFile, 0600)
	sum := sha256.Sum256(der)
	pin = hex.EncodeToString(sum[:])
	return filepath.Clean(stellaHysteriaCertFile), filepath.Clean(stellaHysteriaKeyFile), pin, nil
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
			"privateKey": keys["privateKey"], "shortIds": []string{shortID},
			// `settings` is panel-only client metadata. Xray strips it from the
			// server config, while the subscription generator reads it to emit
			// the required pbk/fp Reality parameters for mobile clients.
			"settings": map[string]any{"publicKey": keys["publicKey"], "fingerprint": "chrome"},
		}})
	case "hysteria2":
		// Prefer UDP/443 for the simplified product layer. Random high UDP
		// ports are often blocked by cloud firewalls or mobile networks, which
		// looks like a client-side timeout even when Xray itself is running.
		// If 443/udp is unavailable locally, fall back to a random available
		// UDP port and let the API surface that clear port to the user.
		port, err := freePort("udp", 443)
		if err != nil {
			return nil, err
		}
		certFile, keyFile, pin, err := ensureHysteriaCertificate()
		if err != nil {
			return nil, err
		}
		client.ID, client.Flow, client.Auth = "", "", uuid.NewString()
		ib.Port, ib.Protocol, ib.Remark, ib.Tag = port, model.Hysteria, "StellaGate · Hysteria2", stellaTagPrefix+"hysteria2"
		ib.Settings = mustJSON(map[string]any{"version": 2, "clients": []model.Client{client}})
		ib.StreamSettings = mustJSON(map[string]any{"network": "hysteria", "security": "tls", "tlsSettings": map[string]any{
			"serverName": "stellagate", "alpn": []string{"h3"},
			"certificates": []map[string]any{{"certificateFile": certFile, "keyFile": keyFile, "usage": "encipherment"}},
			// Client-only metadata. The Xray config builder strips `settings`;
			// clients receive the certificate pin and self-signed compatibility
			// flag in their generated Hysteria2 URI.
			"settings": map[string]any{"pinnedPeerCertSha256": []string{pin}, "allowInsecure": true},
		}})
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

func (s *StellaService) RandomizePort(userID int) (*model.Inbound, error) {
	ib, err := s.StellaInbound(userID)
	if err != nil {
		return nil, err
	}
	if ib == nil {
		return nil, common.NewError("no StellaGate node exists; choose a protocol first")
	}
	network := "tcp"
	if ib.Protocol == model.Hysteria {
		network = "udp"
	}
	port, err := freePort(network, 0)
	if err != nil {
		return nil, err
	}
	ib.Port = port
	updated, _, err := s.inbounds.UpdateInbound(ib)
	if err != nil {
		return nil, err
	}
	if err := s.server.RestartXrayService(); err != nil {
		return nil, common.NewError("port changed but proxy service could not start: ", err)
	}
	return updated, nil
}

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
