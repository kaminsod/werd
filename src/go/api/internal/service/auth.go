package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAuthUserNotFound   = errors.New("user not found")
)

const (
	bcryptCost  = 12
	tokenExpiry = 72 * time.Hour
)

type AuthUser struct {
	ID    string
	Email string
	Name  string
}

type Auth struct {
	q         *storage.Queries
	jwtSecret []byte
}

func NewAuth(q *storage.Queries, jwtSecret string) *Auth {
	return &Auth{
		q:         q,
		jwtSecret: []byte(jwtSecret),
	}
}

// Login verifies credentials and returns a JWT token + user info.
func (s *Auth) Login(ctx context.Context, email, password string) (string, *AuthUser, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.createToken(user.ID.String())
	if err != nil {
		return "", nil, fmt.Errorf("creating token: %w", err)
	}

	return token, &AuthUser{
		ID:    user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
	}, nil
}

// ValidateToken parses and validates a JWT, returning the user ID.
func (s *Auth) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return "", errors.New("missing subject claim")
	}

	return sub, nil
}

// GetUser fetches a user by ID.
func (s *Auth) GetUser(ctx context.Context, id string) (*AuthUser, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrAuthUserNotFound
	}

	user, err := s.q.GetUserByID(ctx, uid)
	if err != nil {
		return nil, ErrAuthUserNotFound
	}

	return &AuthUser{
		ID:    user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
	}, nil
}

// ChangePassword verifies the current password and sets a new one.
func (s *Auth) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := s.q.GetUserByID(ctx, uid)
	if err != nil {
		return fmt.Errorf("fetching user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	return s.q.UpdateUserPassword(ctx, storage.UpdateUserPasswordParams{
		PasswordHash: string(hash),
		ID:           uid,
	})
}

// SeedAdmin creates the initial admin user if it doesn't already exist.
// If the user already exists, it updates the password hash to match the configured password.
func (s *Auth) SeedAdmin(ctx context.Context, email, password string) error {
	if len(password) < 8 {
		return errors.New("admin password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("hashing admin password: %w", err)
	}

	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		// User doesn't exist — create it.
		_, err = s.q.CreateUser(ctx, storage.CreateUserParams{
			Email:        email,
			PasswordHash: string(hash),
			Name:         "Admin",
		})
		if err != nil {
			return fmt.Errorf("creating admin user: %w", err)
		}
		log.Printf("admin user %s created", email)
		return nil
	}

	// User exists — update password hash to match configured password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		if err := s.q.UpdateUserPassword(ctx, storage.UpdateUserPasswordParams{
			PasswordHash: string(hash),
			ID:           user.ID,
		}); err != nil {
			return fmt.Errorf("updating admin password: %w", err)
		}
		log.Printf("admin user %s password updated", email)
	} else {
		log.Printf("admin user %s already exists, password unchanged", email)
	}
	return nil
}

func (s *Auth) createToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenExpiry)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
