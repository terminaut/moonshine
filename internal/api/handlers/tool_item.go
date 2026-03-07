package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/domain"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"
)

type ToolItemHandler struct {
	toolItemBuyService     *services.ToolItemBuyService
	toolItemSellService    *services.ToolItemSellService
	toolItemTakeOnService  *services.ToolItemTakeOnService
	toolItemTakeOffService *services.ToolItemTakeOffService
	toolItemRepo           *repository.ToolItemRepository
	userCache              r.Cache[domain.User]
	fightChecker           *FightChecker
}

func NewToolItemHandler(
	toolItemBuyService *services.ToolItemBuyService,
	toolItemSellService *services.ToolItemSellService,
	toolItemTakeOnService *services.ToolItemTakeOnService,
	toolItemTakeOffService *services.ToolItemTakeOffService,
	toolItemRepo *repository.ToolItemRepository,
	userCache r.Cache[domain.User],
	fightChecker *FightChecker,
) *ToolItemHandler {
	return &ToolItemHandler{
		toolItemBuyService:     toolItemBuyService,
		toolItemSellService:    toolItemSellService,
		toolItemTakeOnService:  toolItemTakeOnService,
		toolItemTakeOffService: toolItemTakeOffService,
		toolItemRepo:           toolItemRepo,
		userCache:              userCache,
		fightChecker:           fightChecker,
	}
}

func (h *ToolItemHandler) invalidateUserCache(ctx context.Context, userID string) {
	if h.userCache != nil {
		_ = h.userCache.Delete(ctx, userID)
	}
}

func (h *ToolItemHandler) GetToolItems(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	items, err := h.toolItemRepo.FindAll()
	if err != nil {
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, dto.ToolItemsFromDomain(items))
}

func (h *ToolItemHandler) BuyToolItem(c echo.Context) error {
	itemSlug := c.Param("slug")
	if itemSlug == "" {
		return ErrBadRequest(c, "item slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.toolItemBuyService.BuyToolItem(c.Request().Context(), userID, itemSlug)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrToolItemNotFound):
			return ErrNotFound(c, "tool item not found")
		case errors.Is(err, services.ErrInsufficientGold):
			return ErrBadRequest(c, "insufficient gold")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "tool item purchased successfully")
}

func (h *ToolItemHandler) SellToolItem(c echo.Context) error {
	itemSlug := c.Param("slug")
	if itemSlug == "" {
		return ErrBadRequest(c, "item slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.toolItemSellService.SellToolItem(c.Request().Context(), userID, itemSlug)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrItemNotOwned):
			return ErrBadRequest(c, "tool item not owned")
		case errors.Is(err, repository.ErrToolItemNotFound):
			return ErrNotFound(c, "tool item not found")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "tool item sold successfully")
}

func (h *ToolItemHandler) TakeOnToolItem(c echo.Context) error {
	itemSlug := c.Param("slug")
	if itemSlug == "" {
		return ErrBadRequest(c, "item slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	item, err := h.toolItemRepo.FindBySlug(itemSlug)
	if err != nil {
		return ErrNotFound(c, "tool item not found")
	}

	err = h.toolItemTakeOnService.TakeOnToolItem(c.Request().Context(), userID, item.ID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrItemNotInInventory):
			return ErrBadRequest(c, "item not in inventory")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "tool item equipped successfully")
}

func (h *ToolItemHandler) TakeOffToolItem(c echo.Context) error {
	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.toolItemTakeOffService.TakeOffToolItem(c.Request().Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNoItemEquipped):
			return ErrBadRequest(c, "no tool item equipped")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrInternalServerError(c)
		}
	}

	h.invalidateUserCache(c.Request().Context(), userID.String())
	return SuccessResponse(c, "tool item removed successfully")
}
