package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"saas-starter/api/internal/handler"
	"saas-starter/api/internal/service"
	"saas-starter/api/internal/store"
)

func main() {
	godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := env("DATABASE_URL_UNPOOLED", env("DATABASE_URL", "postgres://postgres:postgres@localhost:5544/app?sslmode=disable"))
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(5)
	db.SetConnMaxIdleTime(time.Minute)

	site := env("SITE_URL", "http://localhost:3100")
	slog.Info("mail config", "resend", os.Getenv("RESEND_API_KEY") != "", "site", site)
	if os.Getenv("INTERNAL_KEY") == "" {
		slog.Warn("INTERNAL_KEY is not set: rate limits can't tell visitors apart")
	}
	svc := service.New(store.New(db), service.Config{Paddle: paddleConfig(), OAuth: oauthConfig(),
		Mail:    service.ResendMailer{Key: os.Getenv("RESEND_API_KEY"), From: env("MAIL_FROM", service.App.Name+" <hello@example.com>"), ReplyTo: os.Getenv("MAIL_REPLY_TO")},
		SiteURL: site})

	srv := &http.Server{
		Addr:              ":" + env("PORT", "8180"),
		Handler:           handler.New(svc).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	go func() {
		slog.Info("api listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

func paddleConfig() service.Paddle {
	prices := map[string]string{}
	for _, plan := range service.PaidPlanIDs() {
		if id := os.Getenv("PADDLE_PRICE_" + strings.ToUpper(plan)); id != "" {
			prices[id] = plan
		}
	}
	secret := os.Getenv("PADDLE_WEBHOOK_SECRET")
	slog.Info("paddle config", "webhook", secret != "", "prices", len(prices), "portal", os.Getenv("PADDLE_API_KEY") != "")
	return service.Paddle{WebhookSecret: secret, PricePlans: prices, APIKey: os.Getenv("PADDLE_API_KEY")}
}

func oauthConfig() service.OAuth {
	c := service.OAuth{
		GoogleID: os.Getenv("GOOGLE_CLIENT_ID"), GoogleSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubID: os.Getenv("GITHUB_CLIENT_ID"), GitHubSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
	}
	slog.Info("oauth config", "google", c.GoogleID != "" && c.GoogleSecret != "", "github", c.GitHubID != "" && c.GitHubSecret != "")
	return c
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
