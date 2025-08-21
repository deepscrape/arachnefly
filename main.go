package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deepscrape/arachnefly/domain"
	database "github.com/deepscrape/arachnefly/infrastructure/db"
	router "github.com/deepscrape/arachnefly/interfaces/controller"
	"github.com/deepscrape/arachnefly/interfaces/handlers"
	"github.com/deepscrape/arachnefly/interfaces/repository"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Define a struct to hold the query parameters
type DeployQuery struct {
	Clone    bool   `query:"clone"`
	MasterID string `query:"master_id"`
}

var flyApiToken string
var flyApp string
var flyApiUrl = "https://api.machines.dev/v1"

const addr = ":9090"

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system env variables")
	}

	flyApiToken = os.Getenv("FLY_API_TOKEN")
	flyApp = os.Getenv("FLY_APP")

}

func main() {
	// Initialize standard Go html template engine
	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
		Prefork: true, // Enable prefork mode for better performance
		// Concurrency:    256 * 1024, // Set the desired concurrency level
		JSONEncoder:    json.Marshal,
		JSONDecoder:    json.Unmarshal,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		BodyLimit:      4 * 1024 * 1024, // 4 MB
		ReadBufferSize: 16 * 1024,       // 16 KB, or // 4 KB
		Views:          engine,          // Set View Engine
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "https://localhost:4200, https://deepscrape.web.app, https://deepscrape.dev, http://localhost:5000, http://127.0.0.1:8081", //"https://arachnefly.com, https://www.arachnefly.com",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-CSRF-Token",                                              // "Content-Type, Accept, Authorization, X-Requested-With, X-CSRF-Token",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	FirebaseDBManager, err := initializeFirebaseManager( /* cfg, logger */ )
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	// defer FirebaseDBManager.Close()

	// initialize Firestore repository
	firestoreRepo := repository.NewFirestore(FirebaseDBManager, flyApiToken /* , cfg, logger */)

	// Initialize Handlers
	authHandler := handlers.NewHandlers(firestoreRepo, flyApiToken, flyApiUrl, flyApp)

	// Setup routes
	router.SetupRoutes(app, authHandler, FirebaseDBManager.AuthClient())
	// Setup Non-Accessible routes
	router.SetupNonAccessibleRoutes(app)

	// listener, err := reuseport.Listen("tcp4", "0.0.0.0"+addr)
	// if err != nil {
	// 	log.Fatalf("Failed to listen on port 3401 with SO_REUSEPORT: %v", err)
	// }
	// defer listener.Close()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Channel to listen for interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// go fly.WalkResponse()

	// Start Metrics server
	// go func() {
	// 	slog.Info("serving metrics", slog.String("addr", addr))
	// 	// http.Handle("/metrics", prometheus.Middleware())
	// 	// if err := http.Serve(listener, nil); err != nil {
	// 	// 	log.Fatal(err)
	// 	// }
	// }()

	// Start arachnefly MicroService
	go func() {
		log.Println("Starting arachnefly microservice on", os.Getenv("PORT"))
		if err := app.Listen(":" + os.Getenv("PORT")); err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Shutdown Fiber Server App
	if err := app.Shutdown(); err != nil {
		log.Fatal("Failed to shutdown server", zap.Error(err))
	}

	log.Println("arachnefly MicroService gracefully stopped")

	// Wait for an interrupt signal
	<-sigCh

	log.Println("Shutting metrics Server...")

}

func initializeFirebaseManager() (*database.FirebaseManager, error) {
	var firebaseManager *database.FirebaseManager

	opts := &domain.FirebaseManagerOpts{
		CredsFile:    "",
		DatabaseName: "easyscrape",   // cfg.DatabaseKeyspace,
		ProjectID:    "libnet-d76db", // default project ID
	}

	// Check for Firestore emulator
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost != "" {
		// When using emulator, credentials are not required
		opts.CredsFile = "" // Ensure no creds file is used
	}

	var err error
	for i := 0; i < 3; i++ {

		firebaseManager, err = database.NewFirebaseClient(opts)
		if err == nil {
			break
		}

		// logger.Error("Failed to connect to Firestore, retrying... ", zap.Error(err))
		time.Sleep(2 * time.Second)
	}
	return firebaseManager, err
}
