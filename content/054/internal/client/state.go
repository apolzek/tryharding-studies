package client

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/apolzek/trynet/internal/protocol"
)

// Persisted client-side state. Stored under /var/lib/trynet/client.json.
type State struct {
	PrivateKey protocol.Key `json:"private_key"`
	PublicKey  protocol.Key `json:"public_key"`
	NodeID     string       `json:"node_id,omitempty"`
	TailnetIP  string       `json:"tailnet_ip,omitempty"`
	AuthKey    string       `json:"auth_key,omitempty"` // remembered for re-registration
	ControlURL string       `json:"control_url,omitempty"`
	RelayURL   string       `json:"relay_url,omitempty"`
	Insecure   bool         `json:"insecure,omitempty"`
	ListenPort int          `json:"listen_port,omitempty"`
}

// Config is read from /etc/trynet/config.json at startup.
type Config struct {
	ControlURL string `json:"control_url"`
	RelayURL   string `json:"relay_url"`
	Insecure   bool   `json:"insecure"`
	ListenPort int    `json:"listen_port"`
}

var stateMu sync.Mutex

// LoadState reads the client's persisted state, returning os.ErrNotExist if
// it's missing.
func LoadState(path string) (*State, error) {
	stateMu.Lock()
	defer stateMu.Unlock()
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SaveState persists the state atomically.
func SaveState(path string, s *State) error {
	stateMu.Lock()
	defer stateMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadConfig reads /etc/trynet/config.json.
func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.ControlURL == "" {
		return nil, errors.New("config: control_url required")
	}
	if c.ListenPort == 0 {
		c.ListenPort = 41641
	}
	return &c, nil
}
