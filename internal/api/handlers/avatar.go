package handlers

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/repository"
)

type AvatarHandler struct {
	avatarService *services.AvatarService
	userRepo      *repository.UserRepository
}

func NewAvatarHandler(db *sqlx.DB) *AvatarHandler {
	avatarRepo := repository.NewAvatarRepository(db)
	avatarService := services.NewAvatarService(avatarRepo)
	userRepo := repository.NewUserRepository(db)

	return &AvatarHandler{
		avatarService: avatarService,
		userRepo:      userRepo,
	}
}

func (h *AvatarHandler) GetAllAvatars(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	avatars, err := h.avatarService.GetAllAvatars(c.Request().Context())
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.AvatarsFromDomain(avatars))
}
