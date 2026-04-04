// Redis cache adapter for NexGo.
// Implements the distributed cache interface using raw TCP connection to Redis.
// Zero external dependencies — speaks the Redis protocol (RESP) directly.
package cache

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr        string        // "localhost:6379"
	Password    string        // "" for no auth
	DB          int           // 0-15
	MaxIdle     int           // max idle connections
	DialTimeout time.Duration
	ReadTimeout time.Duration
	KeyPrefix   string        // prefix all keys, e.g. "nexgo:"
}

// DefaultRedisConfig returns sensible Redis defaults.
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:        "localhost:6379",
		DB:          0,
		MaxIdle:     10,
		DialTimeout: 5 * time.Second,
		ReadTimeout: 3 * time.Second,
		KeyPrefix:   "nexgo:",
	}
}

// RedisCache is a Redis-backed cache using raw RESP protocol.
type RedisCache struct {
	config RedisConfig
	pool   chan net.Conn
	mu     sync.Mutex
}

// NewRedis creates a Redis cache adapter.
func NewRedis(cfg RedisConfig) (*RedisCache, error) {
	if cfg.MaxIdle < 1 {
		cfg.MaxIdle = 10
	}

	rc := &RedisCache{
		config: cfg,
		pool:   make(chan net.Conn, cfg.MaxIdle),
	}

	// Test connection
	conn, err := rc.dial()
	if err != nil {
		return nil, fmt.Errorf("redis: connect failed: %w", err)
	}
	rc.putConn(conn)

	return rc, nil
}

// Get retrieves a cached value by key.
func (rc *RedisCache) Get(key string) ([]byte, error) {
	conn, err := rc.getConn()
	if err != nil {
		return nil, err
	}
	defer rc.putConn(conn)

	reply, err := rc.do(conn, "GET", rc.prefixed(key))
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, nil
	}
	return reply, nil
}

// Set stores a value with TTL.
func (rc *RedisCache) Set(key string, value []byte, ttl time.Duration) error {
	conn, err := rc.getConn()
	if err != nil {
		return err
	}
	defer rc.putConn(conn)

	if ttl > 0 {
		_, err = rc.do(conn, "SETEX", rc.prefixed(key), strconv.Itoa(int(ttl.Seconds())), string(value))
	} else {
		_, err = rc.do(conn, "SET", rc.prefixed(key), string(value))
	}
	return err
}

// Delete removes a key.
func (rc *RedisCache) Delete(key string) error {
	conn, err := rc.getConn()
	if err != nil {
		return err
	}
	defer rc.putConn(conn)

	_, err = rc.do(conn, "DEL", rc.prefixed(key))
	return err
}

// DeletePrefix removes all keys matching a prefix using SCAN.
func (rc *RedisCache) DeletePrefix(prefix string) error {
	conn, err := rc.getConn()
	if err != nil {
		return err
	}
	defer rc.putConn(conn)

	pattern := rc.prefixed(prefix) + "*"
	cursor := "0"

	for {
		// SCAN cursor MATCH pattern COUNT 100
		resp, err := rc.doMulti(conn, "SCAN", cursor, "MATCH", pattern, "COUNT", "100")
		if err != nil {
			return err
		}
		if len(resp) < 2 {
			break
		}

		cursor = resp[0]
		keys := strings.Split(resp[1], "\n")
		for _, k := range keys {
			k = strings.TrimSpace(k)
			if k != "" {
				rc.do(conn, "DEL", k)
			}
		}

		if cursor == "0" {
			break
		}
	}

	return nil
}

// Exists checks if a key exists.
func (rc *RedisCache) Exists(key string) (bool, error) {
	conn, err := rc.getConn()
	if err != nil {
		return false, err
	}
	defer rc.putConn(conn)

	reply, err := rc.do(conn, "EXISTS", rc.prefixed(key))
	if err != nil {
		return false, err
	}
	return string(reply) == "1", nil
}

// TTL returns the remaining time-to-live for a key.
func (rc *RedisCache) TTL(key string) (time.Duration, error) {
	conn, err := rc.getConn()
	if err != nil {
		return 0, err
	}
	defer rc.putConn(conn)

	reply, err := rc.do(conn, "TTL", rc.prefixed(key))
	if err != nil {
		return 0, err
	}
	secs, _ := strconv.Atoi(string(reply))
	if secs < 0 {
		return 0, nil
	}
	return time.Duration(secs) * time.Second, nil
}

// Incr atomically increments a key.
func (rc *RedisCache) Incr(key string) (int64, error) {
	conn, err := rc.getConn()
	if err != nil {
		return 0, err
	}
	defer rc.putConn(conn)

	reply, err := rc.do(conn, "INCR", rc.prefixed(key))
	if err != nil {
		return 0, err
	}
	n, _ := strconv.ParseInt(string(reply), 10, 64)
	return n, nil
}

// FlushPrefix removes all keys with the cache prefix.
func (rc *RedisCache) FlushPrefix() error {
	return rc.DeletePrefix("")
}

// Ping checks the Redis connection.
func (rc *RedisCache) Ping() error {
	conn, err := rc.getConn()
	if err != nil {
		return err
	}
	defer rc.putConn(conn)

	reply, err := rc.do(conn, "PING")
	if err != nil {
		return err
	}
	if string(reply) != "PONG" {
		return errors.New("redis: unexpected PING response")
	}
	return nil
}

// Close closes all pooled connections.
func (rc *RedisCache) Close() error {
	close(rc.pool)
	for conn := range rc.pool {
		conn.Close()
	}
	return nil
}

// --- RESP Protocol ---

func (rc *RedisCache) do(conn net.Conn, args ...string) ([]byte, error) {
	// Send command in RESP format
	cmd := formatRESP(args)
	conn.SetWriteDeadline(time.Now().Add(rc.config.ReadTimeout))
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return nil, err
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(rc.config.ReadTimeout))
	return readRESP(bufio.NewReader(conn))
}

func (rc *RedisCache) doMulti(conn net.Conn, args ...string) ([]string, error) {
	cmd := formatRESP(args)
	conn.SetWriteDeadline(time.Now().Add(rc.config.ReadTimeout))
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(rc.config.ReadTimeout))
	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")

	if line[0] == '*' {
		count, _ := strconv.Atoi(line[1:])
		results := make([]string, count)
		for i := 0; i < count; i++ {
			data, err := readRESP(reader)
			if err != nil {
				return results, err
			}
			results[i] = string(data)
		}
		return results, nil
	}

	return []string{line[1:]}, nil
}

func formatRESP(args []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	for _, arg := range args {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}
	return sb.String()
}

func readRESP(reader *bufio.Reader) ([]byte, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")

	if len(line) == 0 {
		return nil, errors.New("redis: empty response")
	}

	switch line[0] {
	case '+': // Simple string
		return []byte(line[1:]), nil
	case '-': // Error
		return nil, errors.New("redis: " + line[1:])
	case ':': // Integer
		return []byte(line[1:]), nil
	case '$': // Bulk string
		size, _ := strconv.Atoi(line[1:])
		if size < 0 {
			return nil, nil // nil bulk string
		}
		data := make([]byte, size+2) // +2 for \r\n
		_, err := reader.Read(data)
		if err != nil {
			return nil, err
		}
		return data[:size], nil
	case '*': // Array
		count, _ := strconv.Atoi(line[1:])
		if count < 0 {
			return nil, nil
		}
		var parts []string
		for i := 0; i < count; i++ {
			elem, err := readRESP(reader)
			if err != nil {
				return nil, err
			}
			parts = append(parts, string(elem))
		}
		return []byte(strings.Join(parts, "\n")), nil
	default:
		return []byte(line), nil
	}
}

// --- Connection Pool ---

func (rc *RedisCache) dial() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", rc.config.Addr, rc.config.DialTimeout)
	if err != nil {
		return nil, err
	}

	// AUTH if password set
	if rc.config.Password != "" {
		reply, err := rc.do(conn, "AUTH", rc.config.Password)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("redis AUTH: %w", err)
		}
		if string(reply) != "OK" {
			conn.Close()
			return nil, errors.New("redis: AUTH failed")
		}
	}

	// SELECT database
	if rc.config.DB > 0 {
		reply, err := rc.do(conn, "SELECT", strconv.Itoa(rc.config.DB))
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("redis SELECT: %w", err)
		}
		if string(reply) != "OK" {
			conn.Close()
			return nil, errors.New("redis: SELECT failed")
		}
	}

	return conn, nil
}

func (rc *RedisCache) getConn() (net.Conn, error) {
	select {
	case conn := <-rc.pool:
		if conn != nil {
			return conn, nil
		}
	default:
	}
	return rc.dial()
}

func (rc *RedisCache) putConn(conn net.Conn) {
	select {
	case rc.pool <- conn:
	default:
		conn.Close()
	}
}

func (rc *RedisCache) prefixed(key string) string {
	return rc.config.KeyPrefix + key
}

// --- Cache interface adapter ---

// RedisCacheAdapter wraps RedisCache to match the NexGo Cache interface.
type RedisCacheAdapter struct {
	rc         *RedisCache
	defaultTTL time.Duration
}

// NewRedisCacheAdapter creates a Cache-compatible Redis adapter.
func NewRedisCacheAdapter(rc *RedisCache, ttl time.Duration) *RedisCacheAdapter {
	return &RedisCacheAdapter{rc: rc, defaultTTL: ttl}
}

// GetCached retrieves a cached response.
func (a *RedisCacheAdapter) GetCached(key string) (int, []byte, bool) {
	data, err := a.rc.Get("resp:" + key)
	if err != nil || data == nil {
		return 0, nil, false
	}
	return 200, data, true
}

// SetCached stores a response.
func (a *RedisCacheAdapter) SetCached(key string, status int, body []byte) {
	a.rc.Set("resp:"+key, body, a.defaultTTL)
}

// DeleteCached removes a cached response.
func (a *RedisCacheAdapter) DeleteCached(key string) {
	a.rc.Delete("resp:" + key)
}

// ClearCached removes all cached responses.
func (a *RedisCacheAdapter) ClearCached() {
	a.rc.DeletePrefix("resp:")
}
