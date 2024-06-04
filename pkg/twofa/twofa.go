package twofa

import (
	"fmt"
	"github.com/skip2/go-qrcode"
	"github.com/twilio/twilio-go"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
)

type Status string

const (
	APPROVED Status = "approved"
	PENDING  Status = "pending"
)

type (
	TwoFA interface {
		Create(id string) (*Response, error)
		GenerateQR(id, url string) error
		Verify(id, code, hash string) error
		Check(id, code, hash string) error
	}
	twoFA struct {
		serviceID    string
		qrUrl        string
		friendlyName string
		restClient   *twilio.RestClient
	}

	Response struct {
		Url    string
		Hash   string
		Status Status
	}
)

func New(serviceID, friendlyName, qrUrl string) TwoFA {
	return &twoFA{
		serviceID:    serviceID,
		qrUrl:        qrUrl,
		friendlyName: friendlyName,
		restClient:   twilio.NewRestClient(),
	}
}

func (t twoFA) Create(id string) (*Response, error) {
	params := &verify.CreateNewFactorParams{}
	params.SetFriendlyName(t.friendlyName)
	params.SetFactorType("totp")

	resp, err := t.restClient.VerifyV2.CreateNewFactor(t.serviceID, id, params)
	if err != nil {
		return nil, err
	}

	if resp.Binding == nil {
		return nil, ErrCanNotCreate2Factor
	}

	qrSecret := (*resp.Binding).(map[string]interface{})["secret"].(string)

	return &Response{
		Url:  fmt.Sprintf(t.qrUrl, qrSecret),
		Hash: *resp.Sid,
	}, nil
}

func (t twoFA) GenerateQR(id, url string) error {
	qrCode, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return err
	}

	if err := qrCode.WriteFile(256, fmt.Sprintf("./files/%s.png", id)); err != nil {
		return err
	}

	return nil
}

func (t twoFA) Verify(id, code, hash string) error {
	params := &verify.UpdateFactorParams{}
	params.SetAuthPayload(code)
	resp, err := t.restClient.VerifyV2.UpdateFactor(t.serviceID, id, hash, params)
	if err != nil {
		return err
	}

	if resp.Status == nil || *resp.Status != "verified" {
		return ErrInvalidCode
	}

	return nil
}

func (t twoFA) Check(id, code, hash string) error {
	params := &verify.CreateChallengeParams{}
	params.SetAuthPayload(code)
	params.SetFactorSid(hash)

	resp, err := t.restClient.VerifyV2.CreateChallenge(t.serviceID, id, params)
	if err != nil {
		return err
	}
	fmt.Println(*resp.Status)
	if resp.Status == nil || *resp.Status != "approved" {
		return ErrInvalidCode
	}

	return nil
}
