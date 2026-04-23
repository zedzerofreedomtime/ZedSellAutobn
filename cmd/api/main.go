package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zedsellauto/internal/app"
	"zedsellauto/internal/config"
)

func main() {
	config.LoadDotEnv(".env")
	cfg := config.Load()

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer application.Close()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           application.Router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
	}

	go func() {
		log.Printf("api listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
