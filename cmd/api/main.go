package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
}

type application struct {
	config config
	logger *slog.Logger
}

func main() {
	var cfg config

	flag.StringVar(&cfg.env, "env", "development", "environment (development|production)")
	flag.IntVar(&cfg.port, "port", 4000, "HTTP port")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := application{
		config: cfg,
		logger: logger,
	}

	srv := http.Server{Addr: fmt.Sprintf(":%d", cfg.port), Handler: app.routes(), ErrorLog: slog.NewLogLogger(logger.Handler(),
		slog.LevelError), IdleTimeout: time.Minute, ReadTimeout: 5 *
		time.Second, WriteTimeout: 10 * time.Second}

	logger.Info("Starting Server on ", "Address", srv.Addr, "env", cfg.env)
	err := srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}
