package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server"
	"github.com/open-telemetry/opamp-go/server/types"
)

type Agent struct {
	InstanceUID     string               `json:"instance_uid"`
	Description     map[string]string    `json:"description"`
	Status          string               `json:"status"`
	Health          string               `json:"health"`
	EffectiveConfig string               `json:"effective_config"`
	LastSeen        time.Time            `json:"last_seen"`
	RemoteConfig    []byte               `json:"remote_config"`
	RemoteHash      []byte               `json:"remote_hash"`
	LastConfigPush  time.Time            `json:"last_config_push"`
	History         []ConfigHistoryEntry `json:"history"`

	conn types.Connection `json:"-"`
}

type ConfigHistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Config    string    `json:"config"`
	Status    string    `json:"status"`
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
}

var (
	agentsMu sync.Mutex
	agents   = map[string]*Agent{}
	dataDir  = "/data"
	opampSrv server.OpAMPServer
)

const rateLimitWindow = 30 * time.Second

func main() {
	if d := os.Getenv("DATA_DIR"); d != "" {
		dataDir = d
	}
	_ = os.MkdirAll(dataDir, 0o755)
	loadState()

	opampSrv = server.New(stdLogger{})
	err := opampSrv.Start(server.StartSettings{
		Settings: server.Settings{
			Callbacks: types.Callbacks{
				OnConnecting: onConnecting,
			},
		},
		ListenEndpoint: "0.0.0.0:4320",
	})
	if err != nil {
		log.Fatalf("opamp server start: %v", err)
	}
	defer opampSrv.Stop(context.Background())
	log.Println("OpAMP listening on :4320/v1/opamp")

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("/web")))
	mux.HandleFunc("/api/agents", apiListAgents)
	mux.HandleFunc("/api/agent", apiAgent)
	mux.HandleFunc("/api/validate", apiValidate)
	mux.HandleFunc("/api/rollback", apiRollback)
	log.Println("UI/API listening on :4321")
	log.Fatal(http.ListenAndServe(":4321", mux))
}

type stdLogger struct{}

func (stdLogger) Debugf(_ context.Context, format string, v ...any) {}
func (stdLogger) Errorf(_ context.Context, format string, v ...any) {
	log.Printf("opamp-err: "+format, v...)
}

func onConnecting(r *http.Request) types.ConnectionResponse {
	return types.ConnectionResponse{
		Accept: true,
		ConnectionCallbacks: types.ConnectionCallbacks{
			OnConnected:       onConnected,
			OnMessage:         onMessage,
			OnConnectionClose: onConnectionClose,
		},
	}
}

func onConnected(_ context.Context, _ types.Connection) {
	log.Println("agent connected (pre-handshake)")
}

func onConnectionClose(conn types.Connection) {
	agentsMu.Lock()
	defer agentsMu.Unlock()
	for _, a := range agents {
		if a.conn == conn {
			a.conn = nil
			a.Status = "disconnected"
			log.Printf("agent %s disconnected", short(a.InstanceUID))
		}
	}
}

func onMessage(_ context.Context, conn types.Connection, msg *protobufs.AgentToServer) *protobufs.ServerToAgent {
	uid := hex.EncodeToString(msg.InstanceUid)

	agentsMu.Lock()
	defer agentsMu.Unlock()

	a, ok := agents[uid]
	if !ok {
		a = &Agent{
			InstanceUID: uid,
			Description: map[string]string{},
			History:     []ConfigHistoryEntry{},
		}
		agents[uid] = a
		log.Printf("agent %s registered", short(uid))
	}
	a.conn = conn
	a.LastSeen = time.Now()
	a.Status = "connected"

	if msg.AgentDescription != nil {
		desc := map[string]string{}
		for k, v := range flattenAttrs(msg.AgentDescription.IdentifyingAttributes) {
			desc[k] = v
		}
		for k, v := range flattenAttrs(msg.AgentDescription.NonIdentifyingAttributes) {
			desc[k] = v
		}
		a.Description = desc
	}
	if msg.EffectiveConfig != nil && msg.EffectiveConfig.ConfigMap != nil {
		for _, cf := range msg.EffectiveConfig.ConfigMap.ConfigMap {
			a.EffectiveConfig = string(cf.Body)
			break
		}
	}
	if msg.Health != nil {
		if msg.Health.Healthy {
			a.Health = "healthy"
		} else {
			a.Health = "unhealthy: " + msg.Health.LastError
		}
	}
	if msg.RemoteConfigStatus != nil {
		switch msg.RemoteConfigStatus.Status {
		case protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED:
			markHistory(a, msg.RemoteConfigStatus.LastRemoteConfigHash, "applied")
		case protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED:
			markHistory(a, msg.RemoteConfigStatus.LastRemoteConfigHash, "failed: "+msg.RemoteConfigStatus.ErrorMessage)
			if prev := previousApplied(a); prev != nil {
				log.Printf("agent %s: auto-rollback to hash %s", short(uid), prev.Hash[:12])
				a.RemoteConfig = []byte(prev.Config)
				a.RemoteHash = mustDecodeHex(prev.Hash)
			}
		case protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLYING:
			markHistory(a, msg.RemoteConfigStatus.LastRemoteConfigHash, "applying")
		}
	}

	if len(a.RemoteConfig) == 0 {
		if svc := a.Description["service.name"]; svc != "" {
			path := filepath.Join(dataDir, "defaults", svc+".yaml")
			if b, err := os.ReadFile(path); err == nil {
				a.RemoteConfig = b
				a.RemoteHash = hashBytes(b)
				a.History = append(a.History, ConfigHistoryEntry{
					Timestamp: time.Now(),
					Config:    string(b),
					Status:    "pending",
					Hash:      hex.EncodeToString(a.RemoteHash),
					Author:    "bootstrap",
				})
				log.Printf("agent %s: bootstrapped from defaults/%s.yaml", short(uid), svc)
			}
		}
	}

	resp := &protobufs.ServerToAgent{InstanceUid: msg.InstanceUid}
	if len(a.RemoteConfig) > 0 {
		resp.RemoteConfig = buildRemoteConfig(a.RemoteConfig, a.RemoteHash)
	}
	saveStateLocked()
	return resp
}

func buildRemoteConfig(body, hash []byte) *protobufs.AgentRemoteConfig {
	return &protobufs.AgentRemoteConfig{
		Config: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				"": {Body: body, ContentType: "text/yaml"},
			},
		},
		ConfigHash: hash,
	}
}

func flattenAttrs(kvs []*protobufs.KeyValue) map[string]string {
	out := map[string]string{}
	for _, kv := range kvs {
		if kv == nil || kv.Value == nil {
			continue
		}
		switch v := kv.Value.Value.(type) {
		case *protobufs.AnyValue_StringValue:
			out[kv.Key] = v.StringValue
		case *protobufs.AnyValue_IntValue:
			out[kv.Key] = fmt.Sprintf("%d", v.IntValue)
		case *protobufs.AnyValue_BoolValue:
			out[kv.Key] = fmt.Sprintf("%t", v.BoolValue)
		case *protobufs.AnyValue_DoubleValue:
			out[kv.Key] = fmt.Sprintf("%g", v.DoubleValue)
		}
	}
	return out
}

func markHistory(a *Agent, hash []byte, status string) {
	if len(hash) == 0 {
		return
	}
	h := hex.EncodeToString(hash)
	for i := range a.History {
		if a.History[i].Hash == h {
			a.History[i].Status = status
			a.History[i].Timestamp = time.Now()
			return
		}
	}
}

func previousApplied(a *Agent) *ConfigHistoryEntry {
	for i := len(a.History) - 2; i >= 0; i-- {
		if a.History[i].Status == "applied" {
			return &a.History[i]
		}
	}
	return nil
}

func hashBytes(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

func mustDecodeHex(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// ---------- HTTP API ----------

func apiListAgents(w http.ResponseWriter, _ *http.Request) {
	agentsMu.Lock()
	defer agentsMu.Unlock()
	list := make([]*Agent, 0, len(agents))
	for _, a := range agents {
		list = append(list, a)
	}
	writeJSON(w, list)
}

func apiAgent(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	agentsMu.Lock()
	defer agentsMu.Unlock()
	a, ok := agents[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, a)
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if time.Since(a.LastConfigPush) < rateLimitWindow {
			http.Error(w, fmt.Sprintf("rate limit: wait %.0fs", (rateLimitWindow-time.Since(a.LastConfigPush)).Seconds()), http.StatusTooManyRequests)
			return
		}
		if err := validateConfig(body); err != nil {
			http.Error(w, "guardrails: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}
		a.RemoteConfig = body
		a.RemoteHash = hashBytes(body)
		a.LastConfigPush = time.Now()
		a.History = append(a.History, ConfigHistoryEntry{
			Timestamp: time.Now(),
			Config:    string(body),
			Status:    "pending",
			Hash:      hex.EncodeToString(a.RemoteHash),
			Author:    "ui",
		})
		if len(a.History) > 20 {
			a.History = a.History[len(a.History)-20:]
		}
		pushToAgent(a)
		saveStateLocked()
		log.Printf("agent %s: config pushed (hash=%s)", short(a.InstanceUID), hex.EncodeToString(a.RemoteHash)[:12])
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiValidate(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	if err := validateConfig(body); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func apiRollback(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	hash := r.URL.Query().Get("hash")
	agentsMu.Lock()
	defer agentsMu.Unlock()
	a, ok := agents[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	for _, h := range a.History {
		if h.Hash == hash {
			a.RemoteConfig = []byte(h.Config)
			a.RemoteHash = mustDecodeHex(hash)
			a.History = append(a.History, ConfigHistoryEntry{
				Timestamp: time.Now(),
				Config:    h.Config,
				Status:    "pending",
				Hash:      hash,
				Author:    "rollback",
			})
			pushToAgent(a)
			saveStateLocked()
			log.Printf("agent %s: rollback to %s", short(a.InstanceUID), hash[:12])
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}
	http.Error(w, "hash not in history", http.StatusNotFound)
}

func pushToAgent(a *Agent) {
	if a.conn == nil {
		return
	}
	_ = a.conn.Send(context.Background(), &protobufs.ServerToAgent{
		InstanceUid:  mustDecodeHex(a.InstanceUID),
		RemoteConfig: buildRemoteConfig(a.RemoteConfig, a.RemoteHash),
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// ---------- state persistence ----------

func loadState() {
	f, err := os.Open(filepath.Join(dataDir, "state.json"))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("load state: %v", err)
		}
		return
	}
	defer f.Close()
	saved := map[string]*Agent{}
	if err := json.NewDecoder(f).Decode(&saved); err != nil {
		log.Printf("load state decode: %v", err)
		return
	}
	for k, v := range saved {
		v.Status = "disconnected"
		agents[k] = v
	}
	log.Printf("loaded %d agents from state", len(agents))
}

func saveStateLocked() {
	tmp := filepath.Join(dataDir, "state.json.tmp")
	f, err := os.Create(tmp)
	if err != nil {
		return
	}
	if err := json.NewEncoder(f).Encode(agents); err != nil {
		f.Close()
		os.Remove(tmp)
		return
	}
	f.Close()
	_ = os.Rename(tmp, filepath.Join(dataDir, "state.json"))
}
