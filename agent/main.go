package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.POST("/start", func(c *gin.Context) {
		if err := exec.Command("systemctl", "start", "minecraft").Run(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "started"})
	})

	r.POST("/stop", func(c *gin.Context) {
		if err := exec.Command("systemctl", "stop", "minecraft").Run(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "stopped"})
	})

	r.GET("/status", func(c *gin.Context) {
		out, _ := exec.Command("systemctl", "is-active", "minecraft").Output()
		c.JSON(200, gin.H{"active": strings.TrimSpace(string(out))})
	})

	r.GET("/logs", func(c *gin.Context) {
		cmd := exec.Command("journalctl", "-u", "minecraft", "-n", "100", "--no-pager")
		out, err := cmd.Output()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		lines := []string{}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		c.JSON(200, lines)
	})

	r.GET("/logs/stream", func(c *gin.Context) {
		cmd := exec.Command("journalctl", "-u", "minecraft", "-f", "--no-pager")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := cmd.Start(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer cmd.Process.Kill()

		c.Stream(func(w *bufio.Writer) bool {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				c.SSEvent("log", scanner.Text())
				w.Flush()
			}
			return false
		})
	})

	socketPath := "/tmp/mc-agent.sock"
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("❌ Socket: %v", err)
	}
	os.Chmod(socketPath, 0666)

	log.Println("✅ mc-agent démarré sur", socketPath)
	http.Serve(listener, r)
}