package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/ncostamagna/go-app-users-lab/internal/user"
	"github.com/ncostamagna/go-http-utils/response"
)

func NewUserHTTPServer(ctx context.Context, endpoints user.Endpoints) http.Handler {

	r := mux.NewRouter()

	opts := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(encodeError),
	}

	r.Handle("/users", httptransport.NewServer(
		endpoint.Endpoint(endpoints.Create),
		decodeCreateUser, encodeResponse,
		opts...,
	)).Methods("POST")

	r.Handle("/users", httptransport.NewServer(
		endpoint.Endpoint(endpoints.GetAll),
		decodeGetAllUser,
		encodeResponse,
		opts...,
	)).Methods("GET")

	r.Handle("/users/login", httptransport.NewServer(
		endpoint.Endpoint(endpoints.Login),
		decodeLoginUser, encodeResponse,
		opts...,
	)).Methods("POST")

	r.Handle("/users/login/2fa", httptransport.NewServer(
		endpoint.Endpoint(endpoints.Login2FA),
		decodeLogin2FAUser, encodeResponse,
		opts...,
	)).Methods("POST")

	r.Handle("/users/2fa", httptransport.NewServer(
		endpoint.Endpoint(endpoints.Create2FA),
		decodeCreate2FAUser, encodeResponse,
		opts...,
	)).Methods("POST")

	r.Handle("/users/{id}", httptransport.NewServer(
		endpoint.Endpoint(endpoints.Get),
		decodeGetUser,
		encodeResponse,
		opts...,
	)).Methods("GET")

	r.Handle("/users/{id}", httptransport.NewServer(
		endpoint.Endpoint(endpoints.Update),
		decodeUpdateUser,
		encodeResponse,
		opts...,
	)).Methods("PATCH")

	r.Handle("/users/{id}", httptransport.NewServer(
		endpoint.Endpoint(endpoints.Delete),
		decodeDeleteUser,
		encodeResponse,
		opts...,
	)).Methods("DELETE")

	return r
}

func decodeCreateUser(_ context.Context, r *http.Request) (interface{}, error) {

	var req user.CreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, response.BadRequest(fmt.Sprintf("invalid request format: '%v'", err.Error()))
	}

	return req, nil
}

func decodeLoginUser(_ context.Context, r *http.Request) (interface{}, error) {

	var req user.LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, response.BadRequest(fmt.Sprintf("invalid request format: '%v'", err.Error()))
	}

	return req, nil
}

func decodeCreate2FAUser(_ context.Context, r *http.Request) (interface{}, error) {

	return user.Create2FAReq{
		Token: r.Header.Get("Authorization"),
	}, nil
}

func decodeLogin2FAUser(_ context.Context, r *http.Request) (interface{}, error) {

	var req user.Login2FAReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, response.BadRequest(fmt.Sprintf("invalid request format: '%v'", err.Error()))
	}
	req.Token = r.Header.Get("Authorization")

	return req, nil
}

func decodeGetUser(_ context.Context, r *http.Request) (interface{}, error) {

	p := mux.Vars(r)
	req := user.GetReq{
		ID: p["id"],
	}

	return req, nil
}

func decodeGetAllUser(_ context.Context, r *http.Request) (interface{}, error) {

	v := r.URL.Query()

	limit, _ := strconv.Atoi(v.Get("limit"))
	page, _ := strconv.Atoi(v.Get("page"))

	req := user.GetAllReq{
		FirstName: v.Get("first_name"),
		LastName:  v.Get("last_name"),
		Limit:     limit,
		Page:      page,
	}

	return req, nil
}

func decodeUpdateUser(_ context.Context, r *http.Request) (interface{}, error) {
	var req user.UpdateReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, response.BadRequest(fmt.Sprintf("invalid request format: '%v'", err.Error()))
	}

	path := mux.Vars(r)
	req.ID = path["id"]

	return req, nil
}

func decodeDeleteUser(_ context.Context, r *http.Request) (interface{}, error) {

	path := mux.Vars(r)
	req := user.DeleteReq{
		ID: path["id"],
	}

	return req, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, resp interface{}) error {
	r := resp.(response.Response)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(r.StatusCode())
	return json.NewEncoder(w).Encode(r)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp := err.(response.Response)
	w.WriteHeader(resp.StatusCode())
	_ = json.NewEncoder(w).Encode(resp)
}
