package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username   string             `bson:"username" json:"username"`
	Email      string             `bson:"email" json:"email"`
	Password   string             `bson:"password" json:"-"`
	UserType   string             `bson:"userType" json:"userType"` // student, faculty, or admin
	RollNumber string             `bson:"rollNumber" json:"rollNumber"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// HashPassword hashes the user's password before saving
func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// ValidatePassword checks if the provided password matches the hashed password
func (u *User) ValidatePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// BeforeSave updates timestamps before saving to database
func (u *User) BeforeSave() {
	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now
}
