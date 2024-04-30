package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"

	"github.com/malfunkt/iprange"
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

func handleWorker(ip string, ports chan int, results chan ProtocolInfo) {
	for p := range ports {
		address := fmt.Sprintf("%s:%d", ip, p)
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err != nil {
			results <- ProtocolInfo{Ip: ip, Port: -p}
			continue
		}
		results <- ProtocolInfo{Ip: ip, Port: p}
		conn.Close()
	}
}

func PortScan(IpRange string, PortRange string) {
	ips, err := convertIPListToPool(strings.Split(IpRange, ","))
	if err != nil {
		Error("%s", err)
		return
	}
	// return
	ports_list, err := parsePorts(PortRange)
	if err != nil {
		Error("%s", err)
		return
	}

	bar := pb.StartNew(len(ports_list) * len(ips))

	var allOpen []ProtocolInfo

	for _, ipNet := range ips {
		ports := make(chan int, 50)
		results := make(chan ProtocolInfo)
		var openSlice []ProtocolInfo

		// 任务生产者-分发任务 (新起一个 goroutinue ，进行分发数据)
		go func(arr []int) {
			for i := 0; i < len(arr); i++ {
				ports <- arr[i]
			}
		}(ports_list)

		// 任务消费者-处理任务  (每一个端口号都分配一个 goroutinue ，进行扫描)
		// 结果生产者-每次得到结果 再写入 结果 chan 中
		for i := 0; i < cap(ports); i++ {
			go handleWorker(ipNet.String(), ports, results)
		}

		// 结果消费者-等待收集结果 (main中的 goroutinue 不断从 chan 中阻塞式读取数据)
		for i := 0; i < len(ports_list); i++ {
			resPortInfo := <-results
			if resPortInfo.Port > 0 {
				openSlice = append(openSlice, resPortInfo)
			}
			bar.Increment()
		}

		// 关闭 chan
		close(ports)
		close(results)

		// 输出
		allOpen = append(allOpen, openSlice...)

	}
	bar.Finish()
	for _, open := range allOpen {
		Success("%s:%-8d Open", open.Ip, open.Port)
	}
	for _, open := range allOpen {
		ScanWithIpAndPort(open.Ip, open.Port, "tcp")
	}
}
