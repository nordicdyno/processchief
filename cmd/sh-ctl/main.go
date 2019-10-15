package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/nordicdyno/simple-hypervisor/pb"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "superctl",
		Short: "super client",
	}

	serverAddr := "localhost:5556"
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("grpc connection fail: %v", err)
	}
	client := pb.NewServicesAPIClient(conn)
	ctx := context.Background()

	nope := &pb.Nope{}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "get services list",
		Run: func(_ *cobra.Command, _ []string) {
			all, err := client.AllServices(ctx, nope)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("services list:")
			for _, srv := range all.Service {
				fmt.Println(srv.String())
			}
		},
	}

	var (
		cmdName    string
		cmdCommand string
	)
	commonFlags := func(cmd *cobra.Command) {
		cmd.Flags().StringVarP(&cmdName, "name", "n",
			"", "service name")
		cmd.Flags().StringVarP(&cmdCommand, "command", "c",
			"", "command line to run")
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "add service",
		Run: func(_ *cobra.Command, _ []string) {
			newSrv := &pb.NewService{Name: cmdName, Commandline: cmdCommand}
			_, err := client.AddService(ctx, newSrv)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	commonFlags(addCmd)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "update service",
		Run: func(_ *cobra.Command, _ []string) {
			newSrv := &pb.NewService{Name: cmdName, Commandline: cmdCommand}
			_, err := client.UpdateService(ctx, newSrv)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	commonFlags(updateCmd)

	rootCmd.AddCommand(listCmd, addCmd, updateCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
