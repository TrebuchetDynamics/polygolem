// Package geoblock checks Polymarket's geoblock endpoint, which reports
// whether the caller's IP is blocked from trading and where it appears to be.
package geoblock

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultURL is Polymarket's public geoblock endpoint.
const DefaultURL = "https://polymarket.com/api/geoblock"

// Client queries the geoblock endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Result is the geoblock verdict for the calling IP.
type Result struct {
	Blocked bool   `json:"blocked"`
	IP      string `json:"ip"`
	Country string `json:"country"`
	Region  string `json:"region"`
}

// New returns a geoblock client. An empty baseURL uses DefaultURL; a nil
// httpClient uses a 5-second-timeout default.
func New(baseURL string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = DefaultURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

// Check queries the geoblock endpoint and returns the verdict.
func (c *Client) Check(ctx context.Context) (Result, error) {
	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Result{}, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Result{}, fmt.Errorf("geoblock status %d", resp.StatusCode)
	}

	var out Result
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Result{}, err
	}
	return out, nil
}
