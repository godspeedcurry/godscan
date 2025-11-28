package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/viper"

	"github.com/malfunkt/iprange"
)

type ProtocolInfo struct {
	Ip   string
	Port int
}
type ServiceInfo struct {
	Addr   string
	Banner string
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

func ipLessThanOrEqual(a, b net.IP) bool {
	for i := range a {
		if a[i] < b[i] {
			return true
		}
		if a[i] > b[i] {
			return false
		}
	}
	return true
}

func incrementIP(ip net.IP) net.IP {
	nextIP := make(net.IP, len(ip))
	copy(nextIP, ip)
	for i := len(nextIP) - 1; i >= 0; i-- {
		nextIP[i]++
		if nextIP[i] > 0 {
			break
		}
	}
	return nextIP
}

// convertIPListToPool 将给定的IP列表转换为IP池
func convertIPListToPool(ipList []string) ([]net.IP, error) {
	var ipPool []net.IP

	for _, ipStr := range ipList {
		ipStr = strings.TrimSpace(ipStr)
		if strings.Contains(ipStr, "-") { // 处理范围格式的IP
			ipRange := strings.Split(ipStr, "-")
			if len(ipRange) != 2 {
				return nil, fmt.Errorf("invalid IP range format: %s", ipStr)
			}

			startIP := net.ParseIP(strings.TrimSpace(ipRange[0]))
			endIP := net.ParseIP(strings.TrimSpace(ipRange[1]))
			if startIP == nil || endIP == nil {
				return nil, fmt.Errorf("invalid IP address in range: %s", ipStr)
			}

			for ip := startIP; ipLessThanOrEqual(ip, endIP); ip = incrementIP(ip) {
				ipPool = append(ipPool, ip)
			}
		} else if strings.Contains(ipStr, "/") { // 处理CIDR格式的IP
			ipr, err := iprange.Parse(ipStr)
			if err != nil {
				return nil, fmt.Errorf("Error parsing CIDR IP: %v", err)
			}
			ipPool = append(ipPool, ipr.Expand()...)
		} else { // 单个IP
			parsedIP := net.ParseIP(ipStr)
			if parsedIP == nil {
				return nil, fmt.Errorf("invalid IP address: %s", ipStr)
			}
			ipPool = append(ipPool, parsedIP)
		}
	}

	return ipPool, nil
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
	ips, err := convertIPListToPool(strings.Split(IpRange, ","))
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
	go func(arr []net.IP) {
		for _, ip := range arr {
			for _, port := range ports_list {
				taskChan <- ProtocolInfo{Ip: ip.String(), Port: port}
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
