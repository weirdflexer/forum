package forum

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireRoleAndSessionBoundCSRF(t *testing.T) {
	app := &Server{cfg: Config{Secret: "test-secret"}}
	anon := identity{ID: "visitor", Raw: "visitor-token"}
	mod := identity{ID: "mod", Raw: "moderator-token", Role: "moderator"}
	admin := identity{ID: "admin", Raw: "admin-token", Role: "admin"}
	for _, tc := range []struct {
		name, method, role, csrf, code string
		anon, staff                    identity
		wantID                         string
	}{
		{name: "missing anonymous session", method: "GET", role: "anonymous", code: "forbidden"},
		{name: "missing staff session", method: "GET", role: "moderator", anon: anon, code: "forbidden"},
		{name: "moderator cannot administer", method: "GET", role: "admin", staff: mod, code: "forbidden"},
		{name: "admin can moderate", method: "GET", role: "moderator", staff: admin, wantID: admin.ID},
		{name: "read does not need CSRF", method: "GET", role: "anonymous", anon: anon, wantID: anon.ID},
		{name: "HEAD does not need CSRF", method: "HEAD", role: "moderator", staff: mod, wantID: mod.ID},
		{name: "write needs CSRF", method: "POST", role: "anonymous", anon: anon, code: "csrf"},
		{name: "anonymous write", method: "POST", role: "anonymous", anon: anon, csrf: app.csrf(anon.Raw), wantID: anon.ID},
		{name: "staff write", method: "POST", role: "moderator", staff: mod, csrf: app.csrf(mod.Raw), wantID: mod.ID},
		{name: "anonymous CSRF cannot authorize staff", method: "POST", role: "moderator", anon: anon, staff: mod, csrf: app.csrf(anon.Raw), code: "csrf"},
		{name: "staff CSRF cannot authorize anonymous", method: "POST", role: "anonymous", anon: anon, staff: mod, csrf: app.csrf(mod.Raw), code: "csrf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/", nil)
			r.Header.Set("X-CSRF-Token", tc.csrf)
			r = r.WithContext(context.WithValue(r.Context(), stateKey{}, &requestState{Anon: tc.anon, Staff: tc.staff}))
			got, err := app.require(r, tc.role)
			if tc.code == "" {
				if err != nil || got.ID != tc.wantID {
					t.Fatalf("require=%#v, %v; want ID %s", got, err, tc.wantID)
				}
				return
			}
			var api *apiError
			if !errors.As(err, &api) || api.Status != http.StatusForbidden || api.Code != tc.code {
				t.Fatalf("require error=%#v, want 403 %s", err, tc.code)
			}
		})
	}
}

func TestIdentifyIgnoresMalformedCookies(t *testing.T) {
	app := &Server{} // No database is needed for cookies that cannot be valid tokens.
	r := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	r.AddCookie(&http.Cookie{Name: "forum_session", Value: "short"})
	r.AddCookie(&http.Cookie{Name: "forum_staff", Value: "short"})
	st := &requestState{}
	if err := app.identify(r, st); err != nil || st.Anon.ID != "" || st.Staff.ID != "" {
		t.Fatalf("malformed cookies produced identities: %#v, %v", st, err)
	}
}
