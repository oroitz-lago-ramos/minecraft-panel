package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const propsFile = "/home/deploy/minecraft/server.properties"

func formatUptime(totalSeconds int64) string {
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if days > 0 {
		return fmt.Sprintf("%dj %dh %dmin", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dmin %ds", hours, minutes, seconds)
	}
	return fmt.Sprintf("%dmin %ds", minutes, seconds)
}

func getVPSUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "erreur"
	}
	fields := strings.Fields(string(data))
	totalSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "erreur"
	}
	return formatUptime(int64(totalSeconds))
}

func getMinecraftUptime() string {
	cmd := exec.Command("systemctl", "show", "minecraft", "--property=ActiveEnterTimestamp")
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	out, err := cmd.Output()
	if err != nil {
		return "offline"
	}
	line := strings.TrimSpace(string(out))
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "offline"
	}
	t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", parts[1])
	if err != nil {
		return "offline"
	}
	return formatUptime(int64(time.Since(t).Seconds()))
}

func getActiveWorld() string {
	data, err := os.ReadFile(propsFile)
	if err != nil {
		return "survival"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "level-name=") {
			return strings.TrimPrefix(line, "level-name=")
		}
	}
	return "survival"
}

func setActiveWorld(name string) {
	data, err := os.ReadFile(propsFile)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "level-name=") {
			lines[i] = "level-name=" + name
		}
	}
	os.WriteFile(propsFile, []byte(strings.Join(lines, "\n")), 0644)
}

func copyDir(src, dst string) error {
	return exec.Command("cp", "-r", src, dst).Run()
}

func unzip(src, dst string) error {
	return exec.Command("unzip", "-o", src, "-d", dst).Run()
}

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

	r.GET("/uptime", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"vps_uptime":       getVPSUptime(),
			"minecraft_uptime": getMinecraftUptime(),
		})
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

		scanner := bufio.NewScanner(stdout)
		c.Stream(func(w io.Writer) bool {
			if scanner.Scan() {
				c.SSEvent("log", scanner.Text())
				return true
			}
			return false
		})
	})

	r.GET("/worlds", func(c *gin.Context) {
		entries, err := os.ReadDir("/home/deploy/minecraft/worlds")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		active := getActiveWorld()

		worlds := []gin.H{}
		for _, e := range entries {
			if e.IsDir() {
				worlds = append(worlds, gin.H{
					"name":   e.Name(),
					"active": e.Name() == active,
				})
			}
		}
		c.JSON(200, worlds)
	})

	r.POST("/worlds/switch", func(c *gin.Context) {
		var body struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(400, gin.H{"error": "nom requis"})
			return
		}

		baseDir := "/home/deploy/minecraft/worlds"
		mcDir := "/home/deploy/minecraft"
		target := filepath.Join(baseDir, body.Name)

		if _, err := os.Stat(target); os.IsNotExist(err) {
			c.JSON(404, gin.H{"error": "map introuvable"})
			return
		}

		// 1. Stop serveur
		exec.Command("systemctl", "stop", "minecraft").Run()
		time.Sleep(3 * time.Second)

		// 2. Sauvegarde la map active
		active := getActiveWorld()
		if active != "" && active != body.Name {
			saveDir := filepath.Join(baseDir, active)
			os.MkdirAll(saveDir, 0755)
			os.Rename(filepath.Join(mcDir, "world"), filepath.Join(saveDir, "world"))
			os.Rename(filepath.Join(mcDir, "world_nether"), filepath.Join(saveDir, "world_nether"))
			os.Rename(filepath.Join(mcDir, "world_the_end"), filepath.Join(saveDir, "world_the_end"))
		}

		// 3. Copie la nouvelle map
		copyDir(filepath.Join(target, "world"), filepath.Join(mcDir, "world"))
		copyDir(filepath.Join(target, "world_nether"), filepath.Join(mcDir, "world_nether"))
		copyDir(filepath.Join(target, "world_the_end"), filepath.Join(mcDir, "world_the_end"))

		// 4. Met à jour server.properties
		setActiveWorld(body.Name)

		// 5. Restart serveur
		exec.Command("systemctl", "start", "minecraft").Run()

		c.JSON(200, gin.H{"status": "switched", "world": body.Name})
	})

	r.POST("/worlds/upload", func(c *gin.Context) {
		file, err := c.FormFile("world")
		if err != nil {
			c.JSON(400, gin.H{"error": "fichier requis"})
			return
		}

		name := strings.TrimSuffix(file.Filename, ".zip")
		destDir := filepath.Join("/home/deploy/minecraft/worlds", name)
		zipPath := destDir + ".zip"

		if err := c.SaveUploadedFile(file, zipPath); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		os.MkdirAll(destDir, 0755)
		if err := unzip(zipPath, destDir); err != nil {
			c.JSON(500, gin.H{"error": "extraction échouée: " + err.Error()})
			return
		}
		os.Remove(zipPath)

		c.JSON(200, gin.H{"status": "uploaded", "world": name})
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
