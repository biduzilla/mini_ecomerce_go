package user

import (
	"errors"
	"ms_auth/internal/core/domain/models"
	"ms_auth/internal/core/validator"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	ROLE_ADMIN  Role = "ROLE_ADMIN"
	ROLE_CLIENT Role = "ROLE_CLIENT"
)

type User struct {
	models.BaseModel
	ID        uuid.UUID
	Nome      string
	Roles     []Role
	Email     string
	Activated bool
	Senha     password
}

type UserDTO struct {
	ID      *uuid.UUID `json:"id"`
	Nome    *string    `json:"nome"`
	Email   *string    `json:"email"`
	Senha   *string    `json:"senha,omitempty"`
	Version *int       `json:"version"`
}

type password struct {
	Plaintext *string
	Hash      []byte
}

func (m *User) ToDTO() *UserDTO {
	return &UserDTO{
		ID:      &m.ID,
		Nome:    &m.Nome,
		Email:   &m.Email,
		Version: &m.Version,
	}
}

func (d UserDTO) ToModel() (*User, error) {
	var model User

	if d.ID != nil {
		model.ID = *d.ID
	}

	if d.Nome != nil {
		model.Nome = *d.Nome
	}

	if d.Email != nil {
		model.Email = *d.Email
	}

	if d.Version != nil {
		model.Version = *d.Version
	}

	if d.Senha != nil {
		err := model.Senha.Set(*d.Senha)
		if err != nil {
			return nil, err
		}
	}

	return &model, nil
}

func (u *User) Validate(v *validator.Validator) {
	v.Check(u.Nome != "", "nome", "must be provided")
	v.Check(len(u.Nome) >= 3, "nome", "must be at least 3 characters long")
	v.Check(len(u.Nome) <= 100, "nome", "must not be more than 100 characters long")
	v.Check(u.Email != "", "email", "must be provided")
	v.Check(validator.Matches(u.Email, validator.EmailRX), "email", "must be a valid email address")
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.Plaintext = &plaintextPassword
	p.Hash = hash
	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.Hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func (u *User) GetID() uuid.UUID {
	return u.ID
}

func (u *User) GetIsAtivo() bool {
	return u.Activated
}

func (u *User) GetUsername() string { return u.Email }

func (u *User) IsAnonymous() bool {
	return false
}

func (u *User) GetRoles() []string {
	roles := make([]string, len(u.Roles))

	for i, r := range u.Roles {
		roles[i] = string(r)
	}

	return roles
}
