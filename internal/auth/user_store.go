package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TechBuilder-360/Rental-API/ent"
	"github.com/TechBuilder-360/Rental-API/ent/user"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrAccountExists = errors.New("an account with this email or phone number already exists")
)

// NewUser is the data required to create a user.
type NewUser struct {
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	PhoneNumber  *string
	DOB          *time.Time
}

// UserStore is the persistence the auth service needs.
type UserStore interface {
	Create(ctx context.Context, u NewUser) (*ent.User, error)
	GetByEmail(ctx context.Context, email string) (*ent.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ent.User, error)
	MarkEmailVerified(ctx context.Context, id uuid.UUID) error
}

// EntUserStore implements UserStore with Ent.
type EntUserStore struct{ db *ent.Client }

func NewEntUserStore(db *ent.Client) *EntUserStore { return &EntUserStore{db: db} }

func (s *EntUserStore) Create(ctx context.Context, u NewUser) (*ent.User, error) {
	created, err := s.db.User.Create().
		SetEmail(u.Email).
		SetPassword(u.PasswordHash).
		SetFirstName(u.FirstName).
		SetLastName(u.LastName).
		SetNillablePhoneNumber(u.PhoneNumber).
		SetNillableDob(u.DOB).
		Save(ctx)
	if ent.IsConstraintError(err) {
		return nil, ErrAccountExists
	}
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

func (s *EntUserStore) GetByEmail(ctx context.Context, email string) (*ent.User, error) {
	u, err := s.db.User.Query().Where(user.EmailEQ(email)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (s *EntUserStore) GetByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	u, err := s.db.User.Get(ctx, id)
	if ent.IsNotFound(err) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (s *EntUserStore) MarkEmailVerified(ctx context.Context, id uuid.UUID) error {
	if err := s.db.User.UpdateOneID(id).SetEmailVerified(true).Exec(ctx); err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	return nil
}
