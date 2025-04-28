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
)

type Holiday struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Date        time.Time          `bson:"date" json:"date"`
	Description string             `bson:"description" json:"description"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// Fix unexported functions by properly casing struct names and function names
type HolidayController struct {
	db *mongo.Database
}

// NewHolidayController creates a new holiday controller
func NewHolidayController(db *mongo.Database) *HolidayController {
	return &HolidayController{db: db}
}

// GetHolidays returns all holidays, optionally filtered by date range
func (hc *HolidayController) GetHolidays(c *gin.Context) {
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

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
			"date": bson.M{
				"$gte": start,
				"$lt":  end,
			},
		}
	}

	cursor, err := hc.db.Collection("holidays").Find(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching holidays"})
		return
	}
	defer cursor.Close(context.Background())

	var holidays []Holiday
	if err := cursor.All(context.Background(), &holidays); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing holidays"})
		return
	}

	c.JSON(http.StatusOK, holidays)
}

// CreateHoliday creates a new holiday
func (hc *HolidayController) CreateHoliday(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required"`
		Date        string `json:"date" binding:"required"`
		Description string `json:"description" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	holiday := Holiday{
		Name:        input.Name,
		Date:        date,
		Description: input.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result, err := hc.db.Collection("holidays").InsertOne(context.Background(), holiday)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating holiday"})
		return
	}

	holiday.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, holiday)
}

// DeleteHoliday deletes a holiday
func (hc *HolidayController) DeleteHoliday(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid holiday ID"})
		return
	}

	result, err := hc.db.Collection("holidays").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting holiday"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Holiday not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Holiday deleted successfully"})
}

// UpdateTimetable updates the timetable for a class
func (hc *HolidayController) UpdateTimetable(c *gin.Context) {
	var input struct {
		ClassID   string `json:"classId" binding:"required"`
		Day       string `json:"day" binding:"required"`
		StartTime string `json:"startTime" binding:"required"`
		EndTime   string `json:"endTime" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	classID, err := primitive.ObjectIDFromHex(input.ClassID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	// Validate day
	validDays := map[string]bool{
		"Monday": true, "Tuesday": true, "Wednesday": true,
		"Thursday": true, "Friday": true,
	}
	if !validDays[input.Day] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day"})
		return
	}

	// Update class schedule
	update := bson.M{
		"$set": bson.M{
			"schedule": map[string]interface{}{
				"day":       input.Day,
				"startTime": input.StartTime,
				"endTime":   input.EndTime,
			},
			"updatedAt": time.Now(),
		},
	}

	result := hc.db.Collection("classes").FindOneAndUpdate(
		context.Background(),
		bson.M{"_id": classID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updatedClass map[string]interface{}
	if err := result.Decode(&updatedClass); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating timetable"})
		return
	}

	c.JSON(http.StatusOK, updatedClass)
}
