package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/apolzek/trynet/internal/protocol"
)

// controlClient is a thin HTTP client for the coordination server.
type controlClient struct {
	baseURL string
	http    *http.Client
}

func newControlClient(baseURL string, insecure bool) *controlClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &controlClient{
		baseURL: baseURL,
		http:    &http.Client{Transport: tr, Timeout: 60 * time.Second},
	}
}

// Register enrols the node and returns the assigned tailnet IP.
func (c *controlClient) Register(ctx context.Context, req *protocol.RegisterRequest) (*protocol.RegisterResponse, error) {
	var resp protocol.RegisterResponse
	if err := c.do(ctx, "POST", "/machine/register", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PollNetMap long-polls for a netmap update. Returns the current map (may be
// the same version if the poll timed out with a keep-alive).
func (c *controlClient) PollNetMap(ctx context.Context, req *protocol.PollRequest) (*protocol.NetMap, error) {
	var nm protocol.NetMap
	// Use a longer timeout than the server's 30s keepalive.
	pollCtx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()
	if err := c.do(pollCtx, "POST", "/machine/map", req, &nm); err != nil {
		return nil, err
	}
	return &nm, nil
}

// ReportEndpoints pushes a fresh list of locally-observed endpoints.
func (c *controlClient) ReportEndpoints(ctx context.Context, r *protocol.EndpointsReport) error {
	return c.do(ctx, "POST", "/machine/endpoints", r, nil)
}

// Logout removes the node from the tailnet.
func (c *controlClient) Logout(ctx context.Context, nk protocol.Key) error {
	body := struct {
		NodeKey protocol.Key `json:"node_key"`
	}{nk}
	return c.do(ctx, "POST", "/machine/logout", body, nil)
}

func (c *controlClient) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, bytes.TrimSpace(msg))
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}

// errBackendNotReady is returned by the CLI socket when the daemon hasn't
// completed its first registration yet.
var errBackendNotReady = errors.New("backend not ready")
