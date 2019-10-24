package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/nordicdyno/processchief"
)

func main() {
	chief := processchief.NewChief()
	srv := processchief.NewControlServer(chief)

	ctx, _ := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		err := srv.Start(ctx)
		chief.StopAll()

		if err != nil && err.Error() != "http: Server closed" {
			log.Fatal("start error:", err)
		}
	}()

	<-c
	srv.Stop(ctx)
	fmt.Println("StopAll at the end of main")
	chief.StopAll()
}
