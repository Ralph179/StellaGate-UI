package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
)

const StellaDeviceSecretPath = "/etc/x-ui/stellagate-device-secret"

// StellaDeviceID returns a stable, hashed local device identifier. Raw
// machine-id values are never uploaded; a local random secret is mixed in and
// kept with mode 0600 so cloning the same image does not create the same ID.
func StellaDeviceID() (string, error) {
	secret, err := readOrCreateStellaDeviceSecret()
	if err != nil {
		return "", err
	}
	machineID := readFirstExistingText("/etc/machine-id", "/var/lib/dbus/machine-id")
	hostname, _ := os.Hostname()
	sum := sha256.Sum256([]byte(strings.Join([]string{
		"stellagate-ui",
		strings.TrimSpace(machineID),
		strings.TrimSpace(hostname),
		strings.TrimSpace(secret),
	}, "\n")))
	return "sgd_" + hex.EncodeToString(sum[:]), nil
}

func readOrCreateStellaDeviceSecret() (string, error) {
	if b, err := os.ReadFile(StellaDeviceSecretPath); err == nil && strings.TrimSpace(string(b)) != "" {
		_ = os.Chmod(StellaDeviceSecretPath, 0600)
		return strings.TrimSpace(string(b)), nil
	}
	if err := os.MkdirAll("/etc/x-ui", 0700); err != nil {
		return "", common.NewError("create StellaGate config dir: ", err)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", common.NewError("generate StellaGate device secret: ", err)
	}
	secret := hex.EncodeToString(buf)
	if err := os.WriteFile(StellaDeviceSecretPath, []byte(secret+"\n"), 0600); err != nil {
		return "", common.NewError("write StellaGate device secret: ", err)
	}
	_ = os.Chmod(StellaDeviceSecretPath, 0600)
	return secret, nil
}

func readFirstExistingText(paths ...string) string {
	for _, path := range paths {
		if b, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(b)) != "" {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}
