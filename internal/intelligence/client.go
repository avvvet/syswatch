package intelligence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client talks to the Central Intelligence API (syswatch-api).
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New creates a new intelligence client.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{},
	}
}

// ResolveRequest is sent to the Central Intelligence API.
type ResolveRequest struct {
	Category     string `json:"category"`
	Manufacturer string `json:"manufacturer"`
	RawValue     string `json:"raw_value"`
}

// ResolveResponse is returned by the Central Intelligence API.
type ResolveResponse struct {
	NetBoxID      int    `json:"netbox_id"`
	CanonicalName string `json:"canonical_name"`
	Confidence    string `json:"confidence"`
	Action        string `json:"action"`
}

// ResolveManufacturer asks the API to resolve a manufacturer name.
func (c *Client) ResolveManufacturer(rawValue string) (*ResolveResponse, error) {
	return c.resolve(ResolveRequest{
		Category: "manufacturer",
		RawValue: rawValue,
	})
}

// ResolveDeviceType asks the API to resolve a device type.
func (c *Client) ResolveDeviceType(rawValue, manufacturer string) (*ResolveResponse, error) {
	return c.resolve(ResolveRequest{
		Category:     "device_type",
		RawValue:     rawValue,
		Manufacturer: manufacturer,
	})
}

// ResolvePlatform asks the API to resolve a platform (OS).
func (c *Client) ResolvePlatform(rawValue string) (*ResolveResponse, error) {
	return c.resolve(ResolveRequest{
		Category: "platform",
		RawValue: rawValue,
	})
}

// Ping checks the Central Intelligence API is reachable.
func (c *Client) Ping() error {
	req, err := http.NewRequest("GET", c.baseURL+"/status", nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("Central Intelligence API unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Central Intelligence API returned status %d", resp.StatusCode)
	}

	return nil
}

// resolve sends a resolve request to the Central Intelligence API.
func (c *Client) resolve(req ResolveRequest) (*ResolveResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/resolve", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Syswatch-Key", c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("resolve request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Central Intelligence API error %d: %s", resp.StatusCode, string(raw))
	}

	var result ResolveResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parsing resolve response: %w", err)
	}

	return &result, nil
}
