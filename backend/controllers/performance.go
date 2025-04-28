package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"classscheduling/models"
)

type PerformanceController struct {
	db *mongo.Database
}

func NewPerformanceController(db *mongo.Database) *PerformanceController {
	return &PerformanceController{db: db}
}

// AddPerformance adds a performance record for a student in a class
func (pc *PerformanceController) AddPerformance(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	var input struct {
		StudentID      primitive.ObjectID `json:"studentId" binding:"required"`
		AssessmentName string             `json:"assessmentName" binding:"required"`
		Score          float64            `json:"score" binding:"required"`
		TotalMarks     float64            `json:"totalMarks" binding:"required"`
		Date           time.Time          `json:"date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Verify class exists
	var class models.Class
	err = pc.db.Collection("classes").FindOne(context.Background(), bson.M{
		"_id":      classID,
		"enrolled": input.StudentID,
	}).Decode(&class)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Class not found or student not enrolled"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching class"})
		return
	}

	// Create performance record
	performance := models.Performance{
		ClassID:        classID,
		StudentID:      input.StudentID,
		AssessmentName: input.AssessmentName,
		Score:          input.Score,
		TotalMarks:     input.TotalMarks,
		Date:           input.Date,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	result, err := pc.db.Collection("performance").InsertOne(context.Background(), performance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording performance"})
		return
	}

	performance.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, performance)
}

// GetClassPerformance retrieves performance records for all students in a class
func (pc *PerformanceController) GetClassPerformance(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	// Build the pipeline for aggregation
	pipeline := []bson.M{
		{
			"$match": bson.M{"classId": classID},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "studentId",
				"foreignField": "_id",
				"as":           "student",
			},
		},
		{
			"$unwind": "$student",
		},
		{
			"$project": bson.M{
				"_id":            1,
				"assessmentName": 1,
				"score":          1,
				"totalMarks":     1,
				"date":           1,
				"student": bson.M{
					"_id":      1,
					"username": 1,
					"email":    1,
				},
			},
		},
		{
			"$sort": bson.M{"date": -1},
		},
	}

	cursor, err := pc.db.Collection("performance").Aggregate(context.Background(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching performance records"})
		return
	}
	defer cursor.Close(context.Background())

	var results []bson.M
	if err := cursor.All(context.Background(), &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding performance records"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetStudentPerformance retrieves performance records for a specific student
func (pc *PerformanceController) GetStudentPerformance(c *gin.Context) {
	studentID, _ := c.Get("userId")
	studentObjID := studentID.(primitive.ObjectID)

	// Optional class ID filter
	classID := c.Query("classId")

	// Build the pipeline for aggregation
	pipeline := []bson.M{
		{
			"$match": bson.M{"studentId": studentObjID},
		},
		{
			"$lookup": bson.M{
				"from":         "classes",
				"localField":   "classId",
				"foreignField": "_id",
				"as":           "class",
			},
		},
		{
			"$unwind": "$class",
		},
	}

	if classID != "" {
		if classObjID, err := primitive.ObjectIDFromHex(classID); err == nil {
			pipeline[0]["$match"].(bson.M)["classId"] = classObjID
		}
	}

	// Add sorting and projection
	pipeline = append(pipeline,
		bson.M{
			"$sort": bson.M{"date": -1},
		},
		bson.M{
			"$project": bson.M{
				"_id":            1,
				"assessmentName": 1,
				"score":          1,
				"totalMarks":     1,
				"date":           1,
				"class": bson.M{
					"_id":      1,
					"name":     1,
					"schedule": 1,
				},
			},
		},
	)

	cursor, err := pc.db.Collection("performance").Aggregate(context.Background(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching performance records"})
		return
	}
	defer cursor.Close(context.Background())

	var results []bson.M
	if err := cursor.All(context.Background(), &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding performance records"})
		return
	}

	// Calculate statistics for each class
	if len(results) > 0 {
		classStats := make(map[string]struct {
			TotalScore      float64
			TotalMarks      float64
			AssessmentCount int
		})

		for _, record := range results {
			class := record["class"].(bson.M)
			classID := class["_id"].(primitive.ObjectID).Hex()
			score := record["score"].(float64)
			totalMarks := record["totalMarks"].(float64)

			stats := classStats[classID]
			stats.TotalScore += score
			stats.TotalMarks += totalMarks
			stats.AssessmentCount++
			classStats[classID] = stats
		}

		// Add statistics to the response
		stats := make(map[string]gin.H)
		for classID, classData := range classStats {
			stats[classID] = gin.H{
				"averageScore":    classData.TotalScore / float64(classData.AssessmentCount),
				"averagePercent":  (classData.TotalScore / classData.TotalMarks) * 100,
				"assessmentCount": classData.AssessmentCount,
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"records": results,
			"stats":   stats,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"records": []bson.M{},
			"stats":   gin.H{},
		})
	}
}

// GetStudentRemarks retrieves remarks for a specific student
func (pc *PerformanceController) GetStudentRemarks(c *gin.Context) {
	studentID, _ := c.Get("userId")
	studentObjID := studentID.(primitive.ObjectID)

	pipeline := []bson.M{
		{
			"$match": bson.M{"studentId": studentObjID},
		},
		{
			"$lookup": bson.M{
				"from":         "classes",
				"localField":   "classId",
				"foreignField": "_id",
				"as":           "class",
			},
		},
		{
			"$unwind": "$class",
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "facultyId",
				"foreignField": "_id",
				"as":           "faculty",
			},
		},
		{
			"$unwind": "$faculty",
		},
		{
			"$project": bson.M{
				"_id":     1,
				"remarks": 1,
				"date":    1,
				"class": bson.M{
					"name": 1,
				},
				"faculty": bson.M{
					"username": 1,
				},
			},
		},
		{
			"$sort": bson.M{"date": -1},
		},
	}

	cursor, err := pc.db.Collection("remarks").Aggregate(context.Background(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching remarks"})
		return
	}
	defer cursor.Close(context.Background())

	var remarks []bson.M
	if err := cursor.All(context.Background(), &remarks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding remarks"})
		return
	}

	c.JSON(http.StatusOK, remarks)
}

// AddRemarks adds a remark for a student in a class
func (pc *PerformanceController) AddRemarks(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	facultyID, _ := c.Get("userId")
	facultyObjID := facultyID.(primitive.ObjectID)

	var input struct {
		StudentID primitive.ObjectID `json:"studentId" binding:"required"`
		Content   string             `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Verify class exists and student is enrolled
	var class models.Class
	err = pc.db.Collection("classes").FindOne(context.Background(), bson.M{
		"_id":      classID,
		"enrolled": input.StudentID,
	}).Decode(&class)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Class not found or student not enrolled"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching class"})
		return
	}

	// Create remark record
	remark := models.Remark{
		ClassID:   classID,
		StudentID: input.StudentID,
		FacultyID: facultyObjID,
		Content:   input.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result, err := pc.db.Collection("remarks").InsertOne(context.Background(), remark)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording remark"})
		return
	}

	remark.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, remark)
}
