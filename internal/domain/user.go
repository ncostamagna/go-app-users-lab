package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID         string         `json:"id" gorm:"type:char(36);not null;primary_key;unique_index"`
	Username   string         `json:"username" gorm:"type:char(20);not null;unique"`
	FirstName  string         `json:"first_name" gorm:"type:char(50);not null"`
	LastName   string         `json:"last_name" gorm:"type:char(50);not null"`
	Email      string         `json:"email" gorm:"type:char(50)"`
	Phone      string         `json:"phone" gorm:"type:char(30)"`
	Password   string         `json:"password,omitempty" gorm:"type:char(150)"`
	TwoFStatus string         `json:"twofa_status" gorm:"type:char(10)"`
	TwoFCode   string         `json:"twofa_code" gorm:"type:char(34)"`
	TwoFActive bool           `json:"twofa_active" gorm:"not null;default:false"`
	CreatedAt  *time.Time     `json:"-"`
	UpdatedAt  *time.Time     `json:"-"`
	Deleted    gorm.DeletedAt `json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {

	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return
}
