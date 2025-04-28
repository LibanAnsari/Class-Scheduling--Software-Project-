package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"classscheduling/models"
)

type UserController struct {
	db *mongo.Database
}

func NewUserController(db *mongo.Database) *UserController {
	return &UserController{db: db}
}

// CreateUser creates a new user (admin only)
func (uc *UserController) CreateUser(c *gin.Context) {
	var input struct {
		Username   string `json:"username" binding:"required"`
		Email      string `json:"email" binding:"required,email"`
		Password   string `json:"password" binding:"required,min=6"`
		UserType   string `json:"userType" binding:"required"`
		RollNumber string `json:"rollNumber"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Validate user type
	if input.UserType != "student" && input.UserType != "faculty" && input.UserType != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user type"})
		return
	}

	// Validate roll number for students
	if input.UserType == "student" && input.RollNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Roll number is required for students"})
		return
	}

	// Check if username is already taken
	count, err := uc.db.Collection("users").CountDocuments(context.Background(), bson.M{"username": input.Username})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
		return
	}

	// Check if email is already registered
	count, err = uc.db.Collection("users").CountDocuments(context.Background(), bson.M{"email": input.Email})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
		return
	}

	// Create new user
	user := models.User{
		Username:   input.Username,
		Email:      input.Email,
		Password:   input.Password,
		UserType:   input.UserType,
		RollNumber: input.RollNumber,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Hash password
	if err := user.HashPassword(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
		return
	}

	// Insert user into database
	result, err := uc.db.Collection("users").InsertOne(context.Background(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating user"})
		return
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	user.Password = "" // Remove password from response

	c.JSON(http.StatusCreated, user)
}

// GetStatistics returns system statistics for admin dashboard
func (uc *UserController) GetStatistics(c *gin.Context) {
	ctx := context.Background()
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	// Parse dates if provided
	var timeFilter bson.M
	if startDate != "" && endDate != "" {
		start, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
			return
		}
		end, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
			return
		}
		// Add one day to end date to include the full day
		end = end.Add(24 * time.Hour)
		timeFilter = bson.M{
			"createdAt": bson.M{
				"$gte": start,
				"$lt":  end,
			},
		}
	}

	// Get user counts by type
	userStats := make(map[string]int64)
	userTypes := []string{"student", "faculty", "admin"}
	for _, userType := range userTypes {
		filter := bson.M{"userType": userType}
		if timeFilter != nil {
			filter = bson.M{
				"$and": []bson.M{
					filter,
					timeFilter,
				},
			}
		}
		count, err := uc.db.Collection("users").CountDocuments(ctx, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user statistics"})
			return
		}
		userStats[userType] = count
	}

	// Get class statistics
	classFilter := bson.M{"status": "active"}
	if timeFilter != nil {
		classFilter = bson.M{
			"$and": []bson.M{
				classFilter,
				timeFilter,
			},
		}
	}

	// Get active classes with enrollment info
	classPipeline := mongo.Pipeline{
		primitive.D{
			{Key: "$match", Value: classFilter},
		},
		primitive.D{
			{Key: "$project", Value: bson.M{
				"name":     1,
				"enrolled": 1,
				"capacity": 1,
			}},
		},
	}

	cursor, err := uc.db.Collection("classes").Aggregate(ctx, classPipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching class statistics"})
		return
	}
	defer cursor.Close(ctx)

	var classes []struct {
		Name     string `bson:"name"`
		Enrolled int    `bson:"enrolled"`
		Capacity int    `bson:"capacity"`
	}
	if err := cursor.All(ctx, &classes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing class statistics"})
		return
	}

	// Calculate average class size
	var totalEnrolled int
	classEnrollment := make([]map[string]interface{}, len(classes))
	for i, class := range classes {
		totalEnrolled += class.Enrolled
		classEnrollment[i] = map[string]interface{}{
			"name":     class.Name,
			"enrolled": class.Enrolled,
			"capacity": class.Capacity,
		}
	}

	averageClassSize := 0.0
	if len(classes) > 0 {
		averageClassSize = float64(totalEnrolled) / float64(len(classes))
	}

	c.JSON(http.StatusOK, gin.H{
		"userCounts":       userStats,
		"totalClasses":     len(classes),
		"averageClassSize": averageClassSize,
		"classEnrollment":  classEnrollment,
		"activeUsers":      userStats["student"] + userStats["faculty"], // Active users based on student + faculty count
	})
}

// GetActivity returns recent system activity for admin dashboard
func (uc *UserController) GetActivity(c *gin.Context) {
	ctx := context.Background()
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	// Parse dates if provided
	filter := bson.M{}
	if startDate != "" && endDate != "" {
		start, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
			return
		}
		end, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
			return
		}
		// Add one day to end date to include the full day
		end = end.Add(24 * time.Hour)
		filter = bson.M{
			"timestamp": bson.M{
				"$gte": start,
				"$lt":  end,
			},
		}
	}

	pipeline := mongo.Pipeline{
		primitive.D{
			{Key: "$match", Value: filter},
		},
		primitive.D{
			{Key: "$sort", Value: bson.M{"timestamp": -1}},
		},
		primitive.D{
			{Key: "$limit", Value: 50},
		},
	}

	cursor, err := uc.db.Collection("activity").Aggregate(ctx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching activity logs"})
		return
	}
	defer cursor.Close(ctx)

	var activities []bson.M
	if err := cursor.All(ctx, &activities); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing activity logs"})
		return
	}

	c.JSON(http.StatusOK, activities)
}

// GetUsers returns a list of users, optionally filtered by user type
func (uc *UserController) GetUsers(c *gin.Context) {
	userType := c.Query("type")
	filter := bson.M{}
	if userType != "" {
		filter["userType"] = userType
	}

	opts := options.Find().SetProjection(bson.M{
		"password": 0,
	})

	cursor, err := uc.db.Collection("users").Find(context.Background(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching users"})
		return
	}
	defer cursor.Close(context.Background())

	var users []models.User
	if err := cursor.All(context.Background(), &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser returns details of a specific user
func (uc *UserController) GetUser(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	err = uc.db.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
		options.FindOne().SetProjection(bson.M{"password": 0}),
	).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser updates an existing user
func (uc *UserController) UpdateUser(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		UserType string `json:"userType"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Get current user for password and type validation
	var currentUser models.User
	err = uc.db.Collection("users").FindOne(context.Background(), bson.M{"_id": userID}).Decode(&currentUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user"})
		return
	}

	update := bson.M{"$set": bson.M{
		"updatedAt": time.Now(),
	}}

	if input.Username != "" {
		// Check if username is already taken by another user
		count, err := uc.db.Collection("users").CountDocuments(
			context.Background(),
			bson.M{
				"username": input.Username,
				"_id":      bson.M{"$ne": userID},
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking username"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
			return
		}
		update["$set"].(bson.M)["username"] = input.Username
	}

	if input.Email != "" {
		// Check if email is already registered to another user
		count, err := uc.db.Collection("users").CountDocuments(
			context.Background(),
			bson.M{
				"email": input.Email,
				"_id":   bson.M{"$ne": userID},
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking email"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
			return
		}
		update["$set"].(bson.M)["email"] = input.Email
	}

	if input.Password != "" {
		// Create temporary user to hash password
		tempUser := models.User{Password: input.Password}
		if err := tempUser.HashPassword(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
			return
		}
		update["$set"].(bson.M)["password"] = tempUser.Password
	}

	if input.UserType != "" {
		// Only admin can change user type
		userType, _ := c.Get("userType")
		if userType != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can change user type"})
			return
		}
		update["$set"].(bson.M)["userType"] = input.UserType
	}

	result := uc.db.Collection("users").FindOneAndUpdate(
		context.Background(),
		bson.M{"_id": userID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After).SetProjection(bson.M{"password": 0}),
	)

	var updatedUser models.User
	if err := result.Decode(&updatedUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating user"})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser deletes a user
func (uc *UserController) DeleteUser(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get user's type before deletion for cleanup
	var user models.User
	err = uc.db.Collection("users").FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user"})
		return
	}

	// Check if user has any active classes if they are faculty
	if user.UserType == "faculty" {
		count, err := uc.db.Collection("classes").CountDocuments(
			context.Background(),
			bson.M{
				"facultyId": userID,
				"status":    "active",
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking faculty classes"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete faculty with active classes"})
			return
		}
	}

	// Delete user
	result, err := uc.db.Collection("users").DeleteOne(context.Background(), bson.M{"_id": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Cleanup associated data
	if user.UserType == "student" {
		// Remove student from all enrolled classes
		_, err = uc.db.Collection("classes").UpdateMany(
			context.Background(),
			bson.M{"enrolled": userID},
			bson.M{
				"$pull": bson.M{"enrolled": userID},
				"$inc":  bson.M{"enrolledCount": -1},
			},
		)
		if err != nil {
			// Log error but don't return it since user is already deleted
			c.JSON(http.StatusOK, gin.H{
				"message": "User deleted but there was an error updating their class enrollments",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
