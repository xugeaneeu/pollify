package users

import (
	"context"
	"errors"
	"net/mail"
	"strings"
)

type Service struct {
	repo   UserRepository
	hasher PasswordHasher
	issuer TokenIssuer
	clock  Clock
}

type RegisterInput struct {
	Email       string
	Password    string
	DisplayName string
}

type LoginInput struct {
	Email    string
	Password string
}

func NewService(repo UserRepository, hasher PasswordHasher, issuer TokenIssuer, clock Clock) (*Service, error) {
	if repo == nil {
		return nil, ErrRepositoryUnbound
	}
	if hasher == nil {
		return nil, ErrHasherUnbound
	}
	if issuer == nil {
		return nil, ErrTokenIssuerUnbound
	}
	if clock == nil {
		return nil, ErrClockUnbound
	}

	return &Service{repo: repo, hasher: hasher, issuer: issuer, clock: clock}, nil
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthSession, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return AuthSession{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return AuthSession{}, err
	}

	displayName := strings.TrimSpace(input.DisplayName)

	_, err = s.repo.GetByEmail(ctx, email)
	if err == nil {
		return AuthSession{}, ErrEmailAlreadyExists
	}
	if !errors.Is(err, ErrUserNotFound) {
		return AuthSession{}, err
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return AuthSession{}, err
	}

	user, err := s.repo.Create(ctx, CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
		DisplayName:  displayName,
		Role:         RoleUser,
		CreatedAt:    s.clock.Now(),
	})
	if err != nil {
		return AuthSession{}, err
	}

	tokens, err := s.issuer.IssueTokens(ctx, user)
	if err != nil {
		return AuthSession{}, err
	}

	return AuthSession{User: user, Tokens: tokens}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthSession, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return AuthSession{}, err
	}
	if strings.TrimSpace(input.Password) == "" {
		return AuthSession{}, ValidationError{Field: "password", Message: "must not be empty"}
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return AuthSession{}, ErrInvalidCredentials
		}
		return AuthSession{}, err
	}

	if err := s.hasher.Verify(input.Password, user.PasswordHash); err != nil {
		return AuthSession{}, ErrInvalidCredentials
	}

	tokens, err := s.issuer.IssueTokens(ctx, user)
	if err != nil {
		return AuthSession{}, err
	}

	return AuthSession{User: user, Tokens: tokens}, nil
}

func (s *Service) CurrentUser(ctx context.Context, actor AuthenticatedUser) (User, error) {
	if strings.TrimSpace(actor.UserID) == "" {
		return User{}, ErrUnauthorized
	}

	return s.repo.GetByID(ctx, actor.UserID)
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" {
		return "", ValidationError{Field: "email", Message: "must not be empty"}
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", ValidationError{Field: "email", Message: "must be a valid email address"}
	}

	return email, nil
}

func validatePassword(password string) error {
	length := len(password)
	if length < 8 {
		return ValidationError{Field: "password", Message: "must be at least 8 characters long"}
	}
	if length > 128 {
		return ValidationError{Field: "password", Message: "must be at most 128 characters long"}
	}

	return nil
}
