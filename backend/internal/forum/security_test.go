package forum

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUnicodeBoundaries(t *testing.T) {
	for _, x := range []struct {
		s        string
		min, max int
		ok       bool
	}{{"Привет", 5, 6, true}, {"яяяя", 5, 120, false}, {"🦉🦉🦉🦉🦉", 5, 5, true}, {"", 1, 5000, false}} {
		if validText(x.s, x.min, x.max) != x.ok {
			t.Errorf("boundary failed for %q", x.s)
		}
	}
}
func TestPasswordHash(t *testing.T) {
	hash, err := HashPassword("correct long password")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("correct long password", hash) || VerifyPassword("wrong", hash) || VerifyPassword("x", "$argon2id$broken") {
		t.Fatal("password verification")
	}
	hash2, _ := HashPassword("correct long password")
	if hash == hash2 {
		t.Fatal("salt must be random")
	}
}
func TestLimiter(t *testing.T) {
	l := limiter{}
	now := time.Now()
	for i := 0; i < 3; i++ {
		if l.allow("a", 3, now) != 0 {
			t.Fatal("early rejection")
		}
	}
	if l.allow("a", 3, now) == 0 {
		t.Fatal("missing rate limit")
	}
	if l.allow("a", 3, now.Add(time.Minute)) != 0 {
		t.Fatal("window did not expire")
	}
	if l.allow("b", 3, now) != 0 {
		t.Fatal("independent key rejected")
	}
}
func TestTokenAndCSRF(t *testing.T) {
	a, b := token(), token()
	if len(a) != 43 || a == b || digest(a) == a {
		t.Fatal("token generation")
	}
	s := Server{cfg: Config{Secret: "one"}}
	x := s.csrf(a)
	if x == s.csrf(b) {
		t.Fatal("CSRF not session bound")
	}
	s.cfg.Secret = "two"
	if x == s.csrf(a) {
		t.Fatal("CSRF not secret bound")
	}
}

func TestCookieAttributes(t *testing.T) {
	for _, secure := range []bool{false, true} {
		s := Server{cfg: Config{SecureCookies: secure}}
		rec := httptest.NewRecorder()
		s.cookie(rec, "forum_session", "opaque", 30*86400)
		c := rec.Result().Cookies()[0]
		if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Secure != secure || c.Path != "/" || c.MaxAge != 30*86400 {
			t.Fatal("unsafe cookie configuration")
		}
	}
}
