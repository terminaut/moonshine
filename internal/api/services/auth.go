package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
	"moonshine/internal/util"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInternalError      = errors.New("internal server error")
)

type SignUpInput struct {
	Username string `valid:"required,length(3|20)"`
	Email    string `valid:"required,email"`
	Password string `valid:"required,length(3|20)"`
}

type SignInInput struct {
	Username string `valid:"required,length(3|20)"`
	Password string `valid:"required,length(3|20)"`
}

type AuthService struct {
	userRepo     *repository.UserRepository
	avatarRepo   *repository.AvatarRepository
	locationRepo *repository.LocationRepository
	jwtKey       string
}

func NewAuthService(userRepo *repository.UserRepository, avatarRepo *repository.AvatarRepository, locationRepo *repository.LocationRepository, jwtKey string) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		avatarRepo:   avatarRepo,
		locationRepo: locationRepo,
		jwtKey:       jwtKey,
	}
}

func (s *AuthService) SignUp(ctx context.Context, input SignUpInput) (*domain.User, string, error) {
	if err := s.validateSignUpInput(input); err != nil {
		return nil, "", err
	}

	hashedPassword, err := util.HashPassword(input.Password)
	if err != nil {
		return nil, "", ErrInternalError
	}

	location, err := s.locationRepo.FindStartLocation()
	if err != nil {
		return nil, "", ErrInternalError
	}

	var avatarID *uuid.UUID
	avatar, err := s.avatarRepo.FindFirst()
	if err == nil && avatar != nil {
		avatarID = &avatar.ID
	}

	user := &domain.User{
		Username:   input.Username,
		Name:       input.Username,
		Email:      input.Email,
		Password:   hashedPassword,
		Attack:     1,
		Defense:    1,
		Hp:         20,
		CurrentHp:  20,
		Level:      1,
		Gold:       100,
		Exp:        0,
		FreeStats:  15,
		LocationID: location.ID,
		AvatarID:   avatarID,
	}

	if err := s.userRepo.Create(user); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			return nil, "", ErrUserAlreadyExists
		}
		return nil, "", ErrInternalError
	}

	token, err := s.generateJWTToken(user.ID)
	if err != nil {
		return nil, "", ErrInternalError
	}

	return user, token, nil
}

func (s *AuthService) SignIn(ctx context.Context, input SignInInput) (*domain.User, string, error) {
	if err := s.validateSignInInput(input); err != nil {
		return nil, "", err
	}

	user, err := s.userRepo.FindByUsername(input.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) || errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", ErrInternalError
	}

	if len(user.Password) == 0 {
		return nil, "", ErrInternalError
	}

	if err := util.CheckPassword(user.Password, input.Password); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.generateJWTToken(user.ID)
	if err != nil {
		return nil, "", ErrInternalError
	}

	return user, token, nil
}

func (s *AuthService) validateSignUpInput(input SignUpInput) error {
	type signUpValidator SignUpInput

	v := signUpValidator(input)

	if _, err := govalidator.ValidateStruct(v); err != nil {
		return ErrInvalidInput
	}
	return nil
}

func (s *AuthService) validateSignInInput(input SignInInput) error {
	type signInValidator SignInInput

	v := signInValidator(input)

	if _, err := govalidator.ValidateStruct(v); err != nil {
		return ErrInvalidInput
	}
	return nil
}

func (s *AuthService) generateJWTToken(id uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"id":  id.String(),
		"exp": time.Now().Add(72 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtKey))
}
