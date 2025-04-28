package routes

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"classscheduling/controllers"
	"classscheduling/middleware"
)

func SetupRoutes(router *gin.Engine, db *mongo.Database) {
	// Initialize controllers
	authController := controllers.NewAuthController(db)
	userController := controllers.NewUserController(db)
	classController := controllers.NewClassController(db)
	attendanceController := controllers.NewAttendanceController(db)
	performanceController := controllers.NewPerformanceController(db)
	holidayController := controllers.NewHolidayController(db)

	// Public routes
	public := router.Group("/api")
	{
		auth := public.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/signup", authController.Signup)
		}
	}

	// Protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		// User routes
		protected.GET("/users", middleware.RoleMiddleware("admin"), userController.GetUsers)
		protected.GET("/users/:id", userController.GetUser)
		protected.POST("/users", middleware.RoleMiddleware("admin"), userController.CreateUser)
		protected.PUT("/users/:id", userController.UpdateUser)
		protected.DELETE("/users/:id", middleware.RoleMiddleware("admin"), userController.DeleteUser)

		// Class routes
		protected.GET("/classes", classController.GetClasses)
		protected.GET("/classes/available", classController.GetAvailableClasses)
		protected.POST("/classes", middleware.RoleMiddleware("admin"), classController.CreateClass)
		protected.GET("/classes/:id", classController.GetClass)
		protected.PUT("/classes/:id", middleware.RoleMiddleware("admin", "faculty"), classController.UpdateClass)
		protected.DELETE("/classes/:id", middleware.RoleMiddleware("admin"), classController.DeleteClass)

		// Faculty specific routes
		protected.GET("/faculty/classes", middleware.RoleMiddleware("faculty"), classController.GetFacultyClasses)
		protected.GET("/faculty/schedule", middleware.RoleMiddleware("faculty"), classController.GetFacultySchedule)
		protected.POST("/faculty/class/cancel", middleware.RoleMiddleware("faculty"), classController.CancelClass)
		protected.POST("/faculty/class/reschedule", middleware.RoleMiddleware("faculty"), classController.RescheduleClass)
		protected.GET("/faculty/schedule-changes", middleware.RoleMiddleware("faculty"), classController.GetScheduleChanges)

		// Student specific routes
		protected.GET("/student/schedule", middleware.RoleMiddleware("student"), classController.GetStudentSchedule)
		protected.GET("/student/enrollments", middleware.RoleMiddleware("student"), classController.GetStudentEnrollments)
		protected.GET("/student/attendance", middleware.RoleMiddleware("student"), attendanceController.GetStudentAttendance)
		protected.GET("/student/performance", middleware.RoleMiddleware("student"), performanceController.GetStudentPerformance)
		protected.GET("/student/remarks", middleware.RoleMiddleware("student"), performanceController.GetStudentRemarks)

		// Enrollment routes
		protected.POST("/classes/:id/enroll", middleware.RoleMiddleware("student"), classController.EnrollInClass)
		protected.POST("/classes/:id/drop", middleware.RoleMiddleware("student"), classController.DropClass)

		// Attendance routes
		protected.POST("/classes/:id/attendance", middleware.RoleMiddleware("faculty"), attendanceController.MarkAttendance)
		protected.GET("/classes/:id/attendance", attendanceController.GetAttendance)

		// Performance routes
		protected.POST("/faculty/class/:id/performance", middleware.RoleMiddleware("faculty"), performanceController.AddPerformance)
		protected.GET("/classes/:id/performance", performanceController.GetClassPerformance)
		protected.POST("/faculty/class/:id/remarks", middleware.RoleMiddleware("faculty"), performanceController.AddRemarks)

		// Admin specific routes
		protected.GET("/admin/statistics", middleware.RoleMiddleware("admin"), userController.GetStatistics)
		protected.GET("/admin/activity", middleware.RoleMiddleware("admin"), userController.GetActivity)

		// Holiday management routes
		protected.GET("/admin/holidays", middleware.RoleMiddleware("admin"), holidayController.GetHolidays)
		protected.POST("/admin/holidays", middleware.RoleMiddleware("admin"), holidayController.CreateHoliday)
		protected.DELETE("/admin/holidays/:id", middleware.RoleMiddleware("admin"), holidayController.DeleteHoliday)

		// Timetable management routes
		protected.POST("/admin/timetable", middleware.RoleMiddleware("admin"), holidayController.UpdateTimetable)
	}
}
