package netbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
)

// Client is the base NetBox API client.
// All NetBox operations are methods on this struct.
type Client struct {
	baseURL string
	token   string
	http    *retryablehttp.Client
	log     zerolog.Logger
}

// NewClient creates a new NetBox API client with retry support.
func NewClient(baseURL, token string, log zerolog.Logger) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil // silence retryablehttp's own logger

	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    retryClient,
		log:     log,
	}
}

// get performs a GET request and decodes the response into result.
func (c *Client) get(path string, result interface{}) error {
	url := c.baseURL + path

	req, err := retryablehttp.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	return c.decodeResponse(resp, result)
}

// post performs a POST request with a JSON body and decodes the response.
func (c *Client) post(path string, body interface{}, result interface{}) error {
	return c.doWithBody(http.MethodPost, path, body, result)
}

// patch performs a PATCH request with a JSON body and decodes the response.
func (c *Client) patch(path string, body interface{}, result interface{}) error {
	return c.doWithBody(http.MethodPatch, path, body, result)
}

// doWithBody is shared logic for POST and PATCH.
func (c *Client) doWithBody(method, path string, body interface{}, result interface{}) error {
	url := c.baseURL + path

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling body: %w", err)
	}

	req, err := retryablehttp.NewRequest(method, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	return c.decodeResponse(resp, result)
}

// setHeaders sets required headers on every request.
func (c *Client) setHeaders(req *retryablehttp.Request) {
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// decodeResponse checks status code and decodes JSON body.
func (c *Client) decodeResponse(resp *http.Response, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	// Treat 4xx and 5xx as errors
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if result == nil {
		return nil
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// listResponse is the common wrapper NetBox uses for list endpoints.
type listResponse struct {
	Count   int               `json:"count"`
	Results []json.RawMessage `json:"results"`
}

// idResponse is used when we only need the ID from a response.
type idResponse struct {
	ID int `json:"id"`
}

// delete sends a DELETE request to NetBox.
func (c *Client) delete(path string) error {
	req, err := retryablehttp.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 204 No Content is success for DELETE
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete failed: status %d", resp.StatusCode)
	}

	return nil
}
