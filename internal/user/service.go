package user

import (
	"context"
	"errors"
	"github.com/ncostamagna/axul_auth/auth"
	"github.com/ncostamagna/go-app-users-lab/internal/domain"
	"golang.org/x/crypto/bcrypt"
	"log"
)

type (
	Filters struct {
		FirstName string
		LastName  string
		Username  string
	}

	Service interface {
		Create(ctx context.Context, firstName, lastName, email, phone, username, password string) (*domain.User, error)
		Login(ctx context.Context, username, password string) (*domain.Login, error)
		Get(ctx context.Context, id string) (*domain.User, error)
		GetAll(ctx context.Context, filters Filters, offset, limit int) ([]domain.User, error)
		Delete(ctx context.Context, id string) error
		Update(ctx context.Context, id string, firstName *string, lastName *string, email *string, phone *string) error
		Count(ctx context.Context, filters Filters) (int, error)
	}
	service struct {
		log  *log.Logger
		auth auth.Auth
		repo Repository
	}
)

func NewService(log *log.Logger, auth auth.Auth, repo Repository) Service {
	return &service{
		log:  log,
		auth: auth,
		repo: repo,
	}
}

func (s service) Create(ctx context.Context, firstName, lastName, email, phone, username, password string) (*domain.User, error) {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := domain.User{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Phone:     phone,
		Username:  username,
		Password:  string(hashedPassword),
	}

	if err := s.repo.Create(ctx, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s service) Login(ctx context.Context, username, password string) (*domain.Login, error) {
	users, err := s.repo.GetAll(ctx, Filters{Username: username}, 0, 1)
	if err != nil {
		return nil, err
	}

	if len(users) < 1 {
		return nil, errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(users[0].Password), []byte(password)); err != nil {
		return nil, err
	}

	l := &domain.Login{
		Status:    "ok",
		TwoFactor: users[0].TwoFActive,
	}

	var errAuth error
	if users[0].TwoFActive && users[0].TwoFStatus == "aproved" {
		l.TwoFactorHash, errAuth = s.auth.Create("", "", users[0].TwoFCode, 60)
	} else {
		l.Token, errAuth = s.auth.Create(users[0].ID, users[0].Username, "", 60)
	}

	if errAuth != nil {
		return nil, errAuth
	}

	return l, nil
}
func (s service) GetAll(ctx context.Context, filters Filters, offset, limit int) ([]domain.User, error) {

	users, err := s.repo.GetAll(ctx, filters, offset, limit)
	if err != nil {
		return nil, err
	}
	for i := range users {
		users[i].Password = ""
	}
	return users, nil
}

func (s service) Get(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

func (s service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s service) Update(ctx context.Context, id string, firstName *string, lastName *string, email *string, phone *string) error {
	return s.repo.Update(ctx, id, firstName, lastName, email, phone)
}

func (s service) Count(ctx context.Context, filters Filters) (int, error) {
	return s.repo.Count(ctx, filters)
}
