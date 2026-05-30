package handlers

import (
	"bufio"
	"context"
	"log"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/oroitz-lago-ramos/minecraft-panel/internal/minecraft"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "https://mc.oroitzlagoramos.com"
	},
}

type WSHandler struct {
	mc          *minecraft.Server
	agentClient *http.Client
}

func NewWSHandler(mc *minecraft.Server) *WSHandler {
	agentClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/tmp/mc-agent.sock")
			},
		},
	}
	return &WSHandler{mc: mc, agentClient: agentClient}
}

func (h *WSHandler) Console(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	resp, err := h.agentClient.Get("http://agent/logs/stream")
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Erreur agent: "+err.Error()))
		return
	}
	defer resp.Body.Close()

	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(scanner.Text())); err != nil {
				break
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		response, err := h.mc.SendCommand(string(msg))
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("Erreur: "+err.Error()))
			continue
		}
		conn.WriteMessage(websocket.TextMessage, []byte(">>> "+response))
	}
}
