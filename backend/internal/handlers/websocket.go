package handlers

import (
	"bufio"
	"log"
	"net/http"
	"os/exec"

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
	mc *minecraft.Server
}

func NewWSHandler(mc *minecraft.Server) *WSHandler {
	return &WSHandler{mc: mc}
}

func (h *WSHandler) Console(c *gin.Context) {
	// 1. Upgrader la connexion HTTP → WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// 2. Lire les logs Minecraft en temps réel
	cmd := exec.Command("tail", "-f", "-n", "50", "/home/deploy/minecraft/logs/latest.log")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Erreur lecture logs"))
		return
	}

	if err := cmd.Start(); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Erreur démarrage journalctl"))
		return
	}
	defer cmd.Process.Kill()

	// 3. Goroutine — envoyer les logs au client
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
				break
			}
		}
	}()

	// 4. Recevoir les commandes du client
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		command := string(msg)
		response, err := h.mc.SendCommand(command)
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("Erreur: "+err.Error()))
			continue
		}
		conn.WriteMessage(websocket.TextMessage, []byte(">>> "+response))
	}
}
