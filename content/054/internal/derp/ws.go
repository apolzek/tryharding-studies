package derp

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// Minimal RFC 6455 server implementation. We only need a tiny subset:
// - server handshake
// - read/write text + binary frames
// - read close / ping
// No extensions, no compression, no fragmentation (single-frame messages only).

type opcode byte

const (
	opContinuation opcode = 0x0
	opText         opcode = 0x1
	opBinary       opcode = 0x2
	opClose        opcode = 0x8
	opPing         opcode = 0x9
	opPong         opcode = 0xA
)

type wsConn struct {
	c  net.Conn
	br *bufio.Reader
}

const wsMagic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func wsUpgrade(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
		return nil, errors.New("not a websocket upgrade")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, errors.New("missing Sec-WebSocket-Key")
	}
	h, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("hijack not supported")
	}
	conn, rw, err := h.Hijack()
	if err != nil {
		return nil, err
	}
	sum := sha1.Sum([]byte(key + wsMagic))
	accept := base64.StdEncoding.EncodeToString(sum[:])
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := rw.WriteString(resp); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &wsConn{c: conn, br: rw.Reader}, nil
}

// Close closes the underlying TCP conn.
func (w *wsConn) Close() error { return w.c.Close() }

// writeFrame sends a single, unmasked frame (server→client: MUST be unmasked).
func (w *wsConn) writeFrame(op opcode, data []byte) error {
	var hdr [14]byte
	hdr[0] = 0x80 | byte(op) // FIN=1
	n := 2
	l := len(data)
	switch {
	case l <= 125:
		hdr[1] = byte(l)
	case l <= 0xFFFF:
		hdr[1] = 126
		binary.BigEndian.PutUint16(hdr[2:4], uint16(l))
		n = 4
	default:
		hdr[1] = 127
		binary.BigEndian.PutUint64(hdr[2:10], uint64(l))
		n = 10
	}
	if _, err := w.c.Write(hdr[:n]); err != nil {
		return err
	}
	_, err := w.c.Write(data)
	return err
}

func (w *wsConn) writeText(data []byte) error   { return w.writeFrame(opText, data) }
func (w *wsConn) writeBinary(data []byte) error { return w.writeFrame(opBinary, data) }

// readAny reads one frame (defragments continuation if needed).
func (w *wsConn) readAny() (opcode, []byte, error) {
	var firstOp opcode
	var payload []byte
	for {
		op, data, fin, err := w.readOne()
		if err != nil {
			return 0, nil, err
		}
		if op == opPing {
			_ = w.writeFrame(opPong, data)
			continue
		}
		if op == opClose {
			return opClose, data, nil
		}
		if len(payload) == 0 {
			firstOp = op
		}
		payload = append(payload, data...)
		if fin {
			return firstOp, payload, nil
		}
	}
}

// readText is a convenience for JSON control frames.
func (w *wsConn) readText() ([]byte, error) {
	op, data, err := w.readAny()
	if err != nil {
		return nil, err
	}
	if op != opText {
		return nil, fmt.Errorf("expected text, got op %d", op)
	}
	return data, nil
}

func (w *wsConn) readOne() (opcode, []byte, bool, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(w.br, hdr[:]); err != nil {
		return 0, nil, false, err
	}
	fin := hdr[0]&0x80 != 0
	op := opcode(hdr[0] & 0x0F)
	masked := hdr[1]&0x80 != 0
	plen := int(hdr[1] & 0x7F)

	switch plen {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(w.br, ext[:]); err != nil {
			return 0, nil, false, err
		}
		plen = int(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(w.br, ext[:]); err != nil {
			return 0, nil, false, err
		}
		plen = int(binary.BigEndian.Uint64(ext[:]))
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(w.br, maskKey[:]); err != nil {
			return 0, nil, false, err
		}
	}
	data := make([]byte, plen)
	if _, err := io.ReadFull(w.br, data); err != nil {
		return 0, nil, false, err
	}
	if masked {
		for i := range data {
			data[i] ^= maskKey[i%4]
		}
	}
	return op, data, fin, nil
}
