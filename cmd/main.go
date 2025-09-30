package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
	"web-server/internal/cache"
	"web-server/internal/config"
	"web-server/internal/db"
	"web-server/internal/handler"
	"web-server/internal/logger"
	"web-server/internal/repository"
	"web-server/internal/service"

	"github.com/gorilla/mux"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pg, err := db.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		log.Error("pgx connect", "err", err)
		os.Exit(1)
	}
	defer pg.Close()

	rdb := cache.New(cfg)

	repo := repository.NewRepository(pg)
	userSvc := service.NewUserService(repo)
	uh := handler.NewUserHandler(log, cfg, userSvc)

	docRepo := repository.NewDocumentRepository(pg)
	docSvc := service.NewDocumentService(docRepo, rdb, time.Duration(cfg.Security.TokenTTLSeconds)*time.Millisecond, "uploads")
	docH := handler.NewDocumentHandler(docSvc, userSvc)

	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/register", uh.Register).Methods("POST")
	api.HandleFunc("/auth", uh.Auth).Methods("POST")

	api.HandleFunc("/auth", uh.Logout).Methods("DELETE")

	r.HandleFunc("/api/docs", docH.ListDocs).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/api/docs", docH.UploadDoc).Methods(http.MethodPost)
	r.HandleFunc("/api/docs/{id}", docH.GetDoc).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/api/docs/{id}", docH.DeleteDoc).Methods(http.MethodDelete)

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info("starting server", "addr", cfg.Server.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server error", "err", err)
	}
}
