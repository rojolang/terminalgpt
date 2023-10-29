// Package config provides a set of functions and types to handle the configuration
// of the application. It includes functions to load, save, and interactively update the configuration.
// It uses a Config struct to hold all the configuration parameters. The package also
// provides a function to interactively configure the settings.
//
// Variables:
// - configFile: The name of the configuration file.
// - startTime: The time at which the application started.
// - completionAPIURL: The URL of the OpenAI API.
// - systemMessage: The system message to be sent to the OpenAI API.
//
// Functions:
// - LoadConfig: Loads the configuration from a file.
// - SaveConfig: Saves the configuration to a file.
// - GetDefaultConfig: Returns a Config struct with default settings.
// - updateConfig: Updates a configuration setting based on user input.
// - InteractiveConfigure: Allows the user to interactively configure the settings.
// - interactiveUpdate: Interactively updates the configuration.
// - printCurrentConfig: Prints the current configuration.
// - updateConfigOption: Updates a configuration option based on user input.
package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ConfigFile       = "config.json"
	StartTime        = time.Now()
	CompletionAPIURL = "https://api.openai.com/v1/chat/completions"
	SystemMessage    = "You are a useful assistant, your input is streamed into command line regarding coding and terminal questions for a user that uses macosx and codes in python and go and uses aws frequently."
)

// Config struct holds all the configuration details
type Config struct {
	ModelName        string  `json:"model"`
	Temperature      float64 `json:"temperature"`
	MaxTokens        int     `json:"max_tokens"`
	TopP             float64 `json:"top_p"`
	FrequencyPenalty float64 `json:"frequency_penalty"`
	PresencePenalty  float64 `json:"presence_penalty"`
	Stream           bool    `json:"stream"`
	PrintStats       bool    `json:"print_stats"`
	AuthorizationKey string  `json:"authorization_key"`
	SystemMessage    string  `json:"system_message"`
}

type Event struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Delta        Message `json:"delta"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LoadConfig function loads the configuration from a file.
// It takes a filename as input and returns a Config struct and an error.
func LoadConfig(file string) (Config, error) {
	var config Config
	configFile, err := os.Open(file)
	if err != nil {
		return config, err
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}

// SaveConfig function saves the configuration to a file.
// It takes a filename and a Config struct as input and returns an error.
func SaveConfig(file string, config Config) error {
	configFile, err := os.Create(file)
	if err != nil {
		return err
	}
	defer configFile.Close()
	jsonWriter := json.NewEncoder(configFile)
	jsonWriter.SetIndent("", "\t")
	jsonWriter.Encode(&config)
	return nil
}

// GetDefaultConfig function returns a Config struct with default settings.
func GetDefaultConfig() Config {
	return Config{
		ModelName:        "gpt-4",
		Temperature:      0.50,
		MaxTokens:        2000,
		TopP:             1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		Stream:           true,
		PrintStats:       true,
		AuthorizationKey: os.Getenv("OPENAI_SECRET_KEY"),
	}
}

// updateConfig function updates a configuration setting based on user input.
// It takes a bufio.Reader, a prompt string, and a function to update the configuration setting.
// It returns an error.
func updateConfig(reader *bufio.Reader, prompt string, updateFunc func(string) error) error {
	fmt.Println(prompt)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %v", err)
	}

	err = updateFunc(strings.TrimSpace(answer))
	if err != nil {
		return fmt.Errorf("failed to update configuration: %v", err)
	}

	return nil
}

// InteractiveConfigure function allows the user to interactively configure the settings.
func InteractiveConfigure() error {
	// Load the configuration file.
	config, err := LoadConfig(ConfigFile)
	if err != nil {
		fmt.Println("Failed to load config file, using default settings.")
		// If the configuration file cannot be loaded, use the default settings.
		config = GetDefaultConfig()
		err = SaveConfig(ConfigFile, config)
		if err != nil {
			return fmt.Errorf("Failed to save default config file: %v", err)
		}
	}

	err = interactiveUpdate(&config)
	if err != nil {
		return fmt.Errorf("Failed to update configuration interactively: %v", err)
	}

	return nil
}

// interactiveUpdate function interactively updates the configuration.
func interactiveUpdate(config *Config) error {
	reader := bufio.NewReader(os.Stdin)

	// Interactively update the configuration.
	for {
		printCurrentConfig(config)

		fmt.Println("\nEnter the number of the setting you want to change, or 'e' to exit:")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("Failed to read user input: %v", err)
		}
		answer = strings.TrimSpace(answer)

		if answer == "e" {
			break
		}

		err = updateConfigOption(reader, answer, config)
		if err != nil {
			fmt.Printf("Failed to update configuration: %v\n", err)
			continue
		}

		err = SaveConfig(ConfigFile, *config)
		if err != nil {
			return fmt.Errorf("Failed to save updated config file: %v", err)
		}
	}

	return nil
}

// printCurrentConfig function prints the current configuration.
func printCurrentConfig(config *Config) {
	fmt.Println("\nCurrent configuration:")
	fmt.Printf("1. Model: %s\n", config.ModelName)
	fmt.Printf("2. Temperature: %f\n", config.Temperature)
	fmt.Printf("3. Max tokens: %d\n", config.MaxTokens)
	fmt.Printf("4. Top P: %f\n", config.TopP)
	fmt.Printf("5. Frequency penalty: %f\n", config.FrequencyPenalty)
	fmt.Printf("6. Presence penalty: %f\n", config.PresencePenalty)
	fmt.Printf("7. Stream: %t\n", config.Stream)
	fmt.Printf("8. Print stats: %t\n", config.PrintStats)
	if len(config.AuthorizationKey) >= 4 {
		fmt.Printf("9. Authorization key: ****%s\n", config.AuthorizationKey[len(config.AuthorizationKey)-4:])
	} else {
		fmt.Println("9. Authorization key is missing.")
	}
}

// updateConfigOption function updates a configuration option based on user input.
func updateConfigOption(reader *bufio.Reader, answer string, config *Config) error {
	var updateErr error
	switch answer {
	case "1":
		updateErr = updateConfig(reader, "Enter the model name:", func(input string) error {
			if input == "" {
				return fmt.Errorf("model name cannot be empty")
			}
			config.ModelName = input
			return nil
		})
	case "2":
		updateErr = updateConfig(reader, "Enter the temperature:", func(input string) error {
			temp, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return fmt.Errorf("invalid temperature value: %v", err)
			}
			config.Temperature = temp
			return nil
		})
	case "3":
		updateErr = updateConfig(reader, "Enter the max tokens:", func(input string) error {
			maxTokens, err := strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("invalid max tokens value: %v", err)
			}
			config.MaxTokens = maxTokens
			return nil
		})
	case "4":
		updateErr = updateConfig(reader, "Enter the Top P:", func(input string) error {
			topP, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return fmt.Errorf("invalid Top P value: %v", err)
			}
			config.TopP = topP
			return nil
		})
	case "5":
		updateErr = updateConfig(reader, "Enter the frequency penalty:", func(input string) error {
			frequencyPenalty, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return fmt.Errorf("invalid frequency penalty value: %v", err)
			}
			config.FrequencyPenalty = frequencyPenalty
			return nil
		})
	case "6":
		updateErr = updateConfig(reader, "Enter the presence penalty:", func(input string) error {
			presencePenalty, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return fmt.Errorf("invalid presence penalty value: %v", err)
			}
			config.PresencePenalty = presencePenalty
			return nil
		})
	case "7":
		updateErr = updateConfig(reader, "Enter the stream (true/false):", func(input string) error {
			stream, err := strconv.ParseBool(input)
			if err != nil {
				return fmt.Errorf("invalid stream value: %v", err)
			}
			config.Stream = stream
			return nil
		})
	case "8":
		updateErr = updateConfig(reader, "Enter the print stats (true/false):", func(input string) error {
			printStats, err := strconv.ParseBool(input)
			if err != nil {
				return fmt.Errorf("invalid print stats value: %v", err)
			}
			config.PrintStats = printStats
			return nil
		})
	case "9":
		updateErr = updateConfig(reader, "Enter the authorization key:", func(input string) error {
			if input == "" {
				return fmt.Errorf("authorization key cannot be empty")
			}
			config.AuthorizationKey = input
			return nil
		})
	default:
		fmt.Println("Invalid option. Please enter a number between 1 and 9, or 'e' to exit.")
	}

	return updateErr
}
