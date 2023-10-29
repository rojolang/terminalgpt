// Package main is the entry point of the application. It handles the interaction with the user and the generation of completions.
//
// The main function reads the configuration file, initializes the GPT model, and enters a loop where it prompts the user for a message, generates a completion, and prints the completion.
//
// Functions:
// - main: The entry point of the application.
package main

import (
	"bufio"
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

	if *configFlag {
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
	gpt := gpt.New(cfg.ModelName, cfg.Temperature, cfg.MaxTokens, cfg.TopP, cfg.FrequencyPenalty, cfg.PresencePenalty, cfg.Stream, cfg.PrintStats, cfg.AuthorizationKey)

	reader := bufio.NewReader(os.Stdin)

	// Interactively update the configuration.
	for {
		fmt.Println("\nEnter your message, or 'e' to exit:")
		answer, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Failed to read user input: %v\n", err)
			continue
		}
		answer = strings.TrimSpace(answer)

		if answer == "e" {
			break
		}

		response, err := gpt.GenerateCompletion(answer)
		if err != nil {
			fmt.Printf("Failed to generate completion: %v\n", err)
			continue
		}

		fmt.Println(response)
	}
}
