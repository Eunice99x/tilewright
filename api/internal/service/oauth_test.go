package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"saas-starter/api/internal/store"
)

type fakeProviders struct {
	mu        sync.Mutex
	responses map[string]string
	forms     map[string]url.Values
	bearers   map[string]string
}

func (f *fakeProviders) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := r.URL.String()
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		f.forms[u], _ = url.ParseQuery(string(b))
	}
	f.bearers[u] = r.Header.Get("Authorization")
	status, body := http.StatusOK, f.responses[u]
	if body == "" {
		status, body = http.StatusBadRequest, `{"error":"invalid_grant"}`
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}, Request: r}, nil
}

func oauthService(st Store, responses map[string]string) (*Service, *fakeProviders) {
	fp := &fakeProviders{responses: responses, forms: map[string]url.Values{}, bearers: map[string]string{}}
	return New(st, Config{OAuth: OAuth{
		GoogleID: "gid", GoogleSecret: "gsecret", GitHubID: "ghid", GitHubSecret: "ghsecret",
		Client: &http.Client{Transport: fp},
	}}), fp
}

func googleOK(email string) map[string]string {
	return map[string]string{
		googleTokenURL:    `{"access_token":"gtok"}`,
		googleUserInfoURL: `{"sub":"sub-` + email + `","email":"` + email + `","email_verified":true}`,
	}
}

func githubOK(emails string) map[string]string {
	return map[string]string{
		githubTokenURL: `{"access_token":"ghtok"}`,

		githubUserURL:   `{"id":7,"email":"public@victim.example"}`,
		githubEmailsURL: emails,
	}
}

const redirect = "https://app.example/auth/x/callback"

func TestOAuthGoogle(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, fp := oauthService(f, googleOK("Jane@Example.com"))

	token, u, err := s.OAuthLogin(ctx, testIP, "google", "thecode", redirect)
	if err != nil || token == "" {
		t.Fatalf("login: %v", err)
	}
	if u.Email != "jane@example.com" || u.Plan != "free" {
		t.Errorf("user = %+v", u)
	}
	if got, err := s.UserFromSession(ctx, token); err != nil || got.ID != u.ID {
		t.Errorf("session: %+v %v", got, err)
	}
	form := fp.forms[googleTokenURL]
	if form.Get("code") != "thecode" || form.Get("redirect_uri") != redirect || form.Get("client_secret") != "gsecret" || form.Get("grant_type") != "authorization_code" {
		t.Errorf("token form = %v", form)
	}
	if fp.bearers[googleUserInfoURL] != "Bearer gtok" {
		t.Errorf("userinfo auth = %q", fp.bearers[googleUserInfoURL])
	}

	_, again, err := s.OAuthLogin(ctx, testIP, "google", "thecode", redirect)
	if err != nil || again.ID != u.ID {
		t.Errorf("second login: %+v %v", again, err)
	}
}

func TestOAuthGoogleUnverifiedRefused(t *testing.T) {
	f := newFakeStore()
	s, _ := oauthService(f, map[string]string{
		googleTokenURL:    `{"access_token":"gtok"}`,
		googleUserInfoURL: `{"sub":"1","email":"jane@example.com","email_verified":false}`,
	})
	if _, _, err := s.OAuthLogin(context.Background(), testIP, "google", "c", redirect); !errors.Is(err, ErrOAuthNoVerifiedEmail) {
		t.Errorf("got %v", err)
	}
	if len(f.users) != 0 {
		t.Error("created a user")
	}
}

func TestOAuthGitHub(t *testing.T) {
	f := newFakeStore()
	s, fp := oauthService(f, githubOK(`[
		{"email":"old@example.com","primary":false,"verified":true},
		{"email":"Dev@Example.com","primary":true,"verified":true}]`))
	_, u, err := s.OAuthLogin(context.Background(), testIP, "github", "thecode", redirect)
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "dev@example.com" {
		t.Errorf("email = %q", u.Email)
	}
	if form := fp.forms[githubTokenURL]; form.Get("code") != "thecode" || form.Get("client_secret") != "ghsecret" || form.Get("redirect_uri") != redirect {
		t.Errorf("token form = %v", form)
	}
	if fp.bearers[githubEmailsURL] != "Bearer ghtok" {
		t.Errorf("emails auth = %q", fp.bearers[githubEmailsURL])
	}
}

func TestOAuthGitHubNeedsPrimaryAndVerified(t *testing.T) {
	cases := map[string]string{
		"only unverified":                `[{"email":"dev@example.com","primary":true,"verified":false}]`,
		"verified but not primary":       `[{"email":"dev@example.com","primary":true,"verified":false},{"email":"x@example.com","primary":false,"verified":true}]`,
		"no emails, even with a profile": `[]`,
	}
	for name, emails := range cases {
		f := newFakeStore()
		s, _ := oauthService(f, githubOK(emails))
		if _, _, err := s.OAuthLogin(context.Background(), testIP, "github", "c", redirect); !errors.Is(err, ErrOAuthNoVerifiedEmail) {
			t.Errorf("%s: got %v", name, err)
		}
		if len(f.users) != 0 {
			t.Errorf("%s: created a user", name)
		}
	}
}

func TestOAuthBadCodeIsRefused(t *testing.T) {
	ctx := context.Background()

	s, _ := oauthService(newFakeStore(), map[string]string{githubTokenURL: `{"error":"bad_verification_code"}`})
	if _, _, err := s.OAuthLogin(ctx, testIP, "github", "c", redirect); err != ErrOAuthFailed {
		t.Errorf("github: got %v", err)
	}
	s, _ = oauthService(newFakeStore(), map[string]string{})
	if _, _, err := s.OAuthLogin(ctx, testIP, "google", "c", redirect); err != ErrOAuthFailed {
		t.Errorf("google: got %v", err)
	}
}

func TestOAuthRefusesConfirmedPasswordAccount(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, _ := oauthService(f, githubOK(`[{"email":"JANE@example.com","primary":true,"verified":true}]`))
	_, u, err := s.Signup(ctx, testIP, "jane@example.com", "longenough")
	if err != nil {
		t.Fatal(err)
	}
	f.setVerified(u.ID)
	if _, _, err := s.OAuthLogin(ctx, testIP, "github", "c", redirect); !errors.Is(err, ErrOAuthPasswordAccount) {
		t.Fatalf("got %v, want ErrOAuthPasswordAccount", err)
	}
	if len(f.users) != 1 {
		t.Errorf("%d users", len(f.users))
	}
}

func TestOAuthClaimsUnconfirmedPasswordAccount(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, _ := oauthService(f, googleOK("jane@example.com"))
	squatterToken, squat, err := s.Signup(ctx, testIP, "jane@example.com", "squatter-password")
	if err != nil {
		t.Fatal(err)
	}
	_, u, err := s.OAuthLogin(ctx, testIP, "google", "c", redirect)
	if err != nil || u.ID != squat.ID || !u.EmailVerified {
		t.Fatalf("got %+v, %v", u, err)
	}
	if _, err := s.UserFromSession(ctx, squatterToken); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("the squatter's session survived: %v", err)
	}
	if _, _, err := s.Login(ctx, testIP, "jane@example.com", "squatter-password"); !errors.Is(err, ErrBadLogin) {
		t.Errorf("the squatter's password still works: %v", err)
	}
	if _, again, err := s.OAuthLogin(ctx, testIP, "google", "c", redirect); err != nil || again.ID != u.ID {
		t.Errorf("returning: %+v, %v", again, err)
	}
}

func TestPasswordLoginRefusedForOAuthOnlyUser(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, _ := oauthService(f, googleOK("jane@example.com"))
	_, u, err := s.OAuthLogin(ctx, testIP, "google", "c", redirect)
	if err != nil {
		t.Fatal(err)
	}
	if f.users[u.ID].PasswordHash != "" {
		t.Fatalf("hash = %q", f.users[u.ID].PasswordHash)
	}
	for _, password := range []string{"", "anything-at-all"} {
		if _, _, err := s.Login(ctx, testIP, "jane@example.com", password); !errors.Is(err, ErrBadLogin) {
			t.Errorf("password %q: got %v", password, err)
		}
	}
	if _, _, err := s.Signup(ctx, testIP, "jane@example.com", "longenough"); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("signup over it: got %v", err)
	}
}

func TestOAuthSignupLimitOnlyForNewUsers(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, fp := oauthService(f, nil)
	login := func(email string) error {
		fp.mu.Lock()
		fp.responses = googleOK(email)
		fp.mu.Unlock()
		_, _, err := s.OAuthLogin(ctx, testIP, "google", "c", redirect)
		return err
	}
	for i := range signupIPLimit {
		if err := login(string(rune('a'+i)) + "@example.com"); err != nil {
			t.Fatalf("signup %d: %v", i, err)
		}
	}
	if err := login("over@example.com"); !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("over the limit: got %v", err)
	}
	if _, err := f.UserByEmail(ctx, "over@example.com"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("over the limit created a user: %v", err)
	}
	if err := login("a@example.com"); err != nil {
		t.Errorf("an existing user is not a signup: %v", err)
	}
}

func TestOAuthReturningIdentitySurvivesAnEmailChange(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, fp := oauthService(f, googleOK("jane@example.com"))
	_, first, err := s.OAuthLogin(ctx, testIP, "google", "c", redirect)
	if err != nil {
		t.Fatal(err)
	}
	fp.responses[googleUserInfoURL] = `{"sub":"sub-jane@example.com","email":"jane@new.example","email_verified":true}`
	if _, u, err := s.OAuthLogin(ctx, testIP, "google", "c", redirect); err != nil || u.ID != first.ID {
		t.Errorf("got %+v, %v; want %s", u, err, first.ID)
	}
}

func TestOAuthOtherIdentityWithSameEmailIsRefused(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	google, gp := oauthService(f, googleOK("bob@corp.example"))
	if _, _, err := google.OAuthLogin(ctx, testIP, "google", "c", redirect); err != nil {
		t.Fatal(err)
	}
	gp.responses[googleUserInfoURL] = `{"sub":"2","email":"bob@corp.example","email_verified":true}`
	if _, _, err := google.OAuthLogin(ctx, testIP, "google", "c", redirect); !errors.Is(err, ErrOAuthOtherIdentity) {
		t.Errorf("another Google account: %v", err)
	}
	github, _ := oauthService(f, githubOK(`[{"email":"bob@corp.example","primary":true,"verified":true}]`))
	if _, _, err := github.OAuthLogin(ctx, testIP, "github", "c", redirect); !errors.Is(err, ErrOAuthOtherIdentity) {
		t.Errorf("GitHub with the same address: %v", err)
	}
}

type losesCreateRace struct {
	*fakeStore
	identityMissed, emailMissed bool
}

func (r *losesCreateRace) UserByOAuthIdentity(ctx context.Context, p, sub string) (store.User, error) {
	if !r.identityMissed {
		r.identityMissed = true
		return store.User{}, store.ErrNotFound
	}
	return r.fakeStore.UserByOAuthIdentity(ctx, p, sub)
}

func (r *losesCreateRace) UserByEmail(ctx context.Context, email string) (store.User, error) {
	if !r.emailMissed {
		r.emailMissed = true
		return store.User{}, store.ErrNotFound
	}
	return r.fakeStore.UserByEmail(ctx, email)
}

func TestOAuthCreateRace(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	existing, err := f.CreateOAuthUser(ctx, "jane@example.com", "google", "sub-jane@example.com")
	if err != nil {
		t.Fatal(err)
	}

	s, _ := oauthService(&losesCreateRace{fakeStore: f}, googleOK("jane@example.com"))
	if _, u, err := s.OAuthLogin(ctx, testIP, "google", "c", redirect); err != nil || u.ID != existing.ID {
		t.Errorf("same identity: got %+v, %v", u, err)
	}

	s, _ = oauthService(&losesCreateRace{fakeStore: f}, githubOK(`[{"email":"jane@example.com","primary":true,"verified":true}]`))
	if _, _, err := s.OAuthLogin(ctx, testIP, "github", "c", redirect); !errors.Is(err, ErrOAuthOtherIdentity) {
		t.Errorf("other identity: %v", err)
	}
}

func TestOAuthRequestChecks(t *testing.T) {
	ctx := context.Background()
	c := &clock{time.Now()}
	off := newTestService(newFakeStore(), c, Paddle{})
	for _, provider := range []string{"google", "github"} {
		if _, _, err := off.OAuthLogin(ctx, testIP, provider, "c", redirect); !errors.Is(err, ErrOAuthDisabled) {
			t.Errorf("%s without env: got %v", provider, err)
		}
	}
	on, _ := oauthService(newFakeStore(), nil)
	for name, in := range map[string][3]string{
		"unknown provider": {"facebook", "c", redirect},
		"no code":          {"google", "", redirect},
		"no redirect uri":  {"github", "c", ""},
	} {
		if _, _, err := on.OAuthLogin(ctx, testIP, in[0], in[1], in[2]); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}
