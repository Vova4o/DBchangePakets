package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Handler manages user HTTP requests.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler returns a new Handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes defines user HTTP routing rules on the given multiplexer.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /users", h.handleRegister)
	mux.HandleFunc("GET /users/{id}", h.handleGetByID)
}

type registerPayload struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var payload registerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.service.Register(r.Context(), RegisterRequest{
		Username: payload.Username,
		Email:    payload.Email,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrEmailTaken) {
			h.respondError(w, http.StatusConflict, err.Error())
			return
		}
		h.logger.Error("failed to register user", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.respondJSON(w, http.StatusCreated, u)
}

func (h *Handler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	u, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			h.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		h.logger.Error("failed to get user", "id", id, "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.respondJSON(w, http.StatusOK, u)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, msg string) {
	h.respondJSON(w, status, errorResponse{Error: msg})
}
