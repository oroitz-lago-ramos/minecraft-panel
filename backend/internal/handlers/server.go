package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oroitz-lago-ramos/minecraft-panel/internal/minecraft"
)

type ServerHandler struct {
	mc *minecraft.Server
}

func NewServerHandler(mc *minecraft.Server) *ServerHandler {
	return &ServerHandler{mc: mc}
}

func (h *ServerHandler) Status(c *gin.Context) {
	status, err := h.mc.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *ServerHandler) Start(c *gin.Context) {
	if err := h.mc.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "serveur démarré"})
}

func (h *ServerHandler) Stop(c *gin.Context) {
	if err := h.mc.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "serveur arrêté"})
}

func (h *ServerHandler) Players(c *gin.Context) {
	players, err := h.mc.GetPlayers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convertir []string en []map pour le frontend
	result := make([]map[string]string, len(players))
	for i, name := range players {
		result[i] = map[string]string{"name": name}
	}
	c.JSON(http.StatusOK, result)
}

func (h *ServerHandler) SendCommand(c *gin.Context) {
	var body struct {
		Command string `json:"command" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "commande requise"})
		return
	}
	response, err := h.mc.SendCommand(body.Command)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"response": response})
}
