package handlers

import (
	"bufio"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type StatsResponse struct {
	CPUPercent float64 `json:"cpuPercent"`
	RAMUsed    uint64  `json:"ramUsed"`
	RAMTotal   uint64  `json:"ramTotal"`
	DiskUsed   uint64  `json:"diskUsed"`
	DiskTotal  uint64  `json:"diskTotal"`
	Uptime     float64 `json:"uptime"`
}

func GetStats(c *gin.Context) {
	stats := StatsResponse{}
	stats.RAMTotal, stats.RAMUsed = getMemInfo()
	stats.CPUPercent = getCPUPercent()
	stats.Uptime = getUptime()
	stats.DiskUsed, stats.DiskTotal = getDiskInfo()
	c.JSON(http.StatusOK, stats)
}

func getMemInfo() (total, used uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer file.Close()

	var memTotal, memFree, buffers, cached uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			memTotal = val
		case "MemFree:":
			memFree = val
		case "Buffers:":
			buffers = val
		case "Cached:":
			cached = val
		}
	}
	total = memTotal / 1024
	used = (memTotal - memFree - buffers - cached) / 1024
	return
}

func getCPUPercent() float64 {
	cpu1 := readCPUStat()
	time.Sleep(100 * time.Millisecond)
	cpu2 := readCPUStat()

	total := float64(cpu2[0]-cpu1[0]) + float64(cpu2[1]-cpu1[1]) +
		float64(cpu2[2]-cpu1[2]) + float64(cpu2[3]-cpu1[3])
	idle := float64(cpu2[3] - cpu1[3])

	if total == 0 {
		return 0
	}
	return (1 - idle/total) * 100
}

func readCPUStat() [4]uint64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return [4]uint64{}
	}
	defer file.Close()

	var cpu [4]uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			for i := 1; i <= 4 && i < len(fields); i++ {
				cpu[i-1], _ = strconv.ParseUint(fields[i], 10, 64)
			}
			break
		}
	}
	return cpu
}

func getUptime() float64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	uptime, _ := strconv.ParseFloat(fields[0], 64)
	return uptime
}
