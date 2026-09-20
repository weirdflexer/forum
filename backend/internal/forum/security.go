package forum

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/argon2"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var loginPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,40}$`)
var slugPattern = regexp.MustCompile(`^[a-z0-9-]{2,40}$`)

func token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func digest(s string) string { b := sha256.Sum256([]byte(s)); return hex.EncodeToString(b[:]) }
func (s *Server) csrf(raw string) string {
	h := hmac.New(sha256.New, []byte(s.cfg.Secret))
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}
func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=65536,t=3,p=2" {
		return false
	}
	salt, e1 := base64.RawStdEncoding.DecodeString(parts[4])
	want, e2 := base64.RawStdEncoding.DecodeString(parts[5])
	if e1 != nil || e2 != nil || len(salt) != 16 || len(want) != 32 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return subtle.ConstantTimeCompare(got, want) == 1
}
func (s *Server) cookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: s.cfg.SecureCookies, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}

type bucket struct {
	count int
	reset time.Time
}
type limiter struct {
	sync.Mutex
	items map[string]bucket
}

func (l *limiter) allow(key string, max int, now time.Time) int {
	l.Lock()
	defer l.Unlock()
	if l.items == nil {
		l.items = map[string]bucket{}
	}
	if len(l.items) > 1000 {
		for k, v := range l.items {
			if !now.Before(v.reset) {
				delete(l.items, k)
			}
		}
	}
	b := l.items[key]
	if !now.Before(b.reset) {
		b = bucket{reset: now.Add(time.Minute)}
	}
	if b.count >= max {
		return int(b.reset.Sub(now).Seconds()) + 1
	}
	b.count++
	l.items[key] = b
	return 0
}
func (s *Server) networkLimit(w http.ResponseWriter, r *http.Request, group string, max int) error {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	// Trust no forwarding headers. Reverse-proxy deployments share a conservative network budget.
	wait := s.limits.allow(group+":"+s.csrf(ip), max, time.Now())
	if wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(wait))
		return problem(429, "rate_limit", fmt.Sprintf("Слишком много действий. Подождите %d сек.", wait))
	}
	return nil
}
