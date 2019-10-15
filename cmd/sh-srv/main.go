package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	supervisor "github.com/nordicdyno/simple-hypervisor"
)

func main() {
	super := supervisor.NewSupervisor()
	srv := supervisor.NewControlServer(super)

	ctx, _ := context.WithCancel(context.Background())
	// fin := make(chan struct{}, 1)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		err := srv.Start(ctx)
		if err != nil {
			log.Fatal("start error: ", err)
		}
	}()

	<-c
	srv.Stop(ctx)
	super.StopAll()
}
