package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chokey2nv/ultainfinity/client"
	"github.com/chokey2nv/ultainfinity/node"
	"github.com/urfave/cli"
)

// Block represents a block in the blockchain.

func main() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Create a channel to receive termination signals
	app := &cli.App{
		UseShortOptionHandling: true,
		Commands: []cli.Command{
			{
				Name:  "node",
				Usage: "start blockchain server",
				Action: func(cCtx *cli.Context) error {
					node.StartServer()
					return nil
				},
			},
			{
				Name:  "client",
				Usage: "start client server",
				Action: func(cCtx *cli.Context) error {
					client.StartServer()
					return nil
				},
			},
			{
				Name:  "all",
				Usage: "start node & client servers",
				Action: func(cCtx *cli.Context) error {
					go client.StartServer()
					go node.StartServer()
					// Wait for termination signal
					<-signalCh

					// Perform cleanup tasks and shutdown
					log.Println("Termination signal received. Performing cleanup.")

					// Gracefully stop the client server
					client.StopServer()

					// Gracefully stop the node server
					node.StopServer()

					// Additional cleanup tasks
					// ...

					log.Println("Cleanup completed. Exiting.")

					return nil
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
