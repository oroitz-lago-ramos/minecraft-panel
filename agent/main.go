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
const worldsDir = "/home/deploy/minecraft/worlds"
const mcDir = "/home/deploy/minecraft"
const activeFile = "/home/deploy/minecraft/worlds/.active"

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
	data, err := os.ReadFile(activeFile)
	if err != nil {
		return "survival"
	}
	return strings.TrimSpace(string(data))
}

func setActiveWorld(name string) {
	os.WriteFile(activeFile, []byte(name), 0644)
}

func copyDir(src, dst string) error {
	os.RemoveAll(dst)
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
		entries, err := os.ReadDir(worldsDir)
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

		target := filepath.Join(worldsDir, body.Name)
		if _, err := os.Stat(target); os.IsNotExist(err) {
			c.JSON(404, gin.H{"error": "map introuvable"})
			return
		}

		// 1. Stop serveur
		exec.Command("systemctl", "stop", "minecraft").Run()
		time.Sleep(3 * time.Second)

		// 2. Sauvegarde la map active dans worlds/
		active := getActiveWorld()
		if active != "" && active != body.Name {
			saveDir := filepath.Join(worldsDir, active)
			os.MkdirAll(saveDir, 0755)
			for _, dir := range []string{"world", "world_nether", "world_the_end"} {
				src := filepath.Join(mcDir, dir)
				dst := filepath.Join(saveDir, dir)
				os.RemoveAll(dst)
				exec.Command("cp", "-r", src, dst).Run()
				os.RemoveAll(src)
			}
		}

		// 3. Copie la nouvelle map vers world/ world_nether/ world_the_end/
		for _, dir := range []string{"world", "world_nether", "world_the_end"} {
			src := filepath.Join(target, dir)
			dst := filepath.Join(mcDir, dir)
			os.RemoveAll(dst)
			if _, err := os.Stat(src); err == nil {
				exec.Command("cp", "-r", src, dst).Run()
			}
		}

		// 4. Met à jour .active
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
		destDir := filepath.Join(worldsDir, name)
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

	r.DELETE("/worlds/:name", func(c *gin.Context) {
		name := c.Param("name")
		if name == getActiveWorld() {
			c.JSON(400, gin.H{"error": "impossible de supprimer la map active"})
			return
		}
		target := filepath.Join(worldsDir, name)
		if _, err := os.Stat(target); os.IsNotExist(err) {
			c.JSON(404, gin.H{"error": "map introuvable"})
			return
		}
		if err := os.RemoveAll(target); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "deleted", "world": name})
	})

	r.GET("/worlds/:name/backup", func(c *gin.Context) {
		name := c.Param("name")
		worldDir := filepath.Join(worldsDir, name)
		if _, err := os.Stat(worldDir); os.IsNotExist(err) {
			c.JSON(404, gin.H{"error": "map introuvable"})
			return
		}
		zipPath := filepath.Join("/tmp", name+".zip")
		os.Remove(zipPath)
		cmd := exec.Command("zip", "-r", zipPath, name)
		cmd.Dir = worldsDir
		if err := cmd.Run(); err != nil {
			c.JSON(500, gin.H{"error": "zip échoué: " + err.Error()})
			return
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", name))
		c.Header("Content-Type", "application/zip")
		c.File(zipPath)
		defer os.Remove(zipPath)
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
