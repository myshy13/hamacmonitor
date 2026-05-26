package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

func padWidth(str string, width int) string {
	if len(str) > width {
		return str[:width]
	}
	return fmt.Sprintf("%-*s", width, str)
}

// --- System Metrics (Thread-safe read/write via Mutex) --- (I didn't make that, Gemini did)
var (
	statsMu   sync.RWMutex
	diskStats = "Fetching..."
	memStats  = "Fetching..."
	cpuStats  = "Fetching..."
	batStats  = "Loading..."
	PrivateIP = "Fetching..."
	PublicIP  = "Fetching..."
)

func checkDiskSpace() string {
	cmd := exec.Command("bash", "-c", "df -h / | grep /")
	output, err := cmd.Output()
	if err != nil {
		return "Error getting disk info"
	}
	return strings.TrimSpace(string(output))
}

func checkRamUsage() string {
	cmd := exec.Command("bash", "-c", "top -l 1 | grep 'PhysMem:'")
	out, err := cmd.Output()
	if err != nil {
		return "Error getting ram details"
	}
	return strings.TrimSpace(string(out))
}

func checkBattery() string {
	cmd := exec.Command("bash", "-c", "pmset -g batt | grep -o '[0-9]*%' | tr -d '%'")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return "100"
	}
	return strings.TrimSpace(string(out))
}

func checkCpuUsage() string {
	cmd := exec.Command("bash", "-c", "top -l 1 | grep -E '^CPU usage:' | awk '{print $3}'")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return "0.0%"
	}
	return strings.TrimSpace(string(out))
}

func fetchPublicIP() string {
	client := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest("GET", "https://ifconfig.me/", nil)
	req.Header.Set("User-Agent", "curl/7.68.0")
	res, err := client.Do(req)
	if err != nil {
		return "No public IP"
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "Error"
	}
	return strings.TrimSpace(string(body))
}

func fetchPrivateIP() string {
	cmd := exec.Command("bash", "-c", "ipconfig getifaddr en0")
	out, err := cmd.Output()
	if err != nil {
		return "No Private IP"
	}
	return strings.TrimSpace(string(out))
}

func listenToKeys(inputChan chan string) {
	exec.Command("stty", "-f", "/dev/tty", "cbreak", "min", "1", "-echo").Run()
	defer exec.Command("stty", "-f", "/dev/tty", "sane").Run()

	var makeByte = make([]byte, 3)
	for {
		os.Stdin.Read(makeByte)
		if makeByte[0] == 27 && makeByte[1] == 91 {
			switch makeByte[2] {
			case 65:
				inputChan <- "UP"
			case 66:
				inputChan <- "DOWN"
			}
		} else if makeByte[0] == 3 {
			os.Exit(0)
		}
		makeByte = make([]byte, 3)
	}
}

func logStats(currentPage int) {
	statsMu.RLock()
	currentDisk := diskStats
	currentMem := memStats
	currentCpu := cpuStats
	currentBat := batStats
	currentPubIP := PublicIP
	currentPrivIP := PrivateIP
	statsMu.RUnlock()

	selectedBg := color.New(color.BgCyan).SprintFunc()
	batEmptyBg := color.New(color.FgBlack).SprintFunc()
	batFullBg := color.New(color.FgCyan).SprintFunc()
	primary := color.New(color.FgCyan).SprintFunc()

	batNum, err := strconv.Atoi(strings.TrimSuffix(currentBat, "%"))
	if err != nil {
		batNum = 100
	}

	totalSegments := 17
	filledSegments := (batNum * totalSegments) / 100
	batRows := make([]string, totalSegments)

	for i := range totalSegments {
		if i < (totalSegments - filledSegments) {
			batRows[i] = batEmptyBg("▒▒▒▒▒▒")
		} else {
			batRows[i] = batFullBg("██████")
		}
	}

	fmt.Print("\033[H")

	var calcPrefix = func(index int) string {
		if currentPage == index {
			return ">"
		}
		return " "
	}

	var calcBg = func(origin string) string {
		if strings.HasPrefix(origin, ">") {
			return selectedBg(origin)
		}
		return origin
	}

	fmt.Println("╭─────────┬──────────────────────────────────────────────────────────────────── " + primary("Battery: "+padWidth(fmt.Sprint(batNum)+"%", 4)) + " ╮")
	fmt.Println("│ "+calcBg(calcPrefix(0)+" Stats")+" │", padWidth("", 74), "│"+batRows[0]+"│")
	fmt.Println("│ "+calcBg(calcPrefix(1)+" Net")+"   │  OS HamacMonitor v0.3.2          ", padWidth("", 42), "│"+batRows[1]+"│")
	fmt.Printf("│         │  OS: %-59s"+"           │"+batRows[2]+"│"+"\n", runtime.GOOS+"           ")

	fmt.Printf("│         │  Timestamp: %-62s "+"│"+batRows[3]+"│"+"\n", time.Now().Format("15:04:05"))
	fmt.Println("│         │", padWidth("", 74), "│"+batRows[4]+"│")

	switch currentPage {
	case 0:
		fmt.Println("│         ├───────── " + primary("Storage") + " ──────────────────────────────────────────────────────────┤" + batRows[5] + "│")
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[6]+"│")
		fmt.Printf("│         │  %-73s "+"│"+batRows[7]+"│"+"\n", padWidth(currentDisk, 73))
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[8]+"│")

		fmt.Println("│         ├───────── " + primary("Memory") + " ───────────────────────────────────────────────────────────┤" + batRows[9] + "│")
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[10]+"│")
		fmt.Printf("│         │  %-73s "+"│"+batRows[11]+"│"+"\n", padWidth(currentMem, 73))
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[12]+"│")

		fmt.Println("│         ├───────── " + primary("CPU Usage") + " ────────────────────────────────────────────────────────┤" + batRows[13] + "│")
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[14]+"│")
		fmt.Printf("│         │  Percent usage: %-58s"+" │"+batRows[15]+"│"+"\n", padWidth(currentCpu, 58))
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[16]+"│")
		fmt.Print("╰─────────┴────────────────────────────────────────────────────────────────────────────┴──────╯\n")
	case 1:
		fmt.Println("│         ├───────── " + primary("Ip Address ── Public") + " ─────────────────────────────────────────────┤" + batRows[5] + "│")
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[6]+"│")
		fmt.Printf("│         │  %-73s "+"│"+batRows[7]+"│"+"\n", padWidth(currentPubIP, 73))
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[8]+"│")

		fmt.Println("│         ├───────── " + primary("Ip Address ── Private") + " ────────────────────────────────────────────┤" + batRows[9] + "│")
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[10]+"│")
		fmt.Printf("│         │  %-73s "+"│"+batRows[11]+"│"+"\n", padWidth(currentPrivIP, 73))
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[12]+"│")

		fmt.Println("│         │", padWidth("", 74), "│"+batRows[13]+"│")
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[14]+"│")
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[15]+"│")
		fmt.Println("│         │", padWidth("", 74), "│"+batRows[16]+"│")
		fmt.Print("╰─────────┴────────────────────────────────────────────────────────────────────────────┴──────╯\n")
	}
}

func main() {
	fmt.Print("\033[2J")
	fmt.Print("\033[H")
	fmt.Print("\n\n\n")

	fmt.Println(" _                                                            _         ")
	fmt.Println("| |__   __ _ _ __ ___   __ _  ___ _ __ ___   ___  _ __  _  __| |_ ___  _ __ ")
	fmt.Println("| '_ \\ / _" + "`" + " | '_ " + "`" + " _ \\ / _" + "`" + " |/ __| '_ " + "`" + " _ \\ / _ \\| '_ \\| |/ _" + "`" + " __/ _ \\| '__|")
	fmt.Println("| | | | (_| | | | | | | (_| | (__| | | | | | (_) | | | | | | (_| || (_) | |   ")
	fmt.Println("|_| |_|\\__,_|_| |_| |_|\\__,_|\\___|_| |_| |_|\\___/|_| |_|_|\\__\\__,_\\___/|_|   ")

	time.Sleep(time.Second)

	currentPage := 0
	maxPages := 1

	inputChan := make(chan string)
	go listenToKeys(inputChan)

	go func() {
		for {
			d := checkDiskSpace()
			m := checkRamUsage()
			c := checkCpuUsage()
			b := checkBattery()
			pPriv := fetchPrivateIP()

			statsMu.Lock()
			diskStats = d
			memStats = m
			cpuStats = c
			batStats = b
			PrivateIP = pPriv
			statsMu.Unlock()

			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		for {
			pPub := fetchPublicIP()
			statsMu.Lock()
			PublicIP = pPub
			statsMu.Unlock()
			time.Sleep(30 * time.Second)
		}
	}()

	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case key := <-inputChan:
			if key == "UP" && currentPage > 0 {
				currentPage--
			} else if key == "DOWN" && currentPage < maxPages {
				currentPage++
			}
			logStats(currentPage)

		case <-ticker.C:
			logStats(currentPage)
		}
	}
}
