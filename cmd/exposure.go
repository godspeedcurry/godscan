package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ExposureOptions struct {
	UrlFile       string
	Output        string
	Ports         string
	DialTime      int
	Workers       int
	HTTPTimeout   int
	TargetWorkers int
	DNSTimeout    int
}

var exposureOptions ExposureOptions

func init() {
	exposureCmd := &cobra.Command{
		Use:   "exposure",
		Short: "Exposure surface: port reachability + homepage title to XLSX",
		Run: func(cmd *cobra.Command, args []string) {
			if err := exposureOptions.validate(); err != nil {
				utils.Error("%v", err)
				return
			}
			if err := exposureOptions.run(); err != nil {
				utils.Error("%v", err)
			}
		},
	}
	exposureCmd.Flags().StringVar(&exposureOptions.UrlFile, "host-file", "", "Host list file (one per line)")
	exposureCmd.Flags().StringVar(&exposureOptions.UrlFile, "hf", "", "alias of --host-file")
	exposureCmd.Flags().StringVar(&exposureOptions.Output, "output", "", "Output XLSX path (default output/exposure-YYYY-MM-DD.xlsx)")
	exposureCmd.Flags().StringVar(&exposureOptions.Ports, "ports", "21,22,23,25,53,80,81,88,110,111,123,135,137,139,161,177,389,427,443,445,465,500,515,520,523,548,623,626,636,873,902,1080,1099,1433,1521,1604,1645,1701,1883,1900,2049,2181,2375,2379,2425,3128,3306,3389,4730,5060,5222,5351,5353,5432,5555,5601,5672,5683,5900,5938,5984,6000,6379,7001,7077,8080,8081,8443,8545,8686,9000,9001,9042,9092,9200,9418,9999,11211,27017,33848,37777,50000,50070,61616", "Ports to probe (UDP prefixes U: ignored)")
	exposureCmd.Flags().IntVar(&exposureOptions.DialTime, "dial-timeout", 2, "TCP dial timeout (seconds)")
	exposureCmd.Flags().IntVar(&exposureOptions.Workers, "port-workers", 200, "Concurrent port dials per host (goroutines)")
	exposureCmd.Flags().IntVar(&exposureOptions.HTTPTimeout, "http-timeout", 8, "HTTP timeout for title fetch (seconds)")
	exposureCmd.Flags().IntVar(&exposureOptions.TargetWorkers, "target-workers", 20, "Concurrent target workers")
	exposureCmd.Flags().IntVar(&exposureOptions.DNSTimeout, "dns-timeout", 3, "DNS resolve timeout (seconds)")
	rootCmd.AddCommand(exposureCmd)
}

func (o *ExposureOptions) validate() error {
	if o.UrlFile == "" {
		return fmt.Errorf("host-file is required")
	}
	return nil
}

func (o *ExposureOptions) run() error {
	utils.InitHttp()
	logProxyUsage()
	if o.Output == "" {
		now := time.Now()
		o.Output = filepath.Join("output", fmt.Sprintf("exposure-%04d-%02d-%02d.xlsx", now.Year(), now.Month(), now.Day()))
	}
	_ = os.MkdirAll(filepath.Dir(o.Output), 0o755)
	lines := utils.FileReadLine(o.UrlFile)
	if len(lines) == 0 {
		return fmt.Errorf("url list is empty")
	}
	targets := normalizeHosts(lines)
	utils.Info("total targets: %d", len(targets))

	ports, err := utils.ParsePortsString(o.Ports)
	if err != nil {
		return err
	}
	utils.Info("tcp ports to scan: %d", len(ports))

	type result struct {
		Domain string
		IP     string
		Ports  []int
		Title  string
	}
	out := make([]result, 0, len(targets))

	type job struct{ host string }
	jobCh := make(chan job)
	resCh := make(chan result, len(targets))
	workers := o.TargetWorkers
	if workers <= 0 {
		workers = 20
	}
	for i := 0; i < workers; i++ {
		go func() {
			for j := range jobCh {
				host := j.host
				ip := resolveIPv4Timeout(host, time.Duration(o.DNSTimeout)*time.Second)
				if ip == "" {
					utils.Warning("resolve failed: %s", host)
					resCh <- result{Domain: host, IP: "", Ports: nil, Title: ""}
					continue
				}
				openPorts := utils.QuickPortScan(ip, ports, o.Workers, time.Duration(o.DialTime)*time.Second)
				title := fetchTitle(host, o.HTTPTimeout)
				resCh <- result{Domain: host, IP: ip, Ports: openPorts, Title: title}
			}
		}()
	}
	go func() {
		for _, h := range targets {
			jobCh <- job{host: h}
		}
		close(jobCh)
	}()
	for i := 0; i < len(targets); i++ {
		out = append(out, <-resCh)
	}

	rows := [][]string{{"Domain", "IP", "Open Ports", "Title"}}
	for _, r := range out {
		var portStr string
		if len(r.Ports) > 0 {
			strs := make([]string, len(r.Ports))
			for i, p := range r.Ports {
				strs[i] = fmt.Sprintf("%d", p)
			}
			portStr = strings.Join(strs, ",")
		}
		rows = append(rows, []string{r.Domain, r.IP, portStr, r.Title})
	}
	sheets := []utils.Worksheet{{Name: "exposure", Rows: rows}}
	if err := utils.WriteSimpleXLSX(o.Output, sheets); err != nil {
		return err
	}
	utils.Success("exposure.xlsx saved: %s (rows=%d)", o.Output, len(rows)-1)
	return nil
}

func logProxyUsage() {
	portProxy := getenvAny("ALL_PROXY", "all_proxy")
	if portProxy == "" {
		portProxy = viper.GetString("port-proxy")
	}
	if portProxy != "" {
		utils.Info("Port scan via SOCKS: %s", portProxy)
	} else {
		utils.Info("Port scan via SOCKS: direct")
	}
}

func getenvAny(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func normalizeHosts(lines []string) []string {
	m := make(map[string]struct{})
	for _, raw := range lines {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		host := raw
		if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
			if u, err := url.Parse(raw); err == nil {
				host = u.Hostname()
			}
		} else if strings.HasPrefix(raw, "//") {
			if u, err := url.Parse("http:" + raw); err == nil {
				host = u.Hostname()
			}
		}
		if host != "" {
			m[host] = struct{}{}
		}
	}
	var out []string
	for h := range m {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

func parseTCPPorts(raw string) []int {
	return []int{}
}

func resolveIPv4Timeout(host string, timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return ""
	}
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return ips[0].String()
}

func fetchTitle(host string, httpTimeout int) string {
	targets := []string{
		"https://" + host,
		"http://" + host,
	}
	if httpTimeout <= 0 {
		httpTimeout = 6
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(httpTimeout)*time.Second)
	defer cancel()
	client := &http.Client{
		Transport: utils.CloneDefaultTransport(),
		Timeout:   time.Duration(httpTimeout) * time.Second,
	}
	for _, u := range targets {
		req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
		utils.SetHeaders(req)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		title := extractTitle(resp.Body)
		resp.Body.Close()
		if title != "" {
			return title
		}
	}
	return ""
}

func extractTitle(r io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(r, 512*1024))
	if err != nil {
		return ""
	}
	txt := string(data)
	start := strings.Index(strings.ToLower(txt), "<title>")
	if start == -1 {
		return ""
	}
	end := strings.Index(strings.ToLower(txt), "</title>")
	if end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(txt[start+7 : end])
}
