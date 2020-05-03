package docdbuild

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/shounakdatta/DoCD/internal/docdtypes"
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

// Global variables
var (
	cmdSlice        []cmdReference
	installServices bool = true
)

// BuildCmd : Installs dependencies and builds all services
func BuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Installs dependencies and builds all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Register signal handlers
			signalChan := make(chan os.Signal, 1)
			exitChan := make(chan int)
			signalHandler(signalChan, exitChan)

			// Get config file
			config := docdtypes.ReadConfig()

			// Get working directory
			dir, _ := os.Getwd()

			// Make log directory
			os.MkdirAll(dir+"\\logs", os.ModePerm)

			// Check if Admin
			isAdmin, _ := checkAdmin()
			if !isAdmin {
				color.Yellow(
					"Warning: You are not running DoCD as an administrator.\n" +
						"Service installations will be skipped.")
				installServices = false
			}

			// Initialize services
			InitializeServices(config)

			// Initialize webhook
			http.HandleFunc("/github-push-master", autoDeploy)
			go http.ListenAndServe(":6000", nil)

			// Wait for exit signal
			_ = <-exitChan

			// Kill all services in their respective terminals
			fmt.Println("Terminating services...")
			for _, cmdRef := range cmdSlice {
				cmdRef.Cmd.Process.Kill()
				cmdRef.Cmd.Process.Wait()
				cmdRef.LogFile.Close()
			}

			return nil
		},
	}
}

// InitializeServices : Installs service dependecies and launches services
func InitializeServices(config docdtypes.Config) {
	// Get working directory
	dir, _ := os.Getwd()
	for _, service := range config.Services {
		// Create log file
		logFile, err := os.Create(service.LogFilePath)
		if err != nil {
			panic(err)
		}

		// Install services and service dependencies
		installService(service, config.BasePackageManager)
		refreshEnv()
		installServiceDependencies(service, dir)

		// Build service
		startService(service, dir, logFile)
		color.Cyan("All services started")
		color.Cyan("To terminate session, press CTRL+C")

	}
}
