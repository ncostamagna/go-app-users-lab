package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/ncostamagna/axul_auth/auth"
	"github.com/ncostamagna/go-app-users-lab/internal/domain"
	"github.com/skip2/go-qrcode"
	"github.com/twilio/twilio-go"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
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
		log          *log.Logger
		auth         auth.Auth
		twilioClient *twilio.RestClient
		repo         Repository
	}
)

func NewService(log *log.Logger, auth auth.Auth, twilioClient *twilio.RestClient, repo Repository) Service {
	return &service{
		log:          log,
		auth:         auth,
		twilioClient: twilioClient,
		repo:         repo,
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
	if users[0].TwoFActive && users[0].TwoFStatus == "approved" {
		l.TwoFactorHash, errAuth = s.auth.Create(users[0].ID, users[0].Username, users[0].TwoFCode, false, 60)
	} else {
		l.Token, errAuth = s.auth.Create(users[0].ID, users[0].Username, users[0].TwoFCode, true, 600)
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

	var l *domain.Login
	if user.TwoFStatus == "pending" {
		params := &verify.UpdateFactorParams{}
		params.SetAuthPayload(code)
		resp, err := s.twilioClient.VerifyV2.UpdateFactor(os.Getenv("TWILIO_SERVICE_SID"), user.ID, user.TwoFCode, params)
		if err != nil {
			return nil, err
		}

		if resp.Status == nil || *resp.Status != "verified" {
			return nil, errors.New("invaid auth")
		}

		user.TwoFStatus = "approved"
		user.TwoFActive = true
		token, err := s.auth.Create(user.ID, user.Username, user.TwoFCode, true, 3000)
		if err != nil {
			return nil, err
		}

		if err := s.Update(ctx, user.ID, nil, nil, nil, nil, &user.TwoFStatus, &user.TwoFCode, &user.TwoFActive); err != nil {
			return nil, err
		}

		l = &domain.Login{
			Status:    "ok",
			TwoFactor: user.TwoFActive,
			Token:     token,
		}

	} else {
		params := &verify.CreateChallengeParams{}
		params.SetAuthPayload(code)
		params.SetFactorSid(user.TwoFCode)

		resp, err := s.twilioClient.VerifyV2.CreateChallenge(os.Getenv("TWILIO_SERVICE_SID"), user.ID, params)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		fmt.Println(*resp.Status)
		if resp.Status == nil || *resp.Status != "approved" {
			return nil, errors.New("invaid auth")
		}

		token, err := s.auth.Create(user.ID, user.Username, user.TwoFCode, true, 3000)
		if err != nil {
			return nil, err
		}

		l = &domain.Login{
			Status:    "ok",
			TwoFactor: user.TwoFActive,
			Token:     token,
		}
	}

	return l, nil
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

	if user.TwoFStatus == "approved" {
		/*	if user.TwoFStatus == "pending" {
			generateQR(user.ID, fmt.Sprintf("uri:otpauth://totp/otp-service:UserLab%%20Token%%20Example?secret=%s&issuer=otp-service&algorithm=SHA1&digits=6&period=60", user.TwoFCode))
			return nil
		}*/
		return fmt.Errorf("the 2FA status is %s", user.TwoFStatus)
	}
	params := &verify.CreateNewFactorParams{}
	params.SetFriendlyName("UserLab Token Example")
	params.SetFactorType("totp")

	resp, err := s.twilioClient.VerifyV2.CreateNewFactor(os.Getenv("TWILIO_SERVICE_SID"), user.ID, params)
	if err != nil {
		return err
	}

	if resp.Binding == nil {
		return errors.New("invalid hash")
	}

	qrSecret := (*resp.Binding).(map[string]interface{})["secret"].(string)

	user.TwoFCode = *resp.Sid
	user.TwoFActive = true
	user.TwoFStatus = "pending"

	if err := s.Update(ctx, user.ID, nil, nil, nil, nil, &user.TwoFStatus, &user.TwoFCode, &user.TwoFActive); err != nil {
		return err
	}
	fmt.Printf(os.Getenv("TWILIO_QR"), qrSecret)
	generateQR(user.ID, fmt.Sprintf(os.Getenv("TWILIO_QR"), qrSecret))
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

func generateQR(userID, url string) {
	qrCode, _ := qrcode.New(url, qrcode.Medium)
	fileName := fmt.Sprintf("./files/%s.png", userID)
	err := qrCode.WriteFile(256, fileName)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(fmt.Sprintf("QR code generated and saved as %s.png", userID))
}
