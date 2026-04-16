package main

import (
	"botframework/api"
	"botframework/engine"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	manager := engine.NewSmartManager()

	err := manager.Engine.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	defer func() {
		if err := manager.Engine.Stop(); err != nil {
			log.Printf("Error stopping engine: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", api.HandleHealth(manager.Engine))
	mux.HandleFunc("/v1/models", api.HandleModels(manager.Engine))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("📥 Request: %s %s\n", r.Method, r.URL.Path)
		manager.Engine.ProxyRequest(w, r)
	})

	port := "8080"
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("manager shutdown error: %v", err)
		}
	}()

	fmt.Printf("🌟 BotFramework Manager listening on :%s\n", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
