package cache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const envRedisURL = "REDIS_URL"

type RedisService struct {
	url     string
	enabled bool
}

type RedisStatus struct {
	Enabled   bool   `json:"enabled"`
	Connected bool   `json:"connected"`
	URL       string `json:"url"`
	Error     string `json:"error,omitempty"`
}

func NewRedisServiceFromEnv() *RedisService {
	raw := strings.TrimSpace(os.Getenv(envRedisURL))
	return &RedisService{url: raw, enabled: raw != ""}
}

func (s *RedisService) Status(ctx context.Context) RedisStatus {
	status := RedisStatus{
		Enabled: s.enabled,
		URL:     redactRedisURL(s.url),
	}
	if !s.enabled {
		return status
	}
	if err := s.Ping(ctx); err != nil {
		status.Error = err.Error()
		return status
	}
	status.Connected = true
	return status
}

func (s *RedisService) Ping(ctx context.Context) error {
	if !s.enabled {
		return errors.New("redis disabled")
	}
	_, err := s.command(ctx, "PING")
	return err
}

func (s *RedisService) GetString(ctx context.Context, key string) (string, bool, error) {
	if !s.enabled {
		return "", false, nil
	}
	value, err := s.command(ctx, "GET", key)
	if errors.Is(err, errRedisNil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *RedisService) SetString(ctx context.Context, key string, value string, ttl time.Duration) error {
	if !s.enabled {
		return nil
	}
	if ttl <= 0 {
		_, err := s.command(ctx, "SET", key, value)
		return err
	}
	_, err := s.command(ctx, "SET", key, value, "EX", strconv.Itoa(int(ttl.Seconds())))
	return err
}

func (s *RedisService) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if !s.enabled {
		return false, nil
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	value, err := s.command(ctx, "SET", key, "1", "NX", "EX", strconv.Itoa(int(ttl.Seconds())))
	if errors.Is(err, errRedisNil) {
		return false, nil
	}
	return strings.EqualFold(value, "OK"), err
}

func (s *RedisService) ReleaseLock(ctx context.Context, key string) error {
	if !s.enabled {
		return nil
	}
	_, err := s.command(ctx, "DEL", key)
	return err
}

func (s *RedisService) SetCooldown(ctx context.Context, key string, ttl time.Duration) error {
	return s.SetString(ctx, "cooldown:"+key, "1", ttl)
}

func (s *RedisService) CooldownActive(ctx context.Context, key string) (bool, error) {
	_, ok, err := s.GetString(ctx, "cooldown:"+key)
	return ok, err
}

func (s *RedisService) command(ctx context.Context, args ...string) (string, error) {
	address, password, err := parseRedisURL(s.url)
	if err != nil {
		return "", err
	}

	timeout := 2 * time.Second
	deadline, ok := ctx.Deadline()
	if ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			timeout = time.Second
		}
	}

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	reader := bufio.NewReader(conn)
	if password != "" {
		if _, err := writeCommand(conn, "AUTH", password); err != nil {
			return "", err
		}
		if _, err := readRESP(reader); err != nil {
			return "", err
		}
	}

	if _, err := writeCommand(conn, args...); err != nil {
		return "", err
	}
	return readRESP(reader)
}

func parseRedisURL(raw string) (string, string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", err
	}
	if parsed.Scheme != "redis" {
		return "", "", fmt.Errorf("unsupported redis scheme: %s", parsed.Scheme)
	}
	host := parsed.Host
	if !strings.Contains(host, ":") {
		host += ":6379"
	}
	password := ""
	if parsed.User != nil {
		password, _ = parsed.User.Password()
	}
	return host, password, nil
}

func redactRedisURL(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "invalid"
	}
	if parsed.User != nil {
		parsed.User = url.UserPassword("redis", "redacted")
	}
	return parsed.String()
}

func writeCommand(conn net.Conn, args ...string) (int, error) {
	var builder strings.Builder
	builder.WriteString("*")
	builder.WriteString(strconv.Itoa(len(args)))
	builder.WriteString("\r\n")
	for _, arg := range args {
		builder.WriteString("$")
		builder.WriteString(strconv.Itoa(len(arg)))
		builder.WriteString("\r\n")
		builder.WriteString(arg)
		builder.WriteString("\r\n")
	}
	return conn.Write([]byte(builder.String()))
}

var errRedisNil = errors.New("redis nil")

func readRESP(reader *bufio.Reader) (string, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return "", err
	}
	switch prefix {
	case '+':
		line, err := reader.ReadString('\n')
		return strings.TrimSpace(line), err
	case '-':
		line, _ := reader.ReadString('\n')
		return "", errors.New(strings.TrimSpace(line))
	case ':':
		line, err := reader.ReadString('\n')
		return strings.TrimSpace(line), err
	case '$':
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		length, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			return "", err
		}
		if length < 0 {
			return "", errRedisNil
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return "", err
		}
		return string(buf[:length]), nil
	default:
		return "", fmt.Errorf("unsupported redis response prefix %q", prefix)
	}
}
