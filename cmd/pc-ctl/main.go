package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/nordicdyno/processchief/pb"
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
	client := pb.NewControlAPIClient(conn)
	ctx := context.Background()

	nope := &pb.Nope{}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "processes list",
		Run: func(_ *cobra.Command, _ []string) {
			all, err := client.AllProcesses(ctx, nope)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("processes:")
			for _, status := range all.Statuses {
				fmt.Println(status.String())
			}
		},
	}

	haltCmd := &cobra.Command{
		Use:   "halt",
		Short: "stops supervisor",
		Run: func(_ *cobra.Command, _ []string) {
			res, err := client.Halt(ctx, nope)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(res)
		},
	}

	var (
		processName string
		cmdCommand  string
		workDir     string
		loggerCmd   string
		envVars []string
	)
	nameFlag := func(cmd *cobra.Command) {
		cmd.Flags().StringVarP(&processName, "name", "n",
			"", "process name")
	}
	commonFlags := func(cmd *cobra.Command) {
		nameFlag(cmd)
		cmd.Flags().StringVarP(&cmdCommand, "command", "c",
			"", "command line to run")
		cmd.Flags().StringVarP(&workDir, "work-dir", "w",
			"", "working directory")
		cmd.Flags().StringVarP(&loggerCmd, "logger", "l",
			"", "command line to output log")
		cmd.Flags().StringSliceVarP(&envVars, "env", "e",
			nil, "environment variables")
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "add process",
		Run: func(_ *cobra.Command, _ []string) {
			p := &pb.SetProc{
				Process: &pb.Process{
					Name:              processName,
					CommandLine:       cmdCommand,
					LoggerCommandLine: loggerCmd,
				},
				Env: &pb.ProcEnv{
					EnvVars:              envVars,
					WorkingDir:        workDir,
				},
			}
			res, err := client.AddProcess(ctx, p)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(res)
		},
	}
	commonFlags(addCmd)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "update service",
		Run: func(_ *cobra.Command, _ []string) {
			newSrv := &pb.SetProc{
				Process: &pb.Process{
					Name:              processName,
					CommandLine:       cmdCommand,
					LoggerCommandLine: loggerCmd,
				},
				Env: &pb.ProcEnv{
					EnvVars:              envVars,
					WorkingDir:        workDir,
				},
			}
			res, err := client.UpdateProcess(ctx, newSrv)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(res)
		},
	}
	commonFlags(updateCmd)

	delCmd := &cobra.Command{
		Use:   "delete",
		Short: "delete process",
		Run: func(_ *cobra.Command, _ []string) {
			name := &pb.ProcName{Name: processName}
			res, err := client.DeleteProcess(ctx, name)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(res)
		},
	}
	nameFlag(delCmd)

	var signal int32
	signalFlag := func(cmd *cobra.Command) {
		cmd.Flags().Int32VarP(&signal, "signal", "s", int32(syscall.SIGKILL), "signal number")
	}
	sigCmd := &cobra.Command{
		Use:   "signal",
		Short: "send signal to process",
		Run: func(_ *cobra.Command, _ []string) {
			sig := &pb.Signal{
				Name: processName,
				Signal: signal,
			}
			res, err := client.ProcessSignal(ctx, sig)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(res)
		},
	}
	nameFlag(sigCmd)
	signalFlag(sigCmd)
	logSigCmd := &cobra.Command{
		Use:   "log-signal",
		Short: "send signal to process logger",
		Run: func(_ *cobra.Command, _ []string) {
			sig := &pb.Signal{
				Name: processName,
				Signal: signal,
			}
			res, err := client.LoggerSignal(ctx, sig)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(res)
		},
	}
	nameFlag(logSigCmd)
	signalFlag(logSigCmd)


	rootCmd.AddCommand(
		listCmd, haltCmd, addCmd, updateCmd, delCmd, sigCmd, logSigCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
