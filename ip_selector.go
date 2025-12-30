package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

// Embedded Steam CDN IP list
// These are example IPs. In a real scenario, this list should be comprehensive.
var steamCDNIPs = []string{
	"104.116.243.163",
	"104.116.243.72",
	"2.17.107.170",
	"2.17.107.243",
	"23.192.228.147",
	"23.192.228.139",
	"23.52.74.14",
	"23.212.62.72",
	"23.212.62.73",
}

type IPSelector struct {
	cachedBestIP string
	lastCheck    time.Time
	mu           sync.RWMutex
}

var globalIPSelector = &IPSelector{}

func (s *IPSelector) GetBestIP(testUrl string) string {
	s.mu.RLock()
	// Cache for 10 minutes
	if s.cachedBestIP != "" && time.Since(s.lastCheck) < 10*time.Minute {
		defer s.mu.RUnlock()
		fmt.Printf("[IPSelector] Using cached best IP: %s (Last check: %v ago)\n", s.cachedBestIP, time.Since(s.lastCheck).Round(time.Second))
		return s.cachedBestIP
	}
	s.mu.RUnlock()

	return s.refreshBestIP(testUrl)
}

func (s *IPSelector) refreshBestIP(testUrl string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check
	if s.cachedBestIP != "" && time.Since(s.lastCheck) < 10*time.Minute {
		return s.cachedBestIP
	}

	// Copy base IPs
	candidateIPs := make([]string, len(steamCDNIPs))
	copy(candidateIPs, steamCDNIPs)

	// Resolve via 8.8.8.8
	fmt.Println("[IPSelector] Resolving cdn.steamusercontent.com via 8.8.8.8...")
	dnsIPs, err := lookupIPWithDNS("8.8.8.8", "cdn.steamusercontent.com")
	if err == nil {
		v4 := 0
		v6 := 0
		for _, ip := range dnsIPs {
			if net.ParseIP(ip).To4() != nil {
				v4++
			} else {
				v6++
			}
		}
		fmt.Printf("[IPSelector] DNS returned %d IPs (IPv4: %d, IPv6: %d): %v\n", len(dnsIPs), v4, v6, dnsIPs)
		candidateIPs = append(candidateIPs, dnsIPs...)
	} else {
		fmt.Printf("[IPSelector] DNS lookup failed: %v\n", err)
	}

	bestIP := selectBestIP(candidateIPs, testUrl)
	if bestIP != "" {
		s.cachedBestIP = bestIP
		s.lastCheck = time.Now()
	}
	return s.cachedBestIP
}

func lookupIPWithDNS(dnsServer string, host string) ([]string, error) {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Second * 5,
			}
			return d.DialContext(ctx, "udp", dnsServer+":53")
		},
	}
	return r.LookupHost(context.Background(), host)
}

func selectBestIP(ips []string, testUrl string) string {
	type result struct {
		ip      string
		latency time.Duration
	}

	fmt.Printf("[IPSelector] Starting ping test for %d IPs...\n", len(ips))

	// 1. TCP Ping Test
	results := make(chan result, len(ips))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for _, ip := range ips {
		go func(targetIP string) {
			start := time.Now()
			d := net.Dialer{Timeout: 1500 * time.Millisecond}
			conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(targetIP, "80"))
			if err == nil {
				conn.Close()
				latency := time.Since(start)
				fmt.Printf("[IPSelector] Ping success: %s - %v\n", targetIP, latency)
				select {
				case results <- result{ip: targetIP, latency: latency}:
				case <-ctx.Done():
				}
			}
		}(ip)
	}

	var pingResults []result
	for i := 0; i < len(ips); i++ {
		select {
		case res := <-results:
			pingResults = append(pingResults, res)
		case <-ctx.Done():
			goto PING_DONE
		}
	}
PING_DONE:

	if len(pingResults) == 0 {
		fmt.Println("[IPSelector] No reachable IP found via Ping")
		return ""
	}

	// Sort by latency
	sort.Slice(pingResults, func(i, j int) bool {
		return pingResults[i].latency < pingResults[j].latency
	})

	fmt.Println("[IPSelector] --- All Ping Results (Sorted) ---")
	for i, res := range pingResults {
		fmt.Printf("  %d. %s: %v\n", i+1, res.ip, res.latency)
	}
	fmt.Println("[IPSelector] --------------------------------")

	// Take top 3 for speed test
	topCount := 3
	if len(pingResults) < topCount {
		topCount = len(pingResults)
	}
	topCandidates := pingResults[:topCount]

	fmt.Printf("[IPSelector] Top %d IPs by latency: %v\n", topCount, topCandidates)

	// 2. Download Speed Test
	type speedResult struct {
		ip    string
		speed float64 // MB/s
		err   error
	}

	speedResults := make(chan speedResult, topCount)
	// Increase timeout to 30s for 5MB test
	speedCtx, speedCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer speedCancel()

	fmt.Println("[IPSelector] Starting download speed test...")

	for _, candidate := range topCandidates {
		go func(ip string) {
			speed, err := testDownloadSpeed(speedCtx, ip, testUrl)
			select {
			case speedResults <- speedResult{ip: ip, speed: speed, err: err}:
			case <-speedCtx.Done():
			}
		}(candidate.ip)
	}

	var bestIP string
	var maxSpeed float64 = -1

	for i := 0; i < topCount; i++ {
		select {
		case res := <-speedResults:
			if res.err != nil {
				fmt.Printf("[IPSelector] Speed test failed for %s: %v\n", res.ip, res.err)
				continue
			}
			fmt.Printf("[IPSelector] %s - Speed: %.2f MB/s\n", res.ip, res.speed)
			if res.speed > maxSpeed {
				maxSpeed = res.speed
				bestIP = res.ip
			}
		case <-speedCtx.Done():
			goto SPEED_DONE
		}
	}
SPEED_DONE:

	if bestIP != "" {
		fmt.Printf("[IPSelector] Best IP selected: %s (Speed: %.2f MB/s)\n", bestIP, maxSpeed)
		return bestIP
	}

	// Fallback to lowest latency if speed test failed for all
	fmt.Printf("[IPSelector] Speed test failed for all, falling back to lowest latency: %s\n", topCandidates[0].ip)
	return topCandidates[0].ip
}

func testDownloadSpeed(ctx context.Context, ip string, downloadUrl string) (float64, error) {
	u, err := url.Parse(downloadUrl)
	if err != nil {
		return 0, err
	}

	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Force connection to specific IP
			_, port, _ := net.SplitHostPort(addr)
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
		},
		TLSHandshakeTimeout: 5 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadUrl, nil)
	if err != nil {
		return 0, err
	}

	// Request first 5MB for better speed measurement
	req.Header.Set("Range", "bytes=0-5242880")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Host", u.Host)

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	// Read body to measure throughput
	written, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, err
	}

	duration := time.Since(start)
	if duration == 0 {
		duration = 1 * time.Millisecond
	}

	speedMBps := float64(written) / 1024 / 1024 / duration.Seconds()
	return speedMBps, nil
}
