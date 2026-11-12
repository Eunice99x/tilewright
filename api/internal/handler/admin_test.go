package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"saas-starter/api/internal/service"
	"saas-starter/api/internal/store"
)

const adminTestID = "00000000-0000-4000-8000-000000000001"

type adminStore struct{ service.Store }

func sessionHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (*adminStore) UserBySession(_ context.Context, hash string) (store.User, error) {
	if hash == sessionHash("admin") {
		return store.User{ID: "a", IsAdmin: true}, nil
	}
	if hash == sessionHash("customer") {
		return store.User{ID: "c"}, nil
	}
	return store.User{}, store.ErrNotFound
}

func (*adminStore) AdminUsers(context.Context, store.AdminUserFilter, int, int) ([]store.AdminUser, int, error) {
	return []store.AdminUser{{ID: adminTestID, Email: "x@example.com", SignIn: []string{"password"}}}, 1, nil
}

func (*adminStore) AdminUserByID(_ context.Context, id string) (store.AdminUserDetail, error) {
	if id != adminTestID {
		return store.AdminUserDetail{}, store.ErrNotFound
	}
	return store.AdminUserDetail{AdminUser: store.AdminUser{ID: id, Email: "x@example.com"}, PaddleCustomerID: "ctm_1"}, nil
}

func TestAdminUserRoutes(t *testing.T) {
	srv := httptest.NewServer(New(service.New(&adminStore{}, service.Config{})).Routes())
	defer srv.Close()

	get := func(path, session string) (int, string, http.Header) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if session != "" {
			req.Header.Set("Authorization", "Session "+session)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(body), res.Header
	}

	const customerBody = "{\"error\":\"not found\"}\n"
	for _, path := range []string{"/api/admin/users", "/api/admin/users?plan=gold", "/api/admin/users/" + adminTestID, "/api/admin/users/not-a-uuid"} {
		if code, body, _ := get(path, ""); code != http.StatusUnauthorized {
			t.Errorf("%s with no session: %d %s", path, code, body)
		}
		if code, body, _ := get(path, "customer"); code != http.StatusNotFound || body != customerBody {
			t.Errorf("%s as a customer: %d %s", path, code, body)
		}
	}

	for path, want := range map[string]int{
		"/api/admin/users": http.StatusOK,
		"/api/admin/users?q=%25&plan=free&confirmed=true":       http.StatusOK,
		"/api/admin/users?offset=nope":                          http.StatusOK,
		"/api/admin/users?offset=99999999999999999999":          http.StatusOK,
		"/api/admin/users?plan=gold":                            http.StatusBadRequest,
		"/api/admin/users?confirmed=maybe":                      http.StatusBadRequest,
		"/api/admin/users/" + adminTestID:                       http.StatusOK,
		"/api/admin/users/not-a-uuid":                           http.StatusNotFound,
		"/api/admin/users/00000000-0000-4000-8000-00000000":     http.StatusNotFound,
		"/api/admin/users/00000000-0000-4000-8000-999999999999": http.StatusNotFound,
	} {
		code, body, head := get(path, "admin")
		if code != want {
			t.Errorf("%s: %d, want %d: %s", path, code, want, body)
		}
		if want == http.StatusOK {
			if head.Get("Cache-Control") != "no-store" {
				t.Errorf("%s: Cache-Control %q", path, head.Get("Cache-Control"))
			}
			if strings.Contains(body, `"offset":99999`) {
				t.Errorf("%s: a number too big to parse should mean the first page: %s", path, body)
			}
		}
	}

	_, body, _ := get("/api/admin/users/"+adminTestID, "admin")
	for _, secret := range []string{"password_hash", "token"} {
		if strings.Contains(body, secret) {
			t.Errorf("detail holds %q: %s", secret, body)
		}
	}
	for _, shown := range []string{`"paddle_customer_id":"ctm_1"`, `"sign_in"`} {
		if !strings.Contains(body, shown) {
			t.Errorf("detail lost %q: %s", shown, body)
		}
	}
}
