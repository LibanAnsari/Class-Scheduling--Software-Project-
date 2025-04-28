package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"classscheduling/middleware"
	"classscheduling/models"
)

type AuthController struct {
	db *mongo.Database
}

func NewAuthController(db *mongo.Database) *AuthController {
	return &AuthController{db: db}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	UserType string `json:"userType" binding:"required"`
}

// Login handles user authentication and returns a JWT token
func (ac *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate userType
	if req.UserType != "admin" && req.UserType != "faculty" && req.UserType != "student" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user type"})
		return
	}

	// Find user by username and userType
	var user models.User
	err := ac.db.Collection("users").FindOne(context.Background(), bson.M{
		"username": req.Username,
		"userType": req.UserType,
	}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Validate password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Username, user.UserType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"userType": user.UserType,
		},
	})
}

type SignupRequest struct {
	Username   string `json:"username" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6"`
	UserType   string `json:"userType" binding:"required"`
	RollNumber string `json:"rollNumber"`
}

// Signup handles new user registration
func (ac *AuthController) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Validate user type
	if req.UserType != "student" && req.UserType != "faculty" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user type"})
		return
	}

	// Validate roll number for students
	if req.UserType == "student" && req.RollNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Roll number is required for students"})
		return
	}

	// Check if username is already taken
	count, err := ac.db.Collection("users").CountDocuments(context.Background(), bson.M{"username": req.Username})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
		return
	}

	// Check if email is already registered
	count, err = ac.db.Collection("users").CountDocuments(context.Background(), bson.M{"email": req.Email})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
		return
	}

	// Create new user
	user := models.User{
		Username:   req.Username,
		Email:      req.Email,
		Password:   string(hashedPassword),
		UserType:   req.UserType,
		RollNumber: req.RollNumber,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Insert user into database
	result, err := ac.db.Collection("users").InsertOne(context.Background(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating user"})
		return
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Username, user.UserType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user": gin.H{
			"id":         result.InsertedID,
			"username":   user.Username,
			"email":      user.Email,
			"userType":   user.UserType,
			"rollNumber": user.RollNumber,
		},
	})
}
