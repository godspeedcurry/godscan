package utils

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/viper"

	"github.com/malfunkt/iprange"
	"golang.org/x/net/proxy"
)

type ProtocolInfo struct {
	Ip   string
	Port int
}

func parsePorts(portsStr string) ([]int, error) {
	var ports []int

	portRanges := strings.Split(portsStr, ",")
	for _, portRange := range portRanges {
		portRange = strings.TrimSpace(portRange)

		if strings.Contains(portRange, "-") { // 处理端口范围
			rangeBounds := strings.Split(portRange, "-")
			if len(rangeBounds) != 2 {
				return nil, fmt.Errorf("invalid port range format: %s", portRange)
			}

			startPort, err := strconv.Atoi(strings.TrimSpace(rangeBounds[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port in range: %s", rangeBounds[0])
			}

			endPort, err := strconv.Atoi(strings.TrimSpace(rangeBounds[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port in range: %s", rangeBounds[1])
			}

			if startPort > endPort {
				return nil, fmt.Errorf("invalid port range: start port cannot be greater than end port")
			}

			for port := startPort; port <= endPort; port++ {
				ports = append(ports, port)
			}
		} else { // 处理单个端口
			port, err := strconv.Atoi(portRange)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", portRange)
			}
			ports = append(ports, port)
		}
	}
	uniquePorts, _ := RemoveDuplicateElement(ports)
	return uniquePorts.([]int), nil
}

// ParsePortsString exports port parser for reuse.
func ParsePortsString(portsStr string) ([]int, error) {
	return parsePorts(portsStr)
}

// QuickPortScan scans a single IP for the given TCP ports, returning open ports.
func QuickPortScan(ip string, ports []int, workers int, dialTimeout time.Duration) []int {
	type job struct{ port int }
	type res struct{ port int }
	jobs := make(chan job, len(ports))
	results := make(chan res, len(ports))
	if workers <= 0 {
		workers = 200
	}
	if dialTimeout <= 0 {
		dialTimeout = 2 * time.Second
	}
	dialFn := buildPortDialer(dialTimeout)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				addr := net.JoinHostPort(ip, fmt.Sprintf("%d", j.port))
				var conn net.Conn
				var err error
				if dialFn != nil {
					conn, err = dialFn.Dial("tcp", addr)
				} else {
					conn, err = net.DialTimeout("tcp", addr, dialTimeout)
				}
				if err == nil {
					conn.Close()
					results <- res{port: j.port}
				} else {
					results <- res{port: -j.port}
				}
			}
		}()
	}
	go func() {
		for _, p := range ports {
			jobs <- job{port: p}
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	open := []int{}
	for r := range results {
		if r.port > 0 {
			open = append(open, r.port)
		}
	}
	sort.Ints(open)
	return open
}

func buildPortDialer(timeout time.Duration) proxy.Dialer {
	raw := os.Getenv("ALL_PROXY")
	if raw == "" {
		raw = os.Getenv("all_proxy")
	}
	if raw == "" {
		raw = viper.GetString("port-proxy")
	}
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil
	}
	if !strings.HasPrefix(strings.ToLower(u.Scheme), "socks") {
		return nil
	}
	dialer, err := proxy.SOCKS5("tcp", u.Host, nil, &net.Dialer{Timeout: timeout, KeepAlive: timeout})
	if err != nil {
		return nil
	}
	return dialer
}

// convertTargetListToPool accepts IPs, IP ranges/CIDRs, or domain names.
func convertTargetListToPool(targetList []string) ([]string, error) {
	var targets []string

	for _, raw := range targetList {
		target := sanitizeHost(raw)
		if target == "" {
			continue
		}
		// Try range/CIDR first
		if strings.Contains(target, "/") || strings.Contains(target, "-") {
			if ipr, err := iprange.Parse(target); err == nil {
				for _, ip := range ipr.Expand() {
					targets = append(targets, ip.String())
				}
				continue
			}
		}
		// Plain IP
		if parsed := net.ParseIP(target); parsed != nil {
			targets = append(targets, parsed.String())
			continue
		}
		// Fallback: treat as hostname
		targets = append(targets, target)
	}

	targets = RemoveDuplicatesString(targets)
	return targets, nil
}

func sanitizeHost(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		if u, err := url.Parse(s); err == nil {
			return u.Hostname()
		}
	}
	// Handle scheme-less URLs like //example.com/path
	if strings.HasPrefix(s, "//") {
		if u, err := url.Parse("http:" + s); err == nil {
			return u.Hostname()
		}
	}
	// Try to parse arbitrary URL forms
	if strings.Contains(s, "/") {
		if u, err := url.Parse(s); err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
	}
	return s
}

func handleWorker(tasks <-chan ProtocolInfo, results chan ProtocolInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range tasks {
		address := net.JoinHostPort(task.Ip, strconv.Itoa(task.Port))
		dt := viper.GetInt("port-dial-timeout")
		if dt <= 0 {
			dt = 2
		}
		conn, err := net.DialTimeout("tcp", address, time.Duration(dt)*time.Second)
		if err != nil {
			results <- ProtocolInfo{Ip: task.Ip, Port: -task.Port}
			continue
		}
		results <- ProtocolInfo{Ip: task.Ip, Port: task.Port}
		conn.Close()
	}
}

func PortScan(IpRange string, PortRange string) {
	ips, err := convertTargetListToPool(strings.Split(IpRange, ","))
	if err != nil {
		Error("%s", err)
		return
	}

	ports_list, err := parsePorts(PortRange)
	if err != nil {
		Error("%s", err)
		return
	}
	Info("Total IP(s): %d", len(ips))
	Info("Total Port(s): %d", len(ports_list))
	Info("Total Threads(s): %d", viper.GetInt("threads"))

	bar := pb.StartNew(len(ports_list) * len(ips))

	taskChan := make(chan ProtocolInfo, viper.GetInt("threads"))
	results := make(chan ProtocolInfo)

	var wg sync.WaitGroup

	for i := 0; i < viper.GetInt("threads"); i++ {
		wg.Add(1)
		go handleWorker(taskChan, results, &wg)
	}

	// 任务生产者-分发任务 (新起一个 goroutinue ，进行分发数据)
	go func(arr []string) {
		for _, ip := range arr {
			for _, port := range ports_list {
				taskChan <- ProtocolInfo{Ip: ip, Port: port}
			}
		}
		close(taskChan)
	}(ips)

	go func() {
		wg.Wait()
		close(results)
	}()
	var allOpen []ProtocolInfo
	for resPortInfo := range results {
		if resPortInfo.Port > 0 {
			allOpen = append(allOpen, ProtocolInfo{Ip: resPortInfo.Ip, Port: resPortInfo.Port})
			Success("Open: %s:%d", resPortInfo.Ip, resPortInfo.Port)
		}
		bar.Increment()
	}

	bar.Finish()

	ScanWithIpAndPort(allOpen)
}
