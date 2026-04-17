package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

func (h *Hub) add(c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = true
}

func (h *Hub) remove(c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

func (h *Hub) broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			c.Close()
			delete(h.clients, c)
		}
	}
}

var hub = &Hub{clients: make(map[*websocket.Conn]bool)}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()
	hub.add(conn)
	defer hub.remove(conn)
	log.Printf("[go-ws] client connected (total=%d)", len(hub.clients))

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Printf("[go-ws] received: %s", data)
		hub.broadcast(data)
		_ = mt
	}
}

func pinger() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		hub.mu.Lock()
		for c := range hub.clients {
			c.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		}
		hub.mu.Unlock()
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func main() {
	go pinger()
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/health", health)
	addr := ":8001"
	log.Printf("[go-ws] listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
