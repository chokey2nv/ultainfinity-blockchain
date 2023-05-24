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

func main() {
	// Create a channel to receive termination signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// cli app to separate applications or start both at same time, 
	//with flags to change port
	app := &cli.App{
		UseShortOptionHandling: true,
		Commands: []cli.Command{
			{
				Name:  "node",
				Usage: "start blockchain server",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "port", Usage: "set node port"},
				},
				Action: func(cCtx *cli.Context) error {
					port := cCtx.Int64("port")
					node.StartServer(port)
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
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "node-port", Usage: "set node port"},
				},
				Action: func(cCtx *cli.Context) error {
					port := cCtx.Int64("node-port")
					go client.StartServer()
					go node.StartServer(port)
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
