package handlers

import (
	"moonshine/internal/api/middleware"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/services"
	"moonshine/internal/repository"
)

type BotHandler struct {
	botService *services.BotService
	userRepo   *repository.UserRepository
}

type BotResponse struct {
	Bots []*dto.Bot `json:"bots"`
}

func NewBotHandler(db *sqlx.DB) *BotHandler {
	botService := services.NewBotService(db)
	userRepo := repository.NewUserRepository(db)

	return &BotHandler{
		botService: botService,
		userRepo:   userRepo,
	}
}

func (h *BotHandler) GetBots(c echo.Context) error {
	locationSlug := c.Param("location_slug")
	if locationSlug == "" {
		return ErrBadRequest(c, "location slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := checkNotInFight(c, h.userRepo, userID); err != nil {
		return err
	}

	bots, err := h.botService.GetBotsByLocationSlug(locationSlug)
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, &BotResponse{
		Bots: dto.BotsFromDomain(bots),
	})
}

func (h *BotHandler) Attack(c echo.Context) error {
	botSlug := c.Param("slug")
	if botSlug == "" {
		return ErrBadRequest(c, "bot slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	_, err = h.botService.Attack(c.Request().Context(), botSlug, userID)
	if err != nil {
		if err == repository.ErrBotNotFound {
			return ErrNotFound(c, "bot not found")
		}
		if err == repository.ErrUserNotFound {
			return ErrNotFound(c, "user not found")
		}
		return ErrBadRequest(c, err.Error())
	}

	return SuccessResponse(c, "attack initiated")
}
