package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"call-booking/internal/api"
	"call-booking/internal/db"
)

func main() {
	ctx := context.Background()

	// #region agent log - startup check
	log.Printf("[STARTUP] Starting API server...")
	log.Printf("[STARTUP] DATABASE_URL: %s", os.Getenv("DATABASE_URL"))
	log.Printf("[STARTUP] PORT: %s", os.Getenv("PORT"))
	// #endregion

	pool, err := db.NewPool(ctx)
	if err != nil {
		// #region agent log - db connect fail
		log.Fatalf("[STARTUP] Failed to connect to database: %v", err)
		// #endregion
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		// #region agent log - db ping fail
		log.Fatalf("[STARTUP] Failed to ping database: %v", err)
		// #endregion
	}

	// #region agent log - db ok
	log.Printf("[STARTUP] Database connected successfully")
	// #endregion

	// Run migrations with error handling for idempotent operations
	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		// Log error but don't fail - tables may already exist
		fmt.Printf("Migration warning: %v\n", err)
		fmt.Println("Continuing startup (tables should already exist)...")
	}

	// #region agent log - migrations ok
	log.Printf("[STARTUP] Migrations completed")
	// #endregion

	router := api.NewRouter(pool)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		fmt.Printf("Server starting on port %s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	fmt.Println("Server stopped")
}
