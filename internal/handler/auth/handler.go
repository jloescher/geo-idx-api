package auth

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/auth/password"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// Handler provides token and user endpoints.
type Handler struct {
	cfg    config.Config
	db     *repository.DB
	tokens *repository.TokenRepo
	logger *slog.Logger
}

func NewHandler(cfg config.Config, db *repository.DB, logger *slog.Logger) *Handler {
	return &Handler{cfg: cfg, db: db, tokens: repository.NewTokenRepo(db), logger: logger}
}

type tokenRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Token(c *fiber.Ctx) error {
	var req tokenRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	pool, err := h.db.ReadPool(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var user struct {
		ID       int64
		Password string
	}
	err = pool.QueryRow(c.Context(), `SELECT id, password FROM users WHERE LOWER(email) = LOWER($1)`, req.Email).
		Scan(&user.ID, &user.Password)
	if err != nil || password.Verify(req.Password, user.Password) != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid credentials.")
	}
	if password.NeedsRehash(user.Password) {
		if upgraded, err := password.Hash(req.Password, password.DefaultParams); err == nil {
			_, _ = h.db.Pool.Exec(c.Context(), `UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2`, upgraded, user.ID)
		}
	}
	plain, err := h.tokens.Create(c.Context(), user.ID, "API Login", []string{"idx:full"})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"token": plain, "abilities": []string{"idx:full"}})
}

func (h *Handler) User(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthenticated.")
	}
	tok, user, err := h.tokens.FindByPlaintext(c.Context(), strings.TrimPrefix(auth, "Bearer "))
	if err != nil || tok == nil || user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthenticated.")
	}
	return c.JSON(fiber.Map{"id": user.ID, "name": user.Name, "email": user.Email})
}
