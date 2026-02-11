package handlers

import (
	"errors"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"moonshine/internal/api/dto"
	"moonshine/internal/api/services"
	"moonshine/internal/repository"
)

type AuthHandler struct {
	authService  *services.AuthService
	locationRepo *repository.LocationRepository
	userRepo     *repository.UserRepository
}

func NewAuthHandler(db *sqlx.DB, jwtKey string) *AuthHandler {
	userRepo := repository.NewUserRepository(db)
	avatarRepo := repository.NewAvatarRepository(db)
	locationRepo := repository.NewLocationRepository(db)
	authService := services.NewAuthService(userRepo, avatarRepo, locationRepo, jwtKey)

	return &AuthHandler{
		authService:  authService,
		locationRepo: locationRepo,
		userRepo:     userRepo,
	}
}

type SignUpRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type SignInRequest struct {
	Username string `json:"username" validate:"required" example:"admin"`
	Password string `json:"password" validate:"required" example:"password"`
}

type AuthResponse struct {
	Token string    `json:"token"`
	User  *dto.User `json:"user"`
}

func (h *AuthHandler) SignUp(c echo.Context) error {
	var req SignUpRequest
	if err := c.Bind(&req); err != nil {
		return ErrBadRequest(c, "invalid request")
	}

	if err := c.Validate(&req); err != nil {
		return ErrBadRequest(c, err.Error())
	}

	serviceInput := services.SignUpInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	user, token, err := h.authService.SignUp(c.Request().Context(), serviceInput)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrUserAlreadyExists):
			return ErrConflict(c, "user already exists")
		case errors.Is(err, services.ErrInvalidInput):
			return ErrBadRequest(c, "invalid input")
		default:
			return ErrInternalServerError(c)
		}
	}

	location := resolveUserLocation(user, h.locationRepo)
	inFight, _ := h.userRepo.InFight(user.ID)

	return c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User:  dto.UserFromDomain(user, location, nil, inFight),
	})
}

func (h *AuthHandler) SignIn(c echo.Context) error {
	var req SignInRequest
	if err := c.Bind(&req); err != nil {
		return ErrBadRequest(c, "invalid request")
	}

	if err := c.Validate(&req); err != nil {
		return ErrBadRequest(c, err.Error())
	}

	serviceInput := services.SignInInput{
		Username: req.Username,
		Password: req.Password,
	}

	user, token, err := h.authService.SignIn(c.Request().Context(), serviceInput)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			return ErrUnauthorizedWithMessage(c, "invalid credentials")
		case errors.Is(err, services.ErrInvalidInput):
			return ErrBadRequest(c, "invalid input")
		default:
			return ErrInternalServerError(c)
		}
	}

	location := resolveUserLocation(user, h.locationRepo)
	inFight, _ := h.userRepo.InFight(user.ID)

	return c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User:  dto.UserFromDomain(user, location, nil, inFight),
	})
}
