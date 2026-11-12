package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"saas-starter/api/internal/game"
	"saas-starter/api/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, accessLog, middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/paddle/webhook", h.paddleWebhook)

		r.Post("/auth/signup", h.signup)
		r.Post("/auth/guest", h.guest)
		r.Post("/auth/login", h.login)
		r.Post("/auth/oauth", h.oauth)
		r.Post("/auth/forgot", h.forgot)
		r.Post("/auth/reset", h.resetPassword)
		r.Group(func(r chi.Router) {
			r.Use(h.requireSession)
			r.Post("/auth/logout", h.logout)
			r.Post("/auth/verify", h.verifyEmail)
			r.Post("/auth/resend-verification", h.resendVerification)
			r.Get("/admin/users", h.adminUsers)
			r.Get("/admin/users/{id}", h.adminUser)
			r.Post("/billing/portal", h.billingPortal)
			r.Get("/me", h.me)

			r.Get("/games", h.listGames)
			r.Post("/games", h.startGame)
			r.Post("/games/versus", h.startVersus)
			r.Post("/games/join", h.joinGame)
			r.Get("/games/best", h.bestGames)
			r.Get("/games/{id}", h.getGame)
			r.Post("/games/{id}/play", h.playTiles)
			r.Post("/games/{id}/swap", h.swapTiles)
			r.Post("/games/{id}/pass", h.passTurn)
			r.Post("/games/{id}/end", h.endGame)
		})
	})
	return r
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) signup(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !readJSON(w, r, &in) {
		return
	}
	token, u, err := h.svc.Signup(r.Context(), clientIP(r), in.Email, in.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": u, "token": token})
}

func (h *Handler) guest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	token, u, err := h.svc.GuestSignup(r.Context(), clientIP(r), in.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": u, "token": token})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !readJSON(w, r, &in) {
		return
	}
	token, u, err := h.svc.Login(r.Context(), clientIP(r), in.Email, in.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "token": token})
}

func (h *Handler) oauth(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Provider    string `json:"provider"`
		Code        string `json:"code"`
		RedirectURI string `json:"redirect_uri"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	token, u, err := h.svc.OAuthLogin(r.Context(), clientIP(r), in.Provider, in.Code, in.RedirectURI)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "token": token})
}

func (h *Handler) forgot(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if err := h.svc.RequestPasswordReset(r.Context(), clientIP(r), in.Email); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if err := h.svc.ResetPassword(r.Context(), clientIP(r), in.Token, in.Password); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if err := h.svc.VerifyEmail(r.Context(), *userFrom(r), in.Token); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resendVerification(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ResendVerification(r.Context(), *userFrom(r)); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Logout(r.Context(), sessionToken(r)); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Me(*userFrom(r)))
}

func (h *Handler) adminUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, err := strconv.Atoi(q.Get("offset"))
	if err != nil {
		offset = 0
	}
	page, err := h.svc.AdminUsers(r.Context(), *userFrom(r), service.AdminUsersQuery{Search: q.Get("q"), Plan: q.Get("plan"), Confirmed: q.Get("confirmed"), Offset: offset})
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) adminUser(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.AdminUser(r.Context(), *userFrom(r), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) billingPortal(w http.ResponseWriter, r *http.Request) {
	links, err := h.svc.BillingPortal(r.Context(), *userFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, links)
}

func (h *Handler) paddleWebhook(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_body"})
		return
	}
	if err := h.svc.HandlePaddleWebhook(r.Context(), body, r.Header.Get("Paddle-Signature")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var notWord *game.NotWordError
	if errors.As(err, &notWord) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "words": notWord.Words})
		return
	}
	status := http.StatusInternalServerError
	switch {
	case errors.As(err, new(game.RuleError)):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrGameChanged), errors.Is(err, service.ErrGameFull):
		status = http.StatusConflict
	case errors.Is(err, service.ErrInvalid), errors.Is(err, service.ErrInvalidEmail), errors.Is(err, service.ErrWeakPassword), errors.Is(err, service.ErrInvalidToken):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrUnauthorized), errors.Is(err, service.ErrBadLogin), errors.Is(err, service.ErrBadSignature),
		errors.Is(err, service.ErrOAuthFailed):
		status = http.StatusUnauthorized
	case errors.Is(err, service.ErrOAuthNoVerifiedEmail):
		status = http.StatusForbidden
	case errors.Is(err, service.ErrNoSubscription), errors.Is(err, service.ErrOAuthPasswordAccount), errors.Is(err, service.ErrOAuthOtherIdentity), errors.Is(err, service.ErrEmailTaken):
		status = http.StatusConflict
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrTooManyAttempts):
		status = http.StatusTooManyRequests
	case errors.Is(err, service.ErrBillingDisabled), errors.Is(err, service.ErrOAuthDisabled), errors.Is(err, service.ErrEmailFailed):
		status = http.StatusServiceUnavailable
	}
	if status == http.StatusInternalServerError {
		slog.Error("request failed", "err", err)
		writeJSON(w, status, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
