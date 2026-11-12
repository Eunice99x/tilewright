package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"saas-starter/api/internal/game"
	"saas-starter/api/internal/service"
)

func (h *Handler) listGames(w http.ResponseWriter, r *http.Request) {
	games, err := h.svc.Games(r.Context(), *userFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"games": games})
}

func (h *Handler) startGame(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.StartGame(r.Context(), *userFrom(r))
	writeGame(w, g, err)
}

func (h *Handler) startVersus(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.StartVersus(r.Context(), *userFrom(r))
	writeGame(w, g, err)
}

func (h *Handler) joinGame(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	g, err := h.svc.JoinGame(r.Context(), *userFrom(r), in.Code)
	writeGame(w, g, err)
}

func (h *Handler) getGame(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.Game(r.Context(), *userFrom(r), chi.URLParam(r, "id"))
	writeGame(w, g, err)
}

func (h *Handler) bestGames(w http.ResponseWriter, r *http.Request) {
	best, err := h.svc.BestGames(r.Context(), *userFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"games": best})
}

func (h *Handler) playTiles(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Tiles []game.Placement `json:"tiles"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	g, err := h.svc.PlayTiles(r.Context(), *userFrom(r), chi.URLParam(r, "id"), in.Tiles)
	writeGame(w, g, err)
}

func (h *Handler) swapTiles(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Letters string `json:"letters"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	g, err := h.svc.SwapTiles(r.Context(), *userFrom(r), chi.URLParam(r, "id"), in.Letters)
	writeGame(w, g, err)
}

func (h *Handler) passTurn(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.PassTurn(r.Context(), *userFrom(r), chi.URLParam(r, "id"))
	writeGame(w, g, err)
}

func (h *Handler) endGame(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.EndGame(r.Context(), *userFrom(r), chi.URLParam(r, "id"))
	writeGame(w, g, err)
}

func writeGame(w http.ResponseWriter, g service.GameView, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, g)
}
