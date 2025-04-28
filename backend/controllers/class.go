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

type ClassController struct {
	db *mongo.Database
}

func NewClassController(db *mongo.Database) *ClassController {
	return &ClassController{db: db}
}

// GetClasses returns a list of all classes
func (cc *ClassController) GetClasses(c *gin.Context) {
	ctx := context.Background()
	collection := cc.db.Collection("classes")

	// Setup options for populating faculty information
	lookupStage := bson.D{
		{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "facultyId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "faculty"},
		}},
	}
	unwindStage := bson.D{{Key: "$unwind", Value: "$faculty"}}
	projectStage := bson.D{
		{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 1},
			{Key: "name", Value: 1},
			{Key: "schedule", Value: 1},
			{Key: "capacity", Value: 1},
			{Key: "enrolledCount", Value: 1},
			{Key: "status", Value: 1},
			{Key: "faculty.username", Value: 1},
			{Key: "faculty.email", Value: 1},
		}},
	}

	cursor, err := collection.Aggregate(ctx, mongo.Pipeline{lookupStage, unwindStage, projectStage})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching classes"})
		return
	}
	defer cursor.Close(ctx)

	var classes []bson.M
	if err := cursor.All(ctx, &classes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding classes"})
		return
	}

	c.JSON(http.StatusOK, classes)
}

// GetClass returns details of a specific class
func (cc *ClassController) GetClass(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	var class models.Class
	err = cc.db.Collection("classes").FindOne(context.Background(), bson.M{"_id": classID}).Decode(&class)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching class"})
		return
	}

	// Fetch faculty information
	var faculty models.User
	err = cc.db.Collection("users").FindOne(context.Background(), bson.M{"_id": class.FacultyID}).Decode(&faculty)
	if err == nil {
		class.Faculty = &faculty
	}

	c.JSON(http.StatusOK, class)
}

// CreateClass creates a new class
func (cc *ClassController) CreateClass(c *gin.Context) {
	var input struct {
		Name      string             `json:"name" binding:"required"`
		FacultyID primitive.ObjectID `json:"facultyId" binding:"required"`
		Schedule  string             `json:"schedule" binding:"required"`
		Capacity  int                `json:"capacity" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Verify faculty exists and is of type faculty
	var faculty models.User
	err := cc.db.Collection("users").FindOne(context.Background(), bson.M{
		"_id":      input.FacultyID,
		"userType": "faculty",
	}).Decode(&faculty)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid faculty ID"})
		return
	}

	class := models.Class{
		Name:          input.Name,
		FacultyID:     input.FacultyID,
		Schedule:      input.Schedule,
		Capacity:      input.Capacity,
		EnrolledCount: 0,
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	result, err := cc.db.Collection("classes").InsertOne(context.Background(), class)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating class"})
		return
	}

	class.ID = result.InsertedID.(primitive.ObjectID)
	class.Faculty = &faculty
	c.JSON(http.StatusCreated, class)
}

// UpdateClass updates an existing class
func (cc *ClassController) UpdateClass(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	var input struct {
		Name      string             `json:"name"`
		FacultyID primitive.ObjectID `json:"facultyId"`
		Schedule  string             `json:"schedule"`
		Capacity  int                `json:"capacity"`
		Status    string             `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	update := bson.M{"$set": bson.M{
		"updatedAt": time.Now(),
	}}

	if input.Name != "" {
		update["$set"].(bson.M)["name"] = input.Name
	}
	if !input.FacultyID.IsZero() {
		// Verify faculty exists and is of type faculty
		var faculty models.User
		err := cc.db.Collection("users").FindOne(context.Background(), bson.M{
			"_id":      input.FacultyID,
			"userType": "faculty",
		}).Decode(&faculty)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid faculty ID"})
			return
		}
		update["$set"].(bson.M)["facultyId"] = input.FacultyID
	}
	if input.Schedule != "" {
		update["$set"].(bson.M)["schedule"] = input.Schedule
	}
	if input.Capacity > 0 {
		update["$set"].(bson.M)["capacity"] = input.Capacity
	}
	if input.Status != "" {
		update["$set"].(bson.M)["status"] = input.Status
	}

	result := cc.db.Collection("classes").FindOneAndUpdate(
		context.Background(),
		bson.M{"_id": classID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updatedClass models.Class
	if err := result.Decode(&updatedClass); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating class"})
		return
	}

	// Fetch faculty information
	var faculty models.User
	err = cc.db.Collection("users").FindOne(context.Background(), bson.M{"_id": updatedClass.FacultyID}).Decode(&faculty)
	if err == nil {
		updatedClass.Faculty = &faculty
	}

	c.JSON(http.StatusOK, updatedClass)
}

// DeleteClass deletes a class
func (cc *ClassController) DeleteClass(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	result, err := cc.db.Collection("classes").DeleteOne(context.Background(), bson.M{"_id": classID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting class"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Class deleted successfully"})
}

// EnrollInClass enrolls a student in a class
func (cc *ClassController) EnrollInClass(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	studentID, _ := c.Get("userId")
	studentObjID := studentID.(primitive.ObjectID)

	// Check if class exists and has capacity
	var class models.Class
	err = cc.db.Collection("classes").FindOne(context.Background(), bson.M{
		"_id":    classID,
		"status": "active",
	}).Decode(&class)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Class not found or not active"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching class"})
		return
	}

	if class.EnrolledCount >= class.Capacity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Class is full"})
		return
	}

	// Check if student is already enrolled
	count, err := cc.db.Collection("classes").CountDocuments(context.Background(), bson.M{
		"_id":      classID,
		"enrolled": studentObjID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking enrollment"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already enrolled in this class"})
		return
	}

	// Update class enrollment
	result := cc.db.Collection("classes").FindOneAndUpdate(
		context.Background(),
		bson.M{"_id": classID},
		bson.M{
			"$push": bson.M{"enrolled": studentObjID},
			"$inc":  bson.M{"enrolledCount": 1},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updatedClass models.Class
	if err := result.Decode(&updatedClass); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating enrollment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully enrolled in class",
		"class":   updatedClass,
	})
}

// DropClass removes a student from a class
func (cc *ClassController) DropClass(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	studentID, _ := c.Get("userId")
	studentObjID := studentID.(primitive.ObjectID)

	// Update class enrollment
	result := cc.db.Collection("classes").FindOneAndUpdate(
		context.Background(),
		bson.M{
			"_id":      classID,
			"enrolled": studentObjID,
		},
		bson.M{
			"$pull": bson.M{"enrolled": studentObjID},
			"$inc":  bson.M{"enrolledCount": -1},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updatedClass models.Class
	if err := result.Decode(&updatedClass); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not enrolled in this class"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating enrollment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully dropped class",
		"class":   updatedClass,
	})
}

// GetAvailableClasses returns a list of classes that are active and not full
func (cc *ClassController) GetAvailableClasses(c *gin.Context) {
	ctx := context.Background()

	// Find classes that are active and have space available
	filter := bson.M{
		"status": "active",
		"$expr": bson.M{
			"$lt": []interface{}{"$enrolledCount", "$capacity"},
		},
	}

	opts := options.Find().SetSort(bson.M{"createdAt": -1})

	cursor, err := cc.db.Collection("classes").Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching available classes"})
		return
	}
	defer cursor.Close(ctx)

	var classes []models.Class
	if err := cursor.All(ctx, &classes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding classes"})
		return
	}

	// Populate faculty information for each class
	for i := range classes {
		var faculty models.User
		err := cc.db.Collection("users").FindOne(
			ctx,
			bson.M{"_id": classes[i].FacultyID},
			options.FindOne().SetProjection(bson.M{"password": 0}),
		).Decode(&faculty)
		if err == nil {
			classes[i].Faculty = &faculty
		}
	}

	c.JSON(http.StatusOK, classes)
}

// GetFacultyClasses returns all classes taught by the requesting faculty member
func (cc *ClassController) GetFacultyClasses(c *gin.Context) {
	ctx := context.Background()

	// Get faculty ID from context
	facultyID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	// Find all classes for this faculty
	filter := bson.M{"facultyId": facultyID}
	opts := options.Find().SetSort(bson.M{"createdAt": -1})

	cursor, err := cc.db.Collection("classes").Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching faculty classes"})
		return
	}
	defer cursor.Close(ctx)

	var classes []models.Class
	if err := cursor.All(ctx, &classes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding classes"})
		return
	}

	c.JSON(http.StatusOK, classes)
}

// GetFacultySchedule returns the teaching schedule for a faculty member
func (cc *ClassController) GetFacultySchedule(c *gin.Context) {
	facultyID, _ := c.Get("userId")
	facultyObjID := facultyID.(primitive.ObjectID)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"facultyId": facultyObjID,
			"status":    "active",
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":      1,
			"name":     1,
			"schedule": 1,
		}}},
	}

	cursor, err := cc.db.Collection("classes").Aggregate(context.Background(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching faculty schedule"})
		return
	}
	defer cursor.Close(context.Background())

	var schedule []bson.M
	if err := cursor.All(context.Background(), &schedule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding schedule"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// GetStudentSchedule returns the class schedule for a student
func (cc *ClassController) GetStudentSchedule(c *gin.Context) {
	studentID, _ := c.Get("userId")
	studentObjID := studentID.(primitive.ObjectID)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"enrolled": studentObjID,
			"status":   "active",
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":      1,
			"name":     1,
			"schedule": 1,
		}}},
	}

	cursor, err := cc.db.Collection("classes").Aggregate(context.Background(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching student schedule"})
		return
	}
	defer cursor.Close(context.Background())

	var schedule []bson.M
	if err := cursor.All(context.Background(), &schedule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding schedule"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// GetStudentEnrollments returns the classes a student is enrolled in
func (cc *ClassController) GetStudentEnrollments(c *gin.Context) {
	studentID, _ := c.Get("userId")
	studentObjID := studentID.(primitive.ObjectID)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"enrolled": studentObjID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "facultyId",
			"foreignField": "_id",
			"as":           "faculty",
		}}},
		{{Key: "$unwind", Value: "$faculty"}},
		{{Key: "$project", Value: bson.M{
			"_id":      1,
			"name":     1,
			"schedule": 1,
			"faculty": bson.M{
				"_id":      1,
				"username": 1,
				"email":    1,
			},
		}}},
	}

	cursor, err := cc.db.Collection("classes").Aggregate(context.Background(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching enrollments"})
		return
	}
	defer cursor.Close(context.Background())

	var enrollments []bson.M
	if err := cursor.All(context.Background(), &enrollments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding enrollments"})
		return
	}

	c.JSON(http.StatusOK, enrollments)
}

// CancelClass handles cancellation of a class session
func (cc *ClassController) CancelClass(c *gin.Context) {
	var input struct {
		ClassID primitive.ObjectID `json:"classId" binding:"required"`
		Date    string             `json:"date" binding:"required"`
		Reason  string             `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	facultyID, _ := c.Get("userId")
	facultyObjID := facultyID.(primitive.ObjectID)

	// Verify faculty teaches this class
	count, err := cc.db.Collection("classes").CountDocuments(context.Background(), bson.M{
		"_id":       input.ClassID,
		"facultyId": facultyObjID,
	})
	if err != nil || count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to cancel this class"})
		return
	}

	// Record cancellation
	cancelRecord := bson.M{
		"classId":   input.ClassID,
		"date":      input.Date,
		"reason":    input.Reason,
		"type":      "cancellation",
		"createdAt": time.Now(),
	}

	_, err = cc.db.Collection("schedule_changes").InsertOne(context.Background(), cancelRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording cancellation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Class cancelled successfully"})
}

// RescheduleClass handles rescheduling of a class session
func (cc *ClassController) RescheduleClass(c *gin.Context) {
	var input struct {
		ClassID      primitive.ObjectID `json:"classId" binding:"required"`
		OriginalDate string             `json:"originalDate" binding:"required"`
		NewDate      string             `json:"newDate" binding:"required"`
		NewTime      string             `json:"newTime" binding:"required"`
		Reason       string             `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	facultyID, _ := c.Get("userId")
	facultyObjID := facultyID.(primitive.ObjectID)

	// Verify faculty teaches this class
	count, err := cc.db.Collection("classes").CountDocuments(context.Background(), bson.M{
		"_id":       input.ClassID,
		"facultyId": facultyObjID,
	})
	if err != nil || count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to reschedule this class"})
		return
	}

	// Record rescheduling
	rescheduleRecord := bson.M{
		"classId":      input.ClassID,
		"originalDate": input.OriginalDate,
		"newDate":      input.NewDate,
		"newTime":      input.NewTime,
		"reason":       input.Reason,
		"type":         "reschedule",
		"createdAt":    time.Now(),
	}

	_, err = cc.db.Collection("schedule_changes").InsertOne(context.Background(), rescheduleRecord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording rescheduling"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Class rescheduled successfully"})
}

// GetScheduleChanges returns schedule changes (cancellations and rescheduling) for faculty's classes
func (cc *ClassController) GetScheduleChanges(c *gin.Context) {
	facultyID, _ := c.Get("userId")
	facultyObjID := facultyID.(primitive.ObjectID)

	// Get faculty's classes first
	classCursor, err := cc.db.Collection("classes").Find(context.Background(), bson.M{
		"facultyId": facultyObjID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching faculty classes"})
		return
	}
	defer classCursor.Close(context.Background())

	var classes []models.Class
	if err := classCursor.All(context.Background(), &classes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding classes"})
		return
	}

	// Extract class IDs
	var classIDs []primitive.ObjectID
	for _, class := range classes {
		classIDs = append(classIDs, class.ID)
	}

	// Get schedule changes for these classes
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"classId": bson.M{"$in": classIDs}}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "classes",
			"localField":   "classId",
			"foreignField": "_id",
			"as":           "class",
		}}},
		{{Key: "$unwind", Value: "$class"}},
		{{Key: "$sort", Value: bson.M{"createdAt": -1}}},
	}

	cursor, err := cc.db.Collection("schedule_changes").Aggregate(context.Background(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching schedule changes"})
		return
	}
	defer cursor.Close(context.Background())

	var changes []bson.M
	if err := cursor.All(context.Background(), &changes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding schedule changes"})
		return
	}

	c.JSON(http.StatusOK, changes)
}
