// Package derp implements a small Tailscale-DERP-shaped relay: each client
// connects via WebSocket, announces its node key, and can send frames
// {dst_key, payload} that the server forwards to the matching connection.
//
// The server never decrypts the payload. WireGuard handles authenticity and
// confidentiality end-to-end; the relay is a dumb multiplexer.
package derp

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/apolzek/trynet/internal/protocol"
)

// Frame is the JSON envelope exchanged on the control channel; binary data
// frames use a compact length-prefixed format so we don't JSON-encode every
// WireGuard packet.
type Frame struct {
	Type string        `json:"type"` // "hello", "ping", "pong"
	Key  protocol.Key  `json:"key,omitempty"`
}

// Server is a stateful relay. Register returns an http.Handler.
type Server struct {
	log *log.Logger

	mu    sync.RWMutex
	conns map[protocol.Key]*conn
}

type conn struct {
	key     protocol.Key
	send    chan []byte
	ws      *wsConn
	closed  chan struct{}
	closeMu sync.Once
}

// New returns an empty relay server.
func New(lg *log.Logger) *Server {
	if lg == nil {
		lg = log.Default()
	}
	return &Server{log: lg, conns: map[protocol.Key]*conn{}}
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/relay", s.handleRelay)
	return mux
}

func (s *Server) handleRelay(w http.ResponseWriter, r *http.Request) {
	ws, err := wsUpgrade(w, r)
	if err != nil {
		s.log.Printf("upgrade: %v", err)
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return
	}
	defer ws.Close()

	// First frame must be a JSON hello announcing the client's node key.
	hello, err := ws.readText()
	if err != nil {
		s.log.Printf("hello read: %v", err)
		return
	}
	var f Frame
	if err := json.Unmarshal(hello, &f); err != nil || f.Type != "hello" {
		s.log.Printf("bad hello: %v", err)
		return
	}

	c := &conn{
		key:    f.Key,
		send:   make(chan []byte, 64),
		ws:     ws,
		closed: make(chan struct{}),
	}
	s.register(c)
	defer s.unregister(c)

	s.log.Printf("peer up: %s", c.key)

	// Writer goroutine: serialise outbound frames.
	go func() {
		for {
			select {
			case <-c.closed:
				return
			case data := <-c.send:
				if err := c.ws.writeBinary(data); err != nil {
					c.close()
					return
				}
			case <-time.After(30 * time.Second):
				_ = c.ws.writeText([]byte(`{"type":"ping"}`))
			}
		}
	}()

	// Reader loop.
	for {
		op, data, err := c.ws.readAny()
		if err != nil {
			if err != io.EOF {
				s.log.Printf("read: %v", err)
			}
			return
		}
		switch op {
		case opBinary:
			// Binary framing: 32 bytes dst key | payload
			if len(data) < 32 {
				continue
			}
			var dst protocol.Key
			copy(dst[:], data[:32])
			payload := data[32:]
			s.forward(c.key, dst, payload)
		case opText:
			var fr Frame
			if err := json.Unmarshal(data, &fr); err != nil {
				continue
			}
			switch fr.Type {
			case "ping":
				_ = c.ws.writeText([]byte(`{"type":"pong"}`))
			}
		case opClose:
			return
		}
	}
}

func (s *Server) register(c *conn) {
	s.mu.Lock()
	if prev, ok := s.conns[c.key]; ok {
		prev.close() // kick old session
	}
	s.conns[c.key] = c
	s.mu.Unlock()
}

func (s *Server) unregister(c *conn) {
	s.mu.Lock()
	if cur, ok := s.conns[c.key]; ok && cur == c {
		delete(s.conns, c.key)
	}
	s.mu.Unlock()
	c.close()
	s.log.Printf("peer down: %s", c.key)
}

func (s *Server) forward(srcKey, dstKey protocol.Key, payload []byte) {
	s.mu.RLock()
	dst := s.conns[dstKey]
	s.mu.RUnlock()
	if dst == nil {
		return
	}
	// Prepend source key so the receiver can authenticate / correlate.
	buf := make([]byte, 32+len(payload))
	copy(buf[:32], srcKey[:])
	copy(buf[32:], payload)
	select {
	case dst.send <- buf:
	default:
		// receiver is slow; drop. Relay is best-effort.
	}
}

func (c *conn) close() {
	c.closeMu.Do(func() {
		close(c.closed)
		_ = c.ws.Close()
	})
}

// Run starts listening on addr.
func (s *Server) Run(ctx context.Context, addr, certFile, keyFile string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}
	errCh := make(chan error, 1)
	go func() {
		if certFile != "" && keyFile != "" {
			s.log.Printf("derp: listening wss on %s", addr)
			errCh <- srv.ListenAndServeTLS(certFile, keyFile)
		} else {
			s.log.Printf("derp: listening ws on %s", addr)
			errCh <- srv.ListenAndServe()
		}
	}()
	select {
	case <-ctx.Done():
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

// --- small utility used by the client to build a binary frame ------------

// EncodeFrame builds the binary relay frame: 32-byte dest key || payload.
func EncodeFrame(dst protocol.Key, payload []byte) []byte {
	buf := make([]byte, 32+len(payload))
	copy(buf[:32], dst[:])
	copy(buf[32:], payload)
	return buf
}

// DecodeFrame splits an inbound binary frame into (srcKey, payload).
func DecodeFrame(b []byte) (src protocol.Key, payload []byte, err error) {
	if len(b) < 32 {
		return src, nil, fmt.Errorf("short frame %d", len(b))
	}
	copy(src[:], b[:32])
	return src, b[32:], nil
}

// varint-style length prefix unused at the moment but exposed for future
// multi-stream demuxing.
func writeU16(w io.Writer, n uint16) error {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], n)
	_, err := w.Write(b[:])
	return err
}
