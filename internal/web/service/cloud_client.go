package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
)

type CloudClient struct {
	BaseURL string
	HTTP    *http.Client
}

type CloudActivationClaimRequest struct {
	InviteCode   string `json:"invite_code"`
	DeviceID     string `json:"device_id"`
	Hostname     string `json:"hostname"`
	PublicIP     string `json:"public_ip"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	PanelVersion string `json:"panel_version"`
	AgentVersion string `json:"agent_version"`
}

type CloudActivationClaimResponse struct {
	ServerID        string `json:"server_id"`
	ActivationToken string `json:"activation_token"`
	Error           string `json:"error"`
}

type CloudActivationCheckRequest struct {
	ServerID     string `json:"server_id"`
	DeviceID     string `json:"device_id"`
	PanelVersion string `json:"panel_version"`
	PublicIP     string `json:"public_ip"`
}

type CloudActivationCheckResponse struct {
	Active *bool  `json:"active"`
	Reason string `json:"reason"`
	Error  string `json:"error"`
}

type CloudHeartbeatResponse struct {
	Active *bool  `json:"active"`
	Reason string `json:"reason"`
	Error  string `json:"error"`
}

func NewCloudClient(baseURL string) (*CloudClient, error) {
	normalized, err := normalizeCloudURL(baseURL)
	if err != nil {
		return nil, err
	}
	return &CloudClient{
		BaseURL: normalized,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func normalizeCloudURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", common.NewError("cloud_not_configured")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", common.NewError("invalid_cloud_url")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLocalCloudHost(u.Hostname())) {
		return "", common.NewError("cloud_url_must_be_https")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func isLocalCloudHost(host string) bool {
	host = strings.Trim(host, "[]")
	return strings.EqualFold(host, "localhost") || strings.EqualFold(host, "127.0.0.1") || strings.EqualFold(host, "::1") || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}

func (c *CloudClient) Claim(req CloudActivationClaimRequest) (*CloudActivationClaimResponse, error) {
	var out CloudActivationClaimResponse
	if err := c.doJSON(http.MethodPost, "/api/activation/claim", "", req, &out); err != nil {
		return nil, err
	}
	if out.Error != "" {
		return nil, common.NewError(out.Error)
	}
	if out.ServerID == "" || out.ActivationToken == "" {
		return nil, common.NewError("cloud_invalid_response")
	}
	return &out, nil
}

func (c *CloudClient) Check(token string, req CloudActivationCheckRequest) (*CloudActivationCheckResponse, error) {
	var out CloudActivationCheckResponse
	if err := c.doJSON(http.MethodPost, "/api/activation/check", token, req, &out); err != nil {
		return nil, err
	}
	if out.Error != "" {
		return nil, common.NewError(out.Error)
	}
	return &out, nil
}

func (c *CloudClient) Heartbeat(token string, req any) (*CloudHeartbeatResponse, error) {
	var out CloudHeartbeatResponse
	if err := c.doJSON(http.MethodPost, "/api/activation/heartbeat", token, req, &out); err != nil {
		return nil, err
	}
	if out.Error != "" {
		return nil, common.NewError(out.Error)
	}
	return &out, nil
}

func (c *CloudClient) doJSON(method, path, bearer string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequest(method, c.BaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if bearer != "" {
		httpReq.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return common.NewError("cloud_unreachable")
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, out)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		code := cloudErrorFromBody(respBody)
		if code == "" {
			code = fmt.Sprintf("cloud_http_%d", resp.StatusCode)
		}
		return common.NewError(code)
	}
	return nil
}

func cloudErrorFromBody(body []byte) string {
	var envelope struct {
		Error  string `json:"error"`
		Msg    string `json:"msg"`
		Reason string `json:"reason"`
		Obj    struct {
			Error string `json:"error"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ""
	}
	if envelope.Error != "" {
		return envelope.Error
	}
	if envelope.Obj.Error != "" {
		return envelope.Obj.Error
	}
	if envelope.Reason != "" {
		return envelope.Reason
	}
	return envelope.Msg
}
