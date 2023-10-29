// Package main is the entry point of the application. It handles the interaction with the user and the generation of completions.
//
// The main function reads the configuration file, initializes the GPT model, and enters a loop where it prompts the user for a message, generates a completion, and prints the completion.
//
// Functions:
// - main: The entry point of the application.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rojolang/terminalgpt/config"
	"github.com/rojolang/terminalgpt/gpt"
)

// main is the entry point of the application. It reads the configuration file and initializes the GPT model.
// It then enters a loop where it prompts the user for a message, generates a completion using the GPT model, and prints the completion.
// The loop continues until the user enters 'e' to exit.
func main() {
	configFlag := flag.Bool("config", false, "Configure settings")
	flag.Parse()

	// Check if the configuration file exists.
	_, err := os.Stat(config.ConfigFile)

	// If the config file does not exist or the --config flag is provided, run the interactive configuration.
	if os.IsNotExist(err) || *configFlag {
		err := config.InteractiveConfigure()
		if err != nil {
			fmt.Println("Failed to configure settings.")
			return
		}
	}

	// Load the configuration file.
	cfg, err := config.LoadConfig(config.ConfigFile)
	if err != nil {
		fmt.Println("Failed to load config file, using default settings.")
		// If the configuration file cannot be loaded, use the default settings.
		cfg = config.GetDefaultConfig()
		err = config.SaveConfig(config.ConfigFile, cfg)
		if err != nil {
			fmt.Println("Failed to save default config file.")
		}
	}

	// Configure the GPT model.
	g := gpt.New(cfg)

	// Get all non-flag arguments as the user message.
	userMessage := strings.Join(flag.Args(), " ")

	response, err := g.GenerateCompletion(userMessage)
	if err != nil {
		fmt.Printf("Failed to generate completion: %v\n", err)
		return
	}

	fmt.Println(response)
}
