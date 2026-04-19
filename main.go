package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
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
	store := NewStore()

	srv := &Server{
		Store:     store,
		Templates: templates,
		BoxTypes:  boxTypes,
		Logger:    logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.HandleIndex)
	mux.HandleFunc("/provision", srv.HandleProvision)
	mux.HandleFunc("/consume", srv.HandleConsume)
	mux.HandleFunc("/force-replace", srv.HandleForceReplace)
	mux.HandleFunc("/status", srv.HandleStatus)
	mux.HandleFunc("/user-data", srv.HandleUserData)
	mux.HandleFunc("/meta-data", srv.HandleMetaData)

	addr := "0.0.0.0:8080"
	log.Printf("cloud-init portal listening on http://%s", addr)
	log.Printf("loaded templates: %v", TemplateNames(templates))
	log.Printf("available box types: %v", BoxTypeNames(boxTypes))

	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
