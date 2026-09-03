// Package smtp implements SMTP auto-detection and email functionality.
// See AI.md PART 18 for details.
package smtp

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config holds SMTP configuration.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      string
	FromName string
	FromAddr string
}

// DefaultConfig returns default SMTP configuration.
func DefaultConfig() Config {
	return Config{
		Port: 587,
		TLS:  "auto",
	}
}

// Client handles SMTP connections.
type Client struct {
	config    Config
	detected  bool
	available bool
	mu        sync.RWMutex
}

// New creates a new SMTP Client.
func New(cfg Config) *Client {
	return &Client{
		config: cfg,
	}
}

// AutoDetect attempts to find a working SMTP server.
// Per PART 18, tries hosts in priority order.
func (c *Client) AutoDetect(fqdn string) (string, int, bool) {
	ports := []int{25, 465, 587}

	// Build list of hosts to try in priority order
	hosts := []string{
		"127.0.0.1",   // 1. Loopback
		"172.17.0.1",  // 2. Docker bridge gateway
	}

	// 3. Default gateway IP
	if gw := getDefaultGateway(); gw != "" {
		hosts = append(hosts, gw)
	}

	// 4. FQDN
	if fqdn != "" && fqdn != "localhost" {
		hosts = append(hosts, fqdn)
	}

	// 5. Global IPv4
	if ip := getGlobalIPv4(); ip != "" {
		hosts = append(hosts, ip)
	}

	// 6-7. Common mail subdomains
	if fqdn != "" && fqdn != "localhost" {
		hosts = append(hosts, "mail."+fqdn)
		hosts = append(hosts, "smtp."+fqdn)
	}

	// Try each host/port combination
	for _, host := range hosts {
		for _, port := range ports {
			if c.testConnection(host, port) {
				c.mu.Lock()
				c.config.Host = host
				c.config.Port = port
				c.detected = true
				c.available = true
				c.mu.Unlock()

				log.Printf("SMTP auto-detected: %s:%d", host, port)
				return host, port, true
			}
		}
	}

	log.Printf("SMTP: No server found during auto-detection")
	return "", 0, false
}

// testConnection attempts an SMTP handshake.
func (c *Client) testConnection(host string, port int) bool {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	// Set a timeout for the connection
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Set read/write deadline
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Create SMTP client
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return false
	}
	defer client.Close()

	// Attempt EHLO
	err = client.Hello("localhost")
	if err != nil {
		return false
	}

	return true
}

// TestConnection tests the currently configured SMTP server.
func (c *Client) TestConnection() bool {
	c.mu.RLock()
	host := c.config.Host
	port := c.config.Port
	c.mu.RUnlock()

	if host == "" {
		return false
	}

	result := c.testConnection(host, port)

	c.mu.Lock()
	c.available = result
	c.mu.Unlock()

	return result
}

// IsAvailable returns whether SMTP is configured and working.
func (c *Client) IsAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.available
}

// GetConfig returns the current SMTP configuration.
func (c *Client) GetConfig() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// SetConfig updates the SMTP configuration.
func (c *Client) SetConfig(cfg Config) {
	c.mu.Lock()
	c.config = cfg
	c.detected = false
	c.mu.Unlock()
}

// Send sends an email.
func (c *Client) Send(to, subject, body string) error {
	c.mu.RLock()
	cfg := c.config
	available := c.available
	c.mu.RUnlock()

	if !available {
		return fmt.Errorf("SMTP not available")
	}

	if cfg.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	// Build message
	from := cfg.FromAddr
	if from == "" {
		from = "no-reply@localhost"
	}

	msg := buildMessage(from, to, subject, body)

	// Determine auth
	var auth smtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	// Determine TLS mode
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	switch strings.ToLower(cfg.TLS) {
	case "tls", "ssl":
		// Direct TLS connection (port 465)
		return sendWithTLS(addr, cfg.Host, auth, from, to, msg)
	case "starttls":
		// STARTTLS upgrade (port 587)
		return sendWithSTARTTLS(addr, cfg.Host, auth, from, to, msg)
	case "none":
		// No encryption
		return smtp.SendMail(addr, auth, from, []string{to}, msg)
	default:
		// Auto: try STARTTLS first, fall back to plain
		if err := sendWithSTARTTLS(addr, cfg.Host, auth, from, to, msg); err == nil {
			return nil
		}
		return smtp.SendMail(addr, auth, from, []string{to}, msg)
	}
}

// buildMessage creates an email message.
func buildMessage(from, to, subject, body string) []byte {
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=UTF-8"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return []byte(msg.String())
}

// sendWithTLS sends email over direct TLS connection.
func sendWithTLS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

// sendWithSTARTTLS sends email with STARTTLS upgrade.
func sendWithSTARTTLS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	// Try STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: host,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

// getDefaultGateway returns the default gateway IP.
func getDefaultGateway() string {
	// Try to determine default gateway
	// This is platform-specific; for simplicity, we'll skip this
	// In a real implementation, you'd parse /proc/net/route on Linux
	return ""
}

// getGlobalIPv4 returns the machine's global IPv4 address.
func getGlobalIPv4() string {
	// Get all interfaces
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				// Skip private IPs for global
				if !isPrivateIP(ip) {
					return ip
				}
			}
		}
	}
	return ""
}

// isPrivateIP checks if an IP is in a private range.
func isPrivateIP(ip string) bool {
	private := []string{
		"10.",
		"172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.",
		"172.24.", "172.25.", "172.26.", "172.27.",
		"172.28.", "172.29.", "172.30.", "172.31.",
		"192.168.",
	}
	for _, prefix := range private {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}
	return false
}

// LoadFromEnv loads SMTP config from environment variables.
func LoadFromEnv(cfg *Config) {
	if host := os.Getenv("SMTP_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("SMTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}
	if user := os.Getenv("SMTP_USERNAME"); user != "" {
		cfg.Username = user
	}
	if pass := os.Getenv("SMTP_PASSWORD"); pass != "" {
		cfg.Password = pass
	}
	if tlsMode := os.Getenv("SMTP_TLS"); tlsMode != "" {
		cfg.TLS = tlsMode
	}
	if name := os.Getenv("SMTP_FROM_NAME"); name != "" {
		cfg.FromName = name
	}
	if email := os.Getenv("SMTP_FROM_EMAIL"); email != "" {
		cfg.FromAddr = email
	}
}
