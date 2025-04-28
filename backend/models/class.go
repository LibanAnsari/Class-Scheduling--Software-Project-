package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Class struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name          string               `bson:"name" json:"name"`
	FacultyID     primitive.ObjectID   `bson:"facultyId" json:"facultyId"`
	Faculty       *User                `bson:"-" json:"faculty,omitempty"`
	Schedule      string               `bson:"schedule" json:"schedule"`
	Capacity      int                  `bson:"capacity" json:"capacity"`
	Enrolled      []primitive.ObjectID `bson:"enrolled" json:"enrolled,omitempty"`
	EnrolledCount int                  `bson:"enrolledCount" json:"enrolledCount"`
	Status        string               `bson:"status" json:"status"` // active, cancelled, completed
	CreatedAt     time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time            `bson:"updatedAt" json:"updatedAt"`
}

type Attendance struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	ClassID   primitive.ObjectID   `bson:"classId" json:"classId"`
	Date      time.Time            `bson:"date" json:"date"`
	Present   []primitive.ObjectID `bson:"present" json:"present"`
	Absent    []primitive.ObjectID `bson:"absent" json:"absent"`
	CreatedAt time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time            `bson:"updatedAt" json:"updatedAt"`
}

type Performance struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClassID        primitive.ObjectID `bson:"classId" json:"classId"`
	StudentID      primitive.ObjectID `bson:"studentId" json:"studentId"`
	AssessmentName string             `bson:"assessmentName" json:"assessmentName"`
	Score          float64            `bson:"score" json:"score"`
	TotalMarks     float64            `bson:"totalMarks" json:"totalMarks"`
	Date           time.Time          `bson:"date" json:"date"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type Remark struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClassID   primitive.ObjectID `bson:"classId" json:"classId"`
	StudentID primitive.ObjectID `bson:"studentId" json:"studentId"`
	FacultyID primitive.ObjectID `bson:"facultyId" json:"facultyId"`
	Content   string             `bson:"content" json:"content"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type ScheduleChange struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClassID      primitive.ObjectID `bson:"classId" json:"classId"`
	Type         string             `bson:"type" json:"type"` // cancellation or reschedule
	OriginalDate time.Time          `bson:"originalDate" json:"originalDate"`
	NewDate      *time.Time         `bson:"newDate,omitempty" json:"newDate,omitempty"`
	Reason       string             `bson:"reason" json:"reason"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// BeforeSave updates timestamps for Class
func (c *Class) BeforeSave() {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
}

// BeforeSave updates timestamps for Attendance
func (a *Attendance) BeforeSave() {
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
}

// BeforeSave updates timestamps for Performance
func (p *Performance) BeforeSave() {
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
}

// BeforeSave updates timestamps for Remark
func (r *Remark) BeforeSave() {
	now := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
}

// BeforeSave updates timestamps for ScheduleChange
func (s *ScheduleChange) BeforeSave() {
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
}
