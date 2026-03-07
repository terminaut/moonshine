package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/middleware"
	"moonshine/internal/api/services"
	"moonshine/internal/repository"
)

type ResourceHandler struct {
	resourceService *services.ResourceService
	fightChecker    *FightChecker
}

type ResourceResponse struct {
	Resources []*dto.Resource `json:"resources"`
}

func NewResourceHandler(resourceService *services.ResourceService, fightChecker *FightChecker) *ResourceHandler {
	return &ResourceHandler{
		resourceService: resourceService,
		fightChecker:    fightChecker,
	}
}

func (h *ResourceHandler) GetResources(c echo.Context) error {
	locationSlug := c.Param("location_slug")
	if locationSlug == "" {
		return ErrBadRequest(c, "location slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	resources, err := h.resourceService.GetResourcesByLocationSlug(locationSlug)
	if err != nil {
		if errors.Is(err, repository.ErrLocationNotFound) {
			return ErrNotFound(c, "location not found")
		}
		return ErrInternalServerError(c)
	}

	return c.JSON(http.StatusOK, &ResourceResponse{
		Resources: dto.ResourcesFromDomain(resources),
	})
}

func (h *ResourceHandler) Gather(c echo.Context) error {
	resourceSlug := c.Param("slug")
	if resourceSlug == "" {
		return ErrBadRequest(c, "resource slug is required")
	}

	userID, err := middleware.GetUserIDFromContext(c.Request().Context())
	if err != nil {
		return ErrUnauthorized(c)
	}

	if err := h.fightChecker.CheckNotInFight(c, userID); err != nil {
		return err
	}

	err = h.resourceService.GatherResource(c.Request().Context(), userID, resourceSlug)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrResourceNotFound):
			return ErrNotFound(c, "resource not found")
		case errors.Is(err, repository.ErrUserNotFound):
			return ErrNotFound(c, "user not found")
		default:
			return ErrBadRequest(c, err.Error())
		}
	}

	return SuccessResponse(c, "resource gathered")
}
