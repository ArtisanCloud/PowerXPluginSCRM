package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ABACInput struct {
	Subject  TenantContext  `json:"subject"`
	Resource string         `json:"resource"`
	Action   string         `json:"action"`
	Attrs    map[string]any `json:"attrs,omitempty"`
}

type ABACDecision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

type ABACClient interface {
	Check(ctx context.Context, in ABACInput) (ABACDecision, error)
}

type HTTPABACClient struct {
	Endpoint string
	Client   *http.Client
	// 可选：鉴权头/超时等
}

func NewHTTPABACClient(endpoint string) *HTTPABACClient {
	return &HTTPABACClient{
		Endpoint: endpoint,
		Client:   &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *HTTPABACClient) Check(ctx context.Context, in ABACInput) (ABACDecision, error) {
	b, _ := json.Marshal(in)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return ABACDecision{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return ABACDecision{}, fmt.Errorf("pdp %d: %s", resp.StatusCode, string(rb))
	}
	var out ABACDecision
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ABACDecision{}, err
	}
	return out, nil
}
