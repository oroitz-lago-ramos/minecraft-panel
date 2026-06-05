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

// activeWorldDir retourne le chemin du dossier "world" actif
// c'est toujours /home/deploy/minecraft/world (level-name=world dans server.properties)
const activeWorldDir = "/home/deploy/minecraft/world"

// ─── Helpers uptime ──────────────────────────────────────────────────────────

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

// ─── Active world ─────────────────────────────────────────────────────────────

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

// ─── Helpers fichiers ─────────────────────────────────────────────────────────

func unzip(src, dst string) error {
	return exec.Command("unzip", "-o", src, "-d", dst).Run()
}

// copyWorld copie le dossier "world" entier (src → dst)
// et corrige les permissions pour éviter les session.lock
func copyWorld(src, dst string) error {
	os.RemoveAll(dst)
	if err := exec.Command("cp", "-r", src, dst).Run(); err != nil {
		return err
	}
	exec.Command("chown", "-R", "deploy:deploy", dst).Run()
	return nil
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// ── Start / Stop / Status ─────────────────────────────────────────────────

	r.POST("/start", func(c *gin.Context) {
		if err := exec.Command("sudo", "systemctl", "start", "minecraft").Run(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "started"})
	})

	r.POST("/stop", func(c *gin.Context) {
		if err := exec.Command("sudo", "systemctl", "stop", "minecraft").Run(); err != nil {
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

	// ── Logs ──────────────────────────────────────────────────────────────────

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

	// ── Worlds ────────────────────────────────────────────────────────────────
	// Paper 26.1 : un seul dossier "world/" contient tout (overworld + nether + end)
	// via world/dimensions/minecraft/{overworld,the_nether,the_end}/
	// On ne gère plus world_nether/ et world_the_end/ séparément.

	r.GET("/worlds", func(c *gin.Context) {
		entries, err := os.ReadDir(worldsDir)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		active := getActiveWorld()
		worlds := []gin.H{}
		for _, e := range entries {
			// On ignore les fichiers cachés comme .active
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

		// Vérifie que la map demandée existe dans worlds/
		target := filepath.Join(worldsDir, body.Name)
		if _, err := os.Stat(target); os.IsNotExist(err) {
			c.JSON(404, gin.H{"error": "map introuvable"})
			return
		}

		// 1. Stop le serveur Minecraft
		exec.Command("systemctl", "stop", "minecraft").Run()
		time.Sleep(3 * time.Second)

		// 2. Sauvegarde le world actif dans worlds/<active>/world/
		active := getActiveWorld()
		if active != "" && active != body.Name {
			saveDir := filepath.Join(worldsDir, active)
			os.MkdirAll(saveDir, 0755)
			// Paper 26.1 : on sauvegarde uniquement le dossier "world"
			src := activeWorldDir                  // /home/deploy/minecraft/world
			dst := filepath.Join(saveDir, "world") // worlds/<active>/world
			if err := copyWorld(src, dst); err != nil {
				c.JSON(500, gin.H{"error": "sauvegarde échouée: " + err.Error()})
				return
			}
		}

		// 3. Supprime le world actif du répertoire minecraft/
		os.RemoveAll(activeWorldDir)

		// 4. Copie le nouveau world depuis worlds/<name>/world/
		src := filepath.Join(target, "world")
		if _, err := os.Stat(src); os.IsNotExist(err) {
			// La map n'a pas encore de dossier world/ → le serveur en créera un nouveau
			c.JSON(200, gin.H{"status": "switched (nouveau world)", "world": body.Name})
			setActiveWorld(body.Name)
			exec.Command("systemctl", "start", "minecraft").Run()
			return
		}
		if err := copyWorld(src, activeWorldDir); err != nil {
			c.JSON(500, gin.H{"error": "copie échouée: " + err.Error()})
			return
		}

		// 5. Met à jour .active
		setActiveWorld(body.Name)

		// 6. Restart le serveur
		exec.Command("systemctl", "start", "minecraft").Run()

		c.JSON(200, gin.H{"status": "switched", "world": body.Name})
	})

	r.POST("/worlds/upload", func(c *gin.Context) {
		file, err := c.FormFile("world")
		if err != nil {
			c.JSON(400, gin.H{"error": "fichier requis"})
			return
		}
		// Le zip doit contenir un dossier "world/" à la racine (structure Paper 26.1)
		name := strings.TrimSuffix(file.Filename, ".zip")
		destDir := filepath.Join(worldsDir, name)
		zipPath := destDir + ".zip"

		if err := c.SaveUploadedFile(file, zipPath); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		os.MkdirAll(destDir, 0755)
		if err := unzip(zipPath, destDir); err != nil {
			os.Remove(zipPath)
			c.JSON(500, gin.H{"error": "extraction échouée: " + err.Error()})
			return
		}
		os.Remove(zipPath)
		exec.Command("chown", "-R", "deploy:deploy", destDir).Run()
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
		// On zippe le dossier entier (qui contient world/)
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

	// ── Stats VPS ─────────────────────────────────────────────────────────────

	r.GET("/stats", func(c *gin.Context) {
		// CPU
		cpuOut, _ := exec.Command("bash", "-c",
			`top -bn1 | grep "Cpu(s)" | awk '{print $2}'`).Output()
		cpu := strings.TrimSpace(string(cpuOut))

		// RAM
		ramOut, _ := exec.Command("bash", "-c",
			`free -m | awk 'NR==2{print $3" "$2}'`).Output()
		ramParts := strings.Fields(string(ramOut))
		ramUsed, ramTotal := "0", "0"
		if len(ramParts) == 2 {
			ramUsed = ramParts[0]
			ramTotal = ramParts[1]
		}

		// Disk
		diskOut, _ := exec.Command("bash", "-c",
			`df -m / | awk 'NR==2{print $3" "$2}'`).Output()
		diskParts := strings.Fields(string(diskOut))
		diskUsed, diskTotal := "0", "0"
		if len(diskParts) == 2 {
			diskUsed = diskParts[0]
			diskTotal = diskParts[1]
		}

		c.JSON(200, gin.H{
			"cpu":        cpu,
			"ram_used":   ramUsed,
			"ram_total":  ramTotal,
			"disk_used":  diskUsed,
			"disk_total": diskTotal,
		})
	})

	// ── Socket Unix ───────────────────────────────────────────────────────────

	socketPath := "/tmp/mc-agent.sock"
	os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("❌ Socket: %v", err)
	}
	os.Chmod(socketPath, 0660)
	log.Println("✅ mc-agent démarré sur", socketPath)
	http.Serve(listener, r)
}
