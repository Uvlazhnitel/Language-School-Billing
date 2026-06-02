package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"langschool/internal/backend"
	appruntime "langschool/internal/runtime"
	"langschool/internal/web"
)

func main() {
	ctx := context.Background()
	cfg := appruntime.LoadConfig(appruntime.UserHome())
	rt, err := appruntime.Start(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer rt.Close()

	handler := web.NewHandler(backend.New(rt))
	server := &http.Server{
		Addr:              listenAddr(),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}

func listenAddr() string {
	if addr := strings.TrimSpace(os.Getenv("ADDR")); addr != "" {
		return addr
	}
	host := strings.TrimSpace(os.Getenv("HOST"))
	port := strings.TrimSpace(os.Getenv("PORT"))
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "8080"
	}
	return host + ":" + port
}
