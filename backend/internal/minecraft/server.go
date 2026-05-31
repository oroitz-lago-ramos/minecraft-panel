package minecraft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strings"

	"github.com/gorcon/rcon"
)

type ServerStatus struct {
	Online     bool   `json:"online"`
	Players    int    `json:"players"`
	MaxPlayers int    `json:"maxPlayers"`
	Version    string `json:"version"`
	Uptime     string `json:"uptime"`
}

type World struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type Server struct {
	rconAddr     string
	rconPassword string
	agentClient  *http.Client
}

func NewServer(rconAddr, rconPassword string) *Server {
	agentClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/tmp/mc-agent.sock")
			},
		},
	}
	return &Server{
		rconAddr:     rconAddr,
		rconPassword: rconPassword,
		agentClient:  agentClient,
	}
}

func (s *Server) callAgent(method, path string) error {
	req, err := http.NewRequest(method, "http://agent"+path, nil)
	if err != nil {
		return err
	}
	resp, err := s.agentClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent inaccessible: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("agent error: %d", resp.StatusCode)
	}
	return nil
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
	response, err := s.SendCommand("list")
	if err != nil {
		return status, nil
	}
	var current, max int
	fmt.Sscanf(response, "There are %d of a max of %d players online", &current, &max)
	status.Players = current
	status.MaxPlayers = max
	return status, nil
}

func (s *Server) Start() error {
	return s.callAgent("POST", "/start")
}

func (s *Server) Stop() error {
	return s.callAgent("POST", "/stop")
}

func (s *Server) GetUptime() (map[string]string, error) {
	req, err := http.NewRequest("GET", "http://agent/uptime", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.agentClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent inaccessible: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Server) GetWorlds() ([]World, error) {
	req, err := http.NewRequest("GET", "http://agent/worlds", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.agentClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent inaccessible: %v", err)
	}
	defer resp.Body.Close()
	var worlds []World
	if err := json.NewDecoder(resp.Body).Decode(&worlds); err != nil {
		return nil, err
	}
	return worlds, nil
}

func (s *Server) SwitchWorld(name string) error {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequest("POST", "http://agent/worlds/switch", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.agentClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent inaccessible: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("agent error: %d", resp.StatusCode)
	}
	return nil
}

func (s *Server) UploadWorld(filename string, file io.Reader) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("world", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	req, err := http.NewRequest("POST", "http://agent/worlds/upload", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := s.agentClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent inaccessible: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("agent error: %d", resp.StatusCode)
	}
	return nil
}

func (s *Server) DeleteWorld(name string) error {
	req, err := http.NewRequest("DELETE", "http://agent/worlds/"+name, nil)
	if err != nil {
		return err
	}
	resp, err := s.agentClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent inaccessible: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("agent error: %d", resp.StatusCode)
	}
	return nil
}

func (s *Server) BackupWorld(name string, c interface{ Header(string, string); File(string) }) error {
	req, err := http.NewRequest("GET", "http://agent/worlds/"+name+"/backup", nil)
	if err != nil {
		return err
	}
	resp, err := s.agentClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent inaccessible: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("agent error: %d", resp.StatusCode)
	}
	return nil
}

func (s *Server) GetPlayers() ([]string, error) {
	response, err := s.SendCommand("list")
	if err != nil {
		return nil, err
	}
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

func (s *Server) AgentClient() *http.Client {
	return s.agentClient
}