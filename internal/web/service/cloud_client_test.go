package service

import "testing"

func TestCloudErrorFromBodyUsesReason(t *testing.T) {
	got := cloudErrorFromBody([]byte(`{"active":false,"reason":"invalid_token"}`))
	if got != "invalid_token" {
		t.Fatalf("cloudErrorFromBody reason = %q, want invalid_token", got)
	}
}

func TestActivationInvalidationReason(t *testing.T) {
	tests := map[string]string{
		"revoked":           "revoked",
		"expired":           "expired",
		"invalid_token":     "invalid_token",
		"device_mismatch":   "device_mismatch",
		"cloud_http_401":    "invalid_token",
		"cloud_http_404":    "invalid_token",
		"cloud_unreachable": "",
	}
	for code, want := range tests {
		if got := activationInvalidationReason(code); got != want {
			t.Fatalf("activationInvalidationReason(%q) = %q, want %q", code, got, want)
		}
	}
}
