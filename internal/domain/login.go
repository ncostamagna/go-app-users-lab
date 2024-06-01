package domain

type Login struct {
	Status        string `json:"status"`
	TwoFactor     bool   `json:"two_factor"`
	TwoFactorHash string `json:"two_factor_hash,omitempty"`
	Token         string `json:"token,omitempty"`
}
