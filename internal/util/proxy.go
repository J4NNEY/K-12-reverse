package util

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProxyHealth tracks the health state of a single proxy.
type ProxyHealth struct {
	URL       string
	Score     int       // starts at 10, max 20, min 0; decremented on failure, incremented on success
	LastError time.Time // when the last error occurred
	Cooldown  time.Duration // how long to wait before using again (doubles on failure)
}

// ProxyPool manages a pool of proxies with health tracking and round-robin rotation.
type ProxyPool struct {
	proxies []*ProxyHealth
	mu      sync.Mutex
	idx     int
}

// NewProxyPool creates a proxy pool from a list of proxy URLs.
func NewProxyPool(proxyList []string) *ProxyPool {
	pool := &ProxyPool{
		proxies: make([]*ProxyHealth, 0, len(proxyList)),
	}
	for _, p := range proxyList {
		p = strings.TrimSpace(p)
		if p != "" {
			pool.proxies = append(pool.proxies, &ProxyHealth{
				URL:      p,
				Score:    10,
				Cooldown: 5 * time.Second,
			})
		}
	}
	return pool
}

// Next returns the next healthy proxy in round-robin fashion.
// Returns empty string if no healthy proxy is available.
func (p *ProxyPool) Next() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.proxies) == 0 {
		return ""
	}

	startIdx := p.idx
	for i := 0; i < len(p.proxies); i++ {
		idx := (startIdx + i) % len(p.proxies)
		proxy := p.proxies[idx]

		if proxy.Score <= 0 {
			if time.Since(proxy.LastError) < proxy.Cooldown {
				continue
			}
			proxy.Score = 5
			proxy.Cooldown = 5 * time.Second
		}

		p.idx = (idx + 1) % len(p.proxies)
		return proxy.URL
	}

	return p.proxies[p.idx].URL
}

// MarkFailure marks a proxy as failed, decreasing its health score and setting cooldown.
func (p *ProxyPool) MarkFailure(proxyURL string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, proxy := range p.proxies {
		if proxy.URL == proxyURL {
			proxy.Score -= 3
			if proxy.Score < 0 {
				proxy.Score = 0
			}
			proxy.LastError = time.Now()
			proxy.Cooldown *= 2
			if proxy.Cooldown > 5*time.Minute {
				proxy.Cooldown = 5 * time.Minute
			}
			return
		}
	}
}

// MarkSuccess marks a proxy as successful, increasing its health score.
func (p *ProxyPool) MarkSuccess(proxyURL string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, proxy := range p.proxies {
		if proxy.URL == proxyURL {
			proxy.Score += 1
			if proxy.Score > 20 {
				proxy.Score = 20
			}
			proxy.Cooldown = 5 * time.Second
			return
		}
	}
}

// HealthyCount returns the number of proxies with positive score.
func (p *ProxyPool) HealthyCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	count := 0
	for _, proxy := range p.proxies {
		if proxy.Score > 0 {
			count++
		}
	}
	return count
}

// Size returns the total number of proxies in the pool.
func (p *ProxyPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.proxies)
}

// RandomProxy picks a random healthy proxy.
func (p *ProxyPool) RandomProxy() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	healthy := make([]string, 0)
	for _, proxy := range p.proxies {
		if proxy.Score > 0 {
			healthy = append(healthy, proxy.URL)
		}
	}
	if len(healthy) == 0 {
		return ""
	}
	return healthy[rand.Intn(len(healthy))]
}

// Stats returns a formatted string of proxy health status.
func (p *ProxyPool) Stats() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	var parts []string
	for _, proxy := range p.proxies {
		status := "🟢"
		if proxy.Score <= 0 {
			status = "🔴"
		} else if proxy.Score < 5 {
			status = "🟡"
		}
		parts = append(parts, fmt.Sprintf("%s %s (score:%d)", status, proxy.URL, proxy.Score))
	}
	return strings.Join(parts, "\n")
}

// FormatProxy parses various proxy string formats and normalizes them into protocol://username:password@host:port
// Supported formats:
// 1. Agreement://Host IP:Port:Username:Password (e.g. socks5://192.168.1.1:8080:user:pass)
// 2. Host IP:Port:Username:Password
// 3. Username:Password@Host IP:Port
// 4. Username:Password:Host IP:Port
// 5. Host IP:Port@Username:Password
func FormatProxy(rawProxy string) string {
	rawProxy = strings.TrimSpace(rawProxy)
	if rawProxy == "" {
		return ""
	}

	protocol := "http"
	parts := strings.SplitN(rawProxy, "://", 2)
	if len(parts) == 2 {
		protocol = parts[0]
		rawProxy = parts[1]
	}

	var user, pass, host, port string

	// Helper to check if a string looks like a port
	isPort := func(s string) bool {
		p, err := strconv.Atoi(s)
		return err == nil && p > 0 && p <= 65535
	}

	if strings.Contains(rawProxy, "@") {
		atParts := strings.SplitN(rawProxy, "@", 2)
		part1 := atParts[0]
		part2 := atParts[1]

		p1Parts := strings.SplitN(part1, ":", 2)
		if len(p1Parts) == 2 && isPort(p1Parts[1]) && (strings.Contains(p1Parts[0], ".") || p1Parts[0] == "localhost") {
			// Host:Port@Username:Password
			host = p1Parts[0]
			port = p1Parts[1]
			p2Parts := strings.SplitN(part2, ":", 2)
			if len(p2Parts) == 2 {
				user = p2Parts[0]
				pass = p2Parts[1]
			} else {
				user = part2
			}
		} else {
			// Username:Password@Host:Port (Standard)
			p1Parts := strings.SplitN(part1, ":", 2)
			if len(p1Parts) == 2 {
				user = p1Parts[0]
				pass = p1Parts[1]
			} else {
				user = part1
			}
			p2Parts := strings.SplitN(part2, ":", 2)
			if len(p2Parts) == 2 {
				host = p2Parts[0]
				port = p2Parts[1]
			} else {
				host = part2
			}
		}
	} else {
		colonParts := strings.Split(rawProxy, ":")
		if len(colonParts) == 4 {
			if isPort(colonParts[1]) && (strings.Contains(colonParts[0], ".") || colonParts[0] == "localhost") {
				// Host:Port:Username:Password
				host = colonParts[0]
				port = colonParts[1]
				user = colonParts[2]
				pass = colonParts[3]
			} else if isPort(colonParts[3]) && (strings.Contains(colonParts[2], ".") || colonParts[2] == "localhost") {
				// Username:Password:Host:Port
				user = colonParts[0]
				pass = colonParts[1]
				host = colonParts[2]
				port = colonParts[3]
			} else {
				// Fallback to Host:Port:Username:Password
				host = colonParts[0]
				port = colonParts[1]
				user = colonParts[2]
				pass = colonParts[3]
			}
		} else if len(colonParts) == 2 {
			// Host:Port
			host = colonParts[0]
			port = colonParts[1]
		} else {
			if !strings.Contains(rawProxy, "://") {
				return fmt.Sprintf("%s://%s", protocol, rawProxy)
			}
			return rawProxy
		}
	}

	if user != "" || pass != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%s", protocol, user, pass, host, port)
	}
	if host != "" && port != "" {
		return fmt.Sprintf("%s://%s:%s", protocol, host, port)
	}

	return fmt.Sprintf("%s://%s", protocol, rawProxy)
}
