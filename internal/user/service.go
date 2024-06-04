package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/ncostamagna/axul_auth/auth"
	"github.com/ncostamagna/go-app-users-lab/internal/domain"
	"github.com/ncostamagna/go-app-users-lab/pkg/twofa"
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
		Login2FA(ctx context.Context, user *domain.User, token string) (*domain.Login, error)
		GetUserByToken(ctx context.Context, token string, checkAuthorized bool) (*domain.User, error)
		Create2FA(ctx context.Context, user *domain.User) error
		Get(ctx context.Context, id string) (*domain.User, error)
		GetAll(ctx context.Context, filters Filters, offset, limit int) ([]domain.User, error)
		Delete(ctx context.Context, id string) error
		Update(ctx context.Context, id string, firstName, lastName, email, phone, twoFStatus, twoFCode *string, twoFActive *bool) error
		Count(ctx context.Context, filters Filters) (int, error)
	}
	service struct {
		log         *log.Logger
		auth        auth.Auth
		twoFaClient twofa.TwoFA
		repo        Repository
	}
)

func NewService(log *log.Logger, auth auth.Auth, twoFaClient twofa.TwoFA, repo Repository) Service {
	return &service{
		log:         log,
		auth:        auth,
		twoFaClient: twoFaClient,
		repo:        repo,
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
		TwoFactor: users[0].TwoFActive && users[0].TwoFStatus == string(twofa.APPROVED),
	}

	var errAuth error
	if l.TwoFactor {
		l.TwoFactorHash, errAuth = s.auth.Create(users[0].ID, users[0].Username, "", false, 60)
	} else {
		l.Token, errAuth = s.auth.Create(users[0].ID, users[0].Username, "", true, 600)
	}

	if errAuth != nil {
		return nil, errAuth
	}

	return l, nil
}

func (s service) Login2FA(ctx context.Context, user *domain.User, code string) (*domain.Login, error) {

	if code == "" {
		return nil, ErrCodeRequired
	}

	if user.TwoFStatus == string(twofa.PENDING) {

		if err := s.twoFaClient.Verify(user.ID, code, user.TwoFCode); err != nil {
			return nil, err
		}

		user.TwoFStatus = string(twofa.APPROVED)
		user.TwoFActive = true

		if err := s.Update(ctx, user.ID, nil, nil, nil, nil, &user.TwoFStatus, &user.TwoFCode, &user.TwoFActive); err != nil {
			return nil, err
		}

	} else {

		if err := s.twoFaClient.Check(user.ID, code, user.TwoFCode); err != nil {
			return nil, err
		}

	}

	token, err := s.auth.Create(user.ID, user.Username, "", true, 3000)
	if err != nil {
		return nil, err
	}

	return &domain.Login{
		Status:    "ok",
		TwoFactor: user.TwoFActive,
		Token:     token,
	}, nil
}

func (s service) GetUserByToken(ctx context.Context, token string, checkAuthorized bool) (*domain.User, error) {
	v, err := s.auth.Check(token)
	if err != nil {
		return nil, err
	}
	if v.ID == "" {
		return nil, errors.New("invalid user information")
	}
	if checkAuthorized && !v.Authorized {
		return nil, errors.New("Unauthorized user")
	}

	user, err := s.Get(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s service) Create2FA(ctx context.Context, user *domain.User) error {

	if user.TwoFStatus == string(twofa.APPROVED) {
		return fmt.Errorf("the 2FA status is %s", user.TwoFStatus)
	}
	resp, err := s.twoFaClient.Create(user.ID)
	if err != nil {
		return err
	}

	user.TwoFCode = resp.Hash
	user.TwoFActive = true
	user.TwoFStatus = "pending"

	if err := s.Update(ctx, user.ID, nil, nil, nil, nil, &user.TwoFStatus, &user.TwoFCode, &user.TwoFActive); err != nil {
		return err
	}

	if err := s.twoFaClient.GenerateQR(user.ID, resp.Url); err != nil {
		return err
	}
	return nil
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

func (s service) Update(ctx context.Context, id string, firstName, lastName, email, phone, twoFStatus, twoFCode *string, twoFActive *bool) error {
	return s.repo.Update(ctx, id, firstName, lastName, email, phone, twoFStatus, twoFCode, twoFActive)
}

func (s service) Count(ctx context.Context, filters Filters) (int, error) {
	return s.repo.Count(ctx, filters)
}
