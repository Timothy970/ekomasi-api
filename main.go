package main

import (
	"context"
	"database/sql"
	"ekomasi_backend/config"
	"ekomasi_backend/dtos"
	"ekomasi_backend/handlers"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/routes"
	"ekomasi_backend/utils"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	_ "ekomasi_backend/docs"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/go-sql-driver/mysql"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Define constants for JWT
const (
	JWT_SECRET = "secret"
)

var Db *sql.DB
var redisClient *redis.Client

const migrationDir = "migrations"

// Load environment variables and global configuration
func init() {
	fmt.Println("Initializing server configuration...")
	config.LoadConfig()
}

func main() {
	// Retrieve application configuration
	cfg := config.Get()

	// Standard log configuration
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	/**
		// @title Ekomasi API
	// @version 1.0
	// @description This is the Ekomasi backend API documentation.
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
			cfg.Database.User, cfg.Database.Password,
			cfg.Database.Host, cfg.Database.Port, cfg.Database.Name,
		)
		if err != nil {
			log.Printf("Failed to connect to database: %v", err)
			log.Println(cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		if err := runMigrations(db, *action, *target); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migration completed successfully.")
		return
	}

	// Initialize Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port), // Redis server address
		Username: cfg.Redis.User,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("Connected to Redis")
	// Initialize the database connection
	Db, err = initDBConnection(
		cfg.Database.User, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Name,
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer Db.Close()

	// Pass Db to models
	models.DB = Db
	//pass redis to models
	handlers.Redis = redisClient
	utils.RedisClient = redisClient
	dtos.Redis = redisClient
	middleware.RedisClient = redisClient
	// Start WebSocket broadcaster for multi-instance support
	utils.StartWebSocketBroadcaster()
	// Initialize router with high performance, tenant middleware, and error recovery
	ginEngine := gin.Default()

	// Register WebSocket handler on Gin Engine
	ginEngine.GET("/ws", gin.WrapH(http.HandlerFunc(utils.HandleWebSocket)))

	// Setup all application routes on Gin Engine
	routes.SetupGinRoutes(ginEngine)

	// Swagger route protected with Basic Auth
	swaggerGroup := ginEngine.Group("/swagger", middleware.SwaggerBasicAuth())
	swaggerGroup.GET("/*any", gin.WrapH(httpSwagger.WrapHandler))

	// Static file serving
	ginEngine.Static("/static", "./static")

	// Run schedulers
	handlers.StartVoucherEmailScheduler(10*time.Minute, 0)
	handlers.StartLowStockEmailScheduler(24*time.Hour, 7)

	// Start the server
	port := cfg.Server.Port
	if port == "" {
		port = "8000" // Default port if not specified
	}
	// CORS middleware configuration wrapping Gin Engine
	corsHandler := cors.New(cors.Options{
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler(ginEngine)

	// Create HTTP server instance
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: corsHandler,
	}

	// Start server in background goroutine
	go func() {
		log.Printf("Server started on Gin Engine on port %s", port)
		fmt.Printf("Server listening on port %s (Gin Framework)...\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server ListenAndServe error: %v\n", err)
		}
	}()

	// Listen for OS signal (Ctrl+C, SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown signal received. Shutting down server gracefully...")

	// Create a context with timeout for server shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v\n", err)
	} else {
		log.Println("HTTP server shut down cleanly.")
	}

	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Printf("Error closing Redis client: %v\n", err)
		} else {
			log.Println("Redis client closed.")
		}
	}

	log.Println("Server exiting gracefully.")
}

// Function to initialize a database connection
func initDBConnection(user, password, host, port, dbName string) (*sql.DB, error) {
	cfg := mysql.Config{
		User:      user,
		Passwd:    password,
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%s", host, port),
		DBName:    dbName,
		TLSConfig: "skip-verify",
		Params: map[string]string{
			"parseTime": "true",
			// This was causing the connection to fail when using a DB in Belgium
			// "loc":                  "Africa/Nairobi",
			"allowNativePasswords": "true",
			"multiStatements":      "true",
		},
	}

	// Open database connection
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %v", err)
	}

	// Set database connection pool limits to prevent stale connections and connection leaks
	db.SetMaxOpenConns(25)                 // Maximum number of open connections
	db.SetMaxIdleConns(25)                 // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum amount of time a connection may be reused

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
