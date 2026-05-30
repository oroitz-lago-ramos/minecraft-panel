package minecraft

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/gorcon/rcon"
)

type ServerStatus struct {
	Online  bool   `json:"online"`
	Players int    `json:"players"`
	MaxPlayers int `json:"maxPlayers"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

type Server struct {
	rconAddr     string
	rconPassword string
}

func NewServer(rconAddr, rconPassword string) *Server {
	return &Server{
		rconAddr:     rconAddr,
		rconPassword: rconPassword,
	}
}

func (s *Server) IsOnline() bool {
	conn, err := rcon.Dial(s.rconAddr, s.rconPassword)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *Server) GetStatus() (*ServerStatus, error) {
	status := &ServerStatus{}
	status.Online = s.IsOnline()

	if !status.Online {
		return status, nil
	}

	// Récupérer le nombre de joueurs via RCON
	response, err := s.SendCommand("list")
	if err != nil {
		return status, nil
	}

	// Parser "There are 2 of a max of 20 players online"
	var current, max int
	fmt.Sscanf(response, "There are %d of a max of %d players online", &current, &max)
	status.Players = current
	status.MaxPlayers = max

	return status, nil
}

func (s *Server) Start() error {
	cmd := exec.Command("sudo", "systemctl", "start", "minecraft")
	return cmd.Run()
}

func (s *Server) Stop() error {
	cmd := exec.Command("sudo", "systemctl", "stop", "minecraft")
	return cmd.Run()
}

func (s *Server) GetPlayers() ([]string, error) {
	response, err := s.SendCommand("list")
	if err != nil {
		return nil, err
	}

	// Parser la liste des joueurs
	if !strings.Contains(response, ":") {
		return []string{}, nil
	}

	parts := strings.Split(response, ":")
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		return []string{}, nil
	}

	players := strings.Split(parts[1], ",")
	result := make([]string, 0)
	for _, p := range players {
		name := strings.TrimSpace(p)
		if name != "" {
			result = append(result, name)
		}
	}
	return result, nil
}

func (s *Server) SendCommand(command string) (string, error) {
	conn, err := rcon.Dial(s.rconAddr, s.rconPassword)
	if err != nil {
		return "", fmt.Errorf("impossible de se connecter à RCON: %v", err)
	}
	defer conn.Close()

	response, err := conn.Execute(command)
	if err != nil {
		return "", fmt.Errorf("erreur RCON: %v", err)
	}
	return response, nil
}