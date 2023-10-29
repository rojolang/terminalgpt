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
	TempConfigFile   = "config_temp.json"
)

type Config struct {
	ModelName        string  `json:"model"`
	Temperature      float64 `json:"temperature"`
	MaxTotalTokens   int     `json:"max_total_tokens"`
	MaxTokens        int     `json:"max_tokens"`
	TopP             float64 `json:"top_p"`
	FrequencyPenalty float64 `json:"frequency_penalty"`
	PresencePenalty  float64 `json:"presence_penalty"`
	Stream           bool    `json:"stream"`
	PrintStats       bool    `json:"print_stats"`
	History          bool    `json:"history"`
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

func LoadConfig(file string) (Config, error) {
	var config Config
	configFile, err := os.Open(file)
	if err != nil {
		return config, fmt.Errorf("Failed to open config file: %v", err) // Add error context
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		return config, fmt.Errorf("Failed to parse config file: %v", err) // Add error context
	}

	return config, nil
}

func SaveConfig(file string, config Config) error {
	configFile, err := os.Create(TempConfigFile)
	if err != nil {
		return fmt.Errorf("Failed to create temp config file: %v", err) // Add error context
	}
	defer configFile.Close()
	jsonWriter := json.NewEncoder(configFile)
	jsonWriter.SetIndent("", "\t")
	err = jsonWriter.Encode(&config)
	if err != nil {
		return fmt.Errorf("Failed to encode config: %v", err) // Add error context
	}

	err = os.Rename(TempConfigFile, file)
	if err != nil {
		return fmt.Errorf("Failed to rename temp config file: %v", err) // Add error context
	}
	return nil
}
func GetDefaultConfig() Config {
	return Config{
		ModelName:        "gpt-4",
		Temperature:      0.50,
		MaxTotalTokens:   2000,
		MaxTokens:        500,
		TopP:             1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		Stream:           true,
		PrintStats:       true,
		History:          true,
		SystemMessage:    "You are a useful assistant, your input is streamed into command line regarding coding and terminal questions for a user that uses macosx and codes in python and go and uses aws frequently.",
		AuthorizationKey: os.Getenv("OPENAI_SECRET_KEY"),
	}
}

func InteractiveConfigure() error {
	config, err := LoadConfig(ConfigFile)
	if err != nil {
		fmt.Println("Failed to load config file, using default settings.")
		config = GetDefaultConfig()
	}

	err = interactiveUpdate(&config)
	if err != nil {
		return fmt.Errorf("Failed to update configuration interactively: %v", err)
	}

	err = SaveConfig(ConfigFile, config)
	if err != nil {
		return fmt.Errorf("Failed to save updated config file: %v", err)
	}

	return nil
}

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

func interactiveUpdate(config *Config) error {
	reader := bufio.NewReader(os.Stdin)

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
	}

	return nil
}

func printCurrentConfig(config *Config) {
	fmt.Println("\nCurrent configuration:")
	fmt.Printf("1. Model: %s\n", config.ModelName)
	fmt.Printf("2. Temperature: %f\n", config.Temperature)
	fmt.Printf("3. Max tokens: %d\n", config.MaxTokens)
	fmt.Printf("4. Max response tokens: %d\n", config.MaxTotalTokens)
	fmt.Printf("5. Top P: %f\n", config.TopP)
	fmt.Printf("6. Frequency penalty: %f\n", config.FrequencyPenalty)
	fmt.Printf("7. Presence penalty: %f\n", config.PresencePenalty)
	fmt.Printf("8. Stream: %t\n", config.Stream)
	fmt.Printf("9. Print stats: %t\n", config.PrintStats)
	fmt.Printf("10. History: %t\n", config.History)
	fmt.Printf("11. System message: %s\n", config.SystemMessage)
	if len(config.AuthorizationKey) >= 4 {
		fmt.Printf("12. Authorization key: ****%s\n", config.AuthorizationKey[len(config.AuthorizationKey)-4:])
	} else {
		fmt.Println("12. Authorization key is missing.")
	}
}

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
		updateErr = updateConfig(reader, "Enter the max response tokens:", func(input string) error {
			maxResponseTokens, err := strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("invalid max response tokens value: %v", err)
			}
			config.MaxTotalTokens = maxResponseTokens
			return nil
		})
	case "5":
		updateErr = updateConfig(reader, "Enter the Top P:", func(input string) error {
			topP, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return fmt.Errorf("invalid Top P value: %v", err)
			}
			config.TopP = topP
			return nil
		})
	case "6":
		updateErr = updateConfig(reader, "Enter the frequency penalty:", func(input string) error {
			frequencyPenalty, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return fmt.Errorf("invalid frequency penalty value: %v", err)
			}
			config.FrequencyPenalty = frequencyPenalty
			return nil
		})
	case "7":
		updateErr = updateConfig(reader, "Enter the presence penalty:", func(input string) error {
			presencePenalty, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return fmt.Errorf("invalid presence penalty value: %v", err)
			}
			config.PresencePenalty = presencePenalty
			return nil
		})
	case "8":
		updateErr = updateConfig(reader, "Enter the stream (true/false):", func(input string) error {
			stream, err := strconv.ParseBool(input)
			if err != nil {
				return fmt.Errorf("invalid stream value: %v", err)
			}
			config.Stream = stream
			return nil
		})
	case "9":
		updateErr = updateConfig(reader, "Enter the print stats (true/false):", func(input string) error {
			printStats, err := strconv.ParseBool(input)
			if err != nil {
				return fmt.Errorf("invalid print stats value: %v", err)
			}
			config.PrintStats = printStats
			return nil
		})
	case "10":
		updateErr = updateConfig(reader, "Keep History? (true/false):", func(input string) error {
			history, err := strconv.ParseBool(input)
			if err != nil {
				return fmt.Errorf("invalid history value: %v", err)
			}
			config.History = history
			return nil
		})
	case "11":
		updateErr = updateConfig(reader, "Enter the system message:", func(input string) error {
			if input == "" {
				return fmt.Errorf("system message cannot be empty")
			}
			config.SystemMessage = input
			return nil
		})
	case "12":
		updateErr = updateConfig(reader, "Enter the authorization key:", func(input string) error {
			if input == "" {
				return fmt.Errorf("authorization key cannot be empty")
			}
			config.AuthorizationKey = input
			return nil
		})
	default:
		fmt.Println("Invalid option. Please enter a number between 1 and 12, or 'e' to exit.")
	}

	return updateErr
}
