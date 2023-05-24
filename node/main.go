package node

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/chokey2nv/ultainfinity/node/app"
)

// Block represents a block in the blockchain.
var (
	server      *http.Server
	application *app.Application
	err         error
)

/**
Start server starts blockchain (node) server,
and gracefully shutdown when interrupted
* it starts server from saved file (if any),
and before shutdown saves blockchain data to file (blockchain.json)
*/
func StartServer(port int64) {
	if port == 0 {
		port = 8000
	}
	application, err = app.NewApplication()
	if err != nil {
		log.Fatalf("new application: %v", err)
	}

	server = &http.Server{
		Addr:         ":" + strconv.FormatInt(port, 10),
		Handler:      application.Router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("Starting server (node) on " + server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Save the blockchain to file
	if err := application.SaveApplication(); err != nil {
		log.Fatal("save application:", err)
	}

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server shutdown error:", err)
	}

	log.Println("Server (node) gracefully stopped")
}

//s tops server when running multiple nodes or started together with client server
// in a single command
func StopServer() {
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Save the blockchain to file
	if err := application.SaveApplication(); err != nil {
		log.Fatal("save application:", err)
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server shutdown error:", err)
	}

	log.Println("Server (node) gracefully stopped")
}
