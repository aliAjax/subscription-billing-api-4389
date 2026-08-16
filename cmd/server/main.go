package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subscription-billing-api/internal/subscription/handler"
	"subscription-billing-api/internal/subscription/repository"
	"subscription-billing-api/internal/subscription/service"
	"subscription-billing-api/pkg/config"
	"subscription-billing-api/pkg/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	repo, err := repository.NewJSONRepository(cfg.DataFile)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}

	subscriptionService := service.New(repo)
	subscriptionHandler := handler.New(subscriptionService)

	mux := http.NewServeMux()
	subscriptionHandler.RegisterRoutes(mux)

	rootHandler := middleware.Recover(middleware.Logging(mux))
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           rootHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("subscription billing API listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
