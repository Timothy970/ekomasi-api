package main

import (
	"adenzo_backend/dtos"
	"adenzo_backend/handlers"
	"adenzo_backend/models"
	"adenzo_backend/routes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "adenzo_backend/docs"

	"github.com/go-redis/redis/v8"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/uptrace/uptrace-go/uptrace"
)

// Define constants for JWT
const (
	JWT_SECRET = "secret"
)

var Db *sql.DB
var redisClient *redis.Client

const migrationDir = "migrations"

// Load environment variables
func init() {
	// Load .env file if it exists
	if os.Getenv("ENVIRONMENT") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Error loading .env file: %v", err)
		}
	}
	// Initialize database connection or other configurations here.
	fmt.Println("Initializing server...")
}

func main() {
	//uptrace connection
	// Configure OpenTelemetry with sensible defaults.
	uptrace.ConfigureOpentelemetry(
		// copy your project DSN here or use UPTRACE_DSN env var
		uptrace.WithDSN("https://dhzrO5dpdBhEXY8FlQfzsw@api.uptrace.dev?grpc=4317"),

		uptrace.WithServiceName("Adenzo"),
		uptrace.WithServiceVersion("1.0.0"),
	)

	/**
		// @title AdEnzo API
	// @version 1.0
	// @description This is the AdEnzo backend API documentation.
	// @securityDefinitions.apikey BearerAuth
	// @in header
	// @name Authorization
	*/
	// database migration CLI flags
	migrateOnly := flag.Bool("migrate", false, "Run migrations and exit")
	action := flag.String("action", "up", "Migration action: up | down")
	target := flag.String("target", "all", "Migration target: all | specific migration number like 001")
	flag.Parse()
	// Run migrations only if --migrate is passed
	if *migrateOnly {
		db, err := initDBConnection(
			os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"),
		)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		if err := runMigrations(db, *action, *target); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migration completed successfully.")
		return
	}
	// Open or create the log file
	logFile, err := os.OpenFile("storage/logs/adenzo.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close() // Ensures the log file is closed when the program exits
	log.SetOutput(logFile)
	// Initialize Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password by default
		DB:       0,                // Default DB
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("Connected to Redis")
	// Initialize the database connection
	Db, err = initDBConnection(
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer Db.Close()

	// Pass Db to models
	models.DB = Db
	//pass redis to models
	handlers.Redis = redisClient
	dtos.Redis = redisClient
	// Initialize router
	router := mux.NewRouter()
	// Swagger route
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Run cart reminders every 3 days (check daily at midnight, or use cron if needed)
	// handlers.StartCartReminderScheduler(24*time.Hour, 3)
	handlers.StartCartReminderScheduler(1*time.Hour, 1)
	// Run wishlist reminders every 7 days
	handlers.StartWishlistReminderScheduler(1*time.Minute, 7)
	// handlers.StartWishlistReminderScheduler(24*time.Hour, 7)
	// Define routes
	routes.SetupRoutes(router)

	// Static file serving
	staticDir := "/static/"
	router.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir("./static"))))

	// Middleware
	// router.Use(middleware.Authenticate)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // Default port if not specified
	}
	// CORS middleware
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},                                     // Allow all origins
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH"}, // Allow specific HTTP methods
		AllowedHeaders:   []string{"Content-Type", "Authorization"},         // Allow specific headers
		AllowCredentials: true,
	}).Handler(router)
	log.Printf("Server started on port %s", port)
	fmt.Printf("Server listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}

// Function to initialize a database connection
func initDBConnection(user, password, host, port, dbName string) (*sql.DB, error) {
	cfg := mysql.Config{
		User:   user,
		Passwd: password,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%s", host, port),
		DBName: dbName,
		Params: map[string]string{
			"parseTime":            "true",
			"loc":                  "Africa/Nairobi",
			"allowNativePasswords": "true",
		},
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %v", err)
	}

	// Test the database connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error pinging database: %v", err)
	}

	return db, nil
}

// Migration runner
func runMigrations(db *sql.DB, action, target string) error {
	matches, err := findMigrationFiles(action, target)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("no matching migration files found for action '%s' and target '%s'", action, target)
	}

	sort.Strings(matches)
	for _, file := range matches {
		if err := executeMigrationFile(db, file); err != nil {
			return err
		}
	}
	return nil
}

func findMigrationFiles(action, target string) ([]string, error) {
	files, err := os.ReadDir(migrationDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %v", err)
	}

	var matches []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if strings.HasSuffix(name, "."+action+".sql") &&
			(target == "all" || strings.HasPrefix(name, target+"_")) {
			matches = append(matches, filepath.Join(migrationDir, name))
		}
	}
	return matches, nil
}

func executeMigrationFile(db *sql.DB, file string) error {
	fmt.Println("Running migration:", file)
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", file, err)
	}
	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration %s: %v", file, err)
	}
	return nil
}
