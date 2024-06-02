package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/ncostamagna/go-http-utils/meta"
	"github.com/ncostamagna/go-http-utils/response"
)

type (
	Controller func(ctx context.Context, request interface{}) (interface{}, error)

	Endpoints struct {
		Create     Controller
		Login      Controller
		Login2FA   Controller
		Create2FA  Controller
		TwoFa      Controller
		LoginTwoFa Controller
		Get        Controller
		GetAll     Controller
		Update     Controller
		Delete     Controller
	}

	Create2FAReq struct {
		Token string
	}

	Login2FAReq struct {
		Token string
		Code  string `json:"code"`
	}

	Create2FARes struct {
		QR string
	}

	CreateReq struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}

	LoginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	GetReq struct {
		ID string
	}

	GetAllReq struct {
		FirstName string
		LastName  string
		Limit     int
		Page      int
	}

	UpdateReq struct {
		ID        string
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		Email     *string `json:"email"`
		Phone     *string `json:"phone"`
	}

	DeleteReq struct {
		ID string
	}

	Response struct {
		Status int         `json:"status"`
		Data   interface{} `json:"data,omitempty"`
		Err    string      `json:"error,omitempty"`
		Meta   *meta.Meta  `json:"meta,omitempty"`
	}

	Config struct {
		LimPageDef string
	}
)

func MakeEndpoints(s Service, config Config) Endpoints {

	return Endpoints{
		Create:    makeCreateEndpoint(s),
		Login:     makeLogin(s),
		Login2FA:  makeLogin2FA(s),
		Create2FA: makeCreate2FA(s),
		Get:       makeGetEndpoint(s),
		GetAll:    makeGetAllEndpoint(s, config),
		Update:    makeUpdateEndpoint(s),
		Delete:    makeDeleteEndpoint(s),
	}

}

func makeCreateEndpoint(s Service) Controller {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(CreateReq)

		if req.FirstName == "" {
			return nil, response.BadRequest(ErrFirstNameRequired.Error())
		}

		if req.LastName == "" {
			return nil, response.BadRequest(ErrLastNameRequired.Error())
		}

		if req.Username == "" {
			return nil, response.BadRequest(ErrUsernameRequired.Error())
		}

		if req.Password == "" {
			return nil, response.BadRequest(ErrPasswordRequired.Error())
		}

		user, err := s.Create(ctx, req.FirstName, req.LastName, req.Email, req.Phone, req.Username, req.Password)
		if err != nil {
			return nil, response.InternalServerError(err.Error())
		}

		return response.Created("success", user, nil), nil
	}
}

func makeLogin(s Service) Controller {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(LoginReq)

		user, err := s.Login(ctx, req.Username, req.Password)
		if err != nil {
			return nil, response.InternalServerError(err.Error())
		}

		return response.OK("success", user, nil), nil
	}
}

func makeLogin2FA(s Service) Controller {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(Login2FAReq)

		user, err := s.GetUserByToken(ctx, req.Token, false)
		if err != nil {
			return nil, response.InternalServerError(err.Error())
		}

		login, err := s.Login2FA(ctx, user, req.Code)
		if err != nil {
			return nil, response.InternalServerError(err.Error())
		}

		return response.OK("success", login, nil), nil
	}
}

func makeCreate2FA(s Service) Controller {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(Create2FAReq)

		user, err := s.GetUserByToken(ctx, req.Token, true)
		if err != nil {
			return nil, response.InternalServerError(err.Error())
		}

		if err := s.Create2FA(ctx, user); err != nil {
			return nil, response.InternalServerError(err.Error())
		}
		return response.OK("success",
			Create2FARes{
				QR: fmt.Sprintf("./files/%s.png", user.ID),
			}, nil), nil
	}
}

func makeGetAllEndpoint(s Service, config Config) Controller {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(GetAllReq)

		filters := Filters{
			FirstName: req.FirstName,
			LastName:  req.LastName,
		}

		count, err := s.Count(ctx, filters)
		if err != nil {
			return nil, response.InternalServerError(err.Error())
		}

		meta, err := meta.New(req.Page, req.Limit, count, config.LimPageDef)
		if err != nil {
			return nil, response.InternalServerError(err.Error())
		}

		users, err := s.GetAll(ctx, filters, meta.Offset(), meta.Limit())
		if err != nil {
			return nil, response.InternalServerError(err.Error())
		}

		return response.OK("success", users, meta), nil
	}
}
func makeGetEndpoint(s Service) Controller {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(GetReq)

		user, err := s.Get(ctx, req.ID)
		if err != nil {

			if errors.As(err, &ErrNotFound{}) {
				return nil, response.NotFound(err.Error())
			}

			return nil, response.InternalServerError(err.Error())
		}

		return response.OK("success", user, nil), nil
	}
}

func makeUpdateEndpoint(s Service) Controller {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateReq)

		if req.FirstName != nil && *req.FirstName == "" {
			return nil, response.BadRequest(ErrFirstNameRequired.Error())
		}

		if req.LastName != nil && *req.LastName == "" {
			return nil, response.BadRequest(ErrLastNameRequired.Error())
		}

		err := s.Update(ctx, req.ID, req.FirstName, req.LastName, req.Email, req.Phone, nil, nil, nil)
		if err != nil {

			if errors.As(err, &ErrNotFound{}) {
				return nil, response.NotFound(err.Error())
			}

			return nil, response.InternalServerError(err.Error())
		}

		return response.OK("success", nil, nil), nil
	}
}

func makeDeleteEndpoint(s Service) Controller {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(DeleteReq)

		err := s.Delete(ctx, req.ID)

		if err != nil {

			if errors.As(err, &ErrNotFound{}) {
				return nil, response.NotFound(err.Error())
			}
			return nil, response.InternalServerError(err.Error())
		}

		return response.OK("success", nil, nil), nil
	}
}
