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

type AttendanceController struct {
	db *mongo.Database
}

func NewAttendanceController(db *mongo.Database) *AttendanceController {
	return &AttendanceController{db: db}
}

// MarkAttendance records attendance for a class session
func (ac *AttendanceController) MarkAttendance(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	var input struct {
		Date    time.Time            `json:"date" binding:"required"`
		Present []primitive.ObjectID `json:"present" binding:"required"`
		Absent  []primitive.ObjectID `json:"absent" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Verify class exists
	var class models.Class
	err = ac.db.Collection("classes").FindOne(context.Background(), bson.M{"_id": classID}).Decode(&class)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching class"})
		return
	}

	// Create attendance record
	attendance := models.Attendance{
		ClassID:   classID,
		Date:      input.Date,
		Present:   input.Present,
		Absent:    input.Absent,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result, err := ac.db.Collection("attendance").InsertOne(context.Background(), attendance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording attendance"})
		return
	}

	attendance.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, attendance)
}

// GetAttendance retrieves attendance records for a class
func (ac *AttendanceController) GetAttendance(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	// Optional date range filter
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	filter := bson.M{"classId": classID}
	if startDate != "" && endDate != "" {
		start, err := time.Parse(time.RFC3339, startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
			return
		}
		end, err := time.Parse(time.RFC3339, endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
			return
		}
		filter["date"] = bson.M{
			"$gte": start,
			"$lte": end,
		}
	}

	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})
	cursor, err := ac.db.Collection("attendance").Find(context.Background(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching attendance records"})
		return
	}
	defer cursor.Close(context.Background())

	var attendance []models.Attendance
	if err := cursor.All(context.Background(), &attendance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding attendance records"})
		return
	}

	c.JSON(http.StatusOK, attendance)
}

// GetStudentAttendance retrieves attendance records for a specific student
func (ac *AttendanceController) GetStudentAttendance(c *gin.Context) {
	studentID, _ := c.Get("userId")
	studentObjID := studentID.(primitive.ObjectID)

	// Optional class ID filter
	classID := c.Query("classId")

	// Optional date range filter
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	// Build the pipeline for aggregation
	pipeline := []bson.M{
		{
			"$lookup": bson.M{
				"from":         "classes",
				"localField":   "classId",
				"foreignField": "_id",
				"as":           "class",
			},
		},
		{"$unwind": "$class"},
	}

	// Add filters to match stage
	matchStage := bson.M{
		"$or": []bson.M{
			{"present": studentObjID},
			{"absent": studentObjID},
		},
	}

	if classID != "" {
		classObjID, err := primitive.ObjectIDFromHex(classID)
		if err == nil {
			matchStage["classId"] = classObjID
		}
	}

	if startDate != "" && endDate != "" {
		start, err1 := time.Parse(time.RFC3339, startDate)
		end, err2 := time.Parse(time.RFC3339, endDate)
		if err1 == nil && err2 == nil {
			matchStage["date"] = bson.M{
				"$gte": start,
				"$lte": end,
			}
		}
	}

	pipeline = append(pipeline, bson.M{"$match": matchStage})

	// Add sort stage
	pipeline = append(pipeline, bson.M{
		"$sort": bson.M{"date": -1},
	})

	cursor, err := ac.db.Collection("attendance").Aggregate(context.Background(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching attendance records"})
		return
	}
	defer cursor.Close(context.Background())

	var results []bson.M
	if err := cursor.All(context.Background(), &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding attendance records"})
		return
	}

	c.JSON(http.StatusOK, results)
}
