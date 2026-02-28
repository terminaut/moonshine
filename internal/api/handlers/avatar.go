package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
)

type AvatarHandler struct {
	avatarService *services.AvatarService
	fightChecker  *FightChecker
}

func NewAvatarHandler(avatarService *services.AvatarService, fightChecker *FightChecker) *AvatarHandler {
	return &AvatarHandler{
		avatarService: avatarService,
		fightChecker:  fightChecker,
	}
}

func (h *AvatarHandler) GetAllAvatars(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	avatars, err := h.avatarService.GetAllAvatars(c.Request().Context())
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.AvatarsFromDomain(avatars))
}
