package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

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

var (
	configFile       = "config.json"
	startTime        = time.Now()
	completionAPIURL = "https://api.openai.com/v1/chat/completions"
	systemMessage    = "You are a useful assistant, your input is streamed into command line regarding coding and terminal questions for a user that uses macosx and codes in python and go and uses aws frequently."
)

func createPayload(config Config, userMessage string) string {
	return fmt.Sprintf(`{
		"model": "%s",
		"messages": [
			{
				"role": "system",
				"content": "%s"
			},
			{
				"role": "user",
				"content": "%s"
			}
		],
		"temperature": %f,
		"max_tokens": %d,
		"top_p": %f,
		"frequency_penalty": %f,
		"presence_penalty": %f,
		"stream": %t
	}`, config.ModelName, systemMessage, userMessage, config.Temperature, config.MaxTokens, config.TopP, config.FrequencyPenalty, config.PresencePenalty, config.Stream)
}

func createRequest(config Config, userMessage string) (*http.Request, error) {
	payload := strings.NewReader(createPayload(config, userMessage))
	req, err := http.NewRequest("POST", completionAPIURL, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.AuthorizationKey))
	return req, nil
}

func handleResponse(resp *http.Response, tkm *tiktoken.Tiktoken, userTokens int, printStats bool) error {
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	assistantMsg := ""
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if strings.HasPrefix(line, "data: ") {
			jsonData := line[6:]
			if strings.TrimSpace(jsonData) == "[DONE]" {
				continue
			}
			var event Event
			err = json.Unmarshal([]byte(jsonData), &event)
			if err != nil {
				return err
			}
			for _, choice := range event.Choices {
				fmt.Print(choice.Delta.Content)
				assistantMsg += choice.Delta.Content
				if choice.FinishReason == "stop" {
					fmt.Println()
					if printStats {
						assistantTokens := len(tkm.Encode(assistantMsg, nil, nil))
						fmt.Printf("\n[🪙 %d ( 👤 %d | 💻 %d) ⏰ %ds]\n", userTokens+assistantTokens, userTokens, assistantTokens, int(time.Since(startTime).Seconds()))
					}
				}
			}
		}
	}
	return nil
}

func getTokenCount(text string, modelName string) (*tiktoken.Tiktoken, error) {
	tkm, err := tiktoken.EncodingForModel(modelName)
	if err != nil {
		return nil, fmt.Errorf("EncodingForModel: %v", err)
	}
	return tkm, nil
}

// readConfig function reads the configuration from a file.
func readConfig(file string) (Config, error) {
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

func writeConfig(file string, config Config) error {
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

func defaultConfig() Config {
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

// configure function allows the user to interactively configure the settings.
func configure() {
	// Read the configuration file.
	config, err := readConfig(configFile)
	if err != nil {
		fmt.Println("Failed to read config file, using default settings.")
		// If the configuration file cannot be read, use the default settings.
		config = defaultConfig()
		err = writeConfig(configFile, config)
		if err != nil {
			fmt.Println("Failed to write default config file.")
		}
	}

	reader := bufio.NewReader(os.Stdin)

	// Interactively update the configuration.
	for {
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

		fmt.Println("\nEnter the number of the setting you want to change, or 'e' to exit:")
		answer, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Failed to read user input: %v\n", err)
			continue
		}
		answer = strings.TrimSpace(answer)

		if answer == "e" {
			break
		}

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

		if updateErr != nil {
			fmt.Printf("Failed to update configuration: %v\n", updateErr)
			continue
		}

		err = writeConfig(configFile, config)
		if err != nil {
			fmt.Printf("Failed to write updated config file: %v\n", err)
		}
	}
}

func main() {
	configFlag := flag.Bool("config", false, "Configure settings")
	flag.Parse()

	_, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		config := defaultConfig()
		err := writeConfig(configFile, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write config file: %v\n", err)
			os.Exit(1)
		}
		absPath, _ := filepath.Abs(configFile)
		fmt.Printf("Configuration file created at %s with default values.\n", absPath)
		fmt.Println("You can edit this file directly with 'nano <filepath>', or run './gpt --config' to update the settings interactively.")
		fmt.Println("You can now run the program with './gpt <message>'.")
		return
	}

	if *configFlag {
		configure()
		return
	}

	config, err := readConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read config file: %v\n", err)
		os.Exit(1)
	}

	userMessage := strings.Join(flag.Args(), " ")
	tkm, err := getTokenCount(userMessage, config.ModelName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "getTokenCount error: %v\n", err)
		return
	}
	userTokens := len(tkm.Encode(userMessage, nil, nil))

	req, err := createRequest(config, userMessage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send request: %v\n", err)
		os.Exit(1)
	}
	if err := handleResponse(resp, tkm, userTokens, config.PrintStats); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to handle response: %v\n", err)
		os.Exit(1)
	}
}
