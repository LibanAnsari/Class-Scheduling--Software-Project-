package main

import (
	"classscheduling/routes"
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database

func initDB() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Get MongoDB connection string from environment variable
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	// Set client options and connect to MongoDB
	clientOptions := options.Client().ApplyURI(mongoURI)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Ping the database to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Get database instance
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "classscheduling"
	}
	db = client.Database(dbName)
	log.Printf("Connected to MongoDB database: %s", dbName)

	// Create indexes for users collection
	userIndexes := []mongo.IndexModel{
		{
			Keys:    map[string]interface{}{"username": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    map[string]interface{}{"email": 1},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = db.Collection("users").Indexes().CreateMany(ctx, userIndexes)
	if err != nil {
		log.Printf("Warning: Could not create user indexes: %v", err)
	}

	// Create indexes for classes collection
	classIndexes := []mongo.IndexModel{
		{
			Keys: map[string]interface{}{"facultyId": 1},
		},
		{
			Keys: map[string]interface{}{"enrolled": 1},
		},
		{
			Keys: map[string]interface{}{"status": 1},
		},
	}

	_, err = db.Collection("classes").Indexes().CreateMany(ctx, classIndexes)
	if err != nil {
		log.Printf("Warning: Could not create class indexes: %v", err)
	}
}

func setupRouter() *gin.Engine {
	router := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://127.0.0.1:5500", // VS Code Live Server default
		"http://localhost:5500", // VS Code Live Server alternative
		"http://127.0.0.1:3000", // Backend API
		"http://localhost:3000", // Backend API alternative
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	router.Use(cors.New(config))

	// Serve static files from the frontend directory
	router.Static("/frontend", "../frontend")

	// Setup routes with the controllers
	routes.SetupRoutes(router, db)

	return router
}

func main() {
	// Initialize database connection
	initDB()

	// Setup router
	router := setupRouter()

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server starting on port %s...", port)
	// Start server	log.Printf("Server starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
