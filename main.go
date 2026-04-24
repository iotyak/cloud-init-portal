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
)

func main() {
	cfg := LoadAppConfig()

	logger, err := NewProvisionLogger("./provision.log")
	if err != nil {
		log.Fatalf("failed to initialize provision logger: %v", err)
	}
	defer logger.Close()

	templates, err := LoadCloudInitTemplates("./templates")
	if err != nil {
		log.Fatalf("failed to load templates: %v", err)
	}

	boxTypes := DefaultBoxTypes()
	store, err := NewStoreWithPersistence(cfg.StateFile)
	if err != nil {
		log.Fatalf("failed to initialize state store: %v", err)
	}

	srv := &Server{
		Store:             store,
		Templates:         templates,
		BoxTypes:          boxTypes,
		Logger:            logger,
		PublicBaseURL:     cfg.PublicBaseURL,
		TrustProxyHeaders: cfg.TrustProxyHeaders,
		StatusLimiter:     newFixedWindowLimiter(cfg.StatusRateLimit),
		WriteLimiter:      newFixedWindowLimiter(cfg.WriteRateLimit),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.HandleIndex)
	mux.HandleFunc("/logs", srv.HandleLogsPage)
	mux.HandleFunc("/provision", srv.HandleProvision)
	mux.HandleFunc("/consume", srv.HandleConsume)
	mux.HandleFunc("/force-replace", srv.HandleForceReplace)
	mux.HandleFunc("/status", srv.HandleStatus)
	mux.HandleFunc("/api/logs", srv.HandleLogsAPI)
	mux.HandleFunc("/user-data", srv.HandleUserData)
	mux.HandleFunc("/meta-data", srv.HandleMetaData)

	addr := "0.0.0.0:8080"
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           middlewareChain(mux, srv),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("cloud-init portal listening on http://%s", addr)
	log.Printf("loaded templates: %v", TemplateNames(templates))
	log.Printf("available box types: %v", BoxTypeNames(boxTypes))
	if cfg.PublicBaseURL != "" {
		log.Printf("public base URL override enabled: %s", cfg.PublicBaseURL)
	}
	if cfg.StateFile != "" {
		log.Printf("state persistence enabled: %s", cfg.StateFile)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("received signal %s, shutting down", sig)
	case err := <-errCh:
		if err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		os.Exit(1)
	}
}
