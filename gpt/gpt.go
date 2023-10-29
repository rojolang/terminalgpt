package gpt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkoukk/tiktoken-go"
	"github.com/rojolang/terminalgpt/config"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// GPT struct represents the GPT-3 model and its configuration.
type GPT struct {
	cfg *config.Config // Use config.Config instead of gpt.Config
}

// New function initializes a new GPT instance with the provided configuration.
func New(cfg *config.Config) *GPT { // Use config.Config instead of gpt.Config
	return &GPT{
		cfg: cfg,
	}
}

// CreatePayload function creates the payload for the GPT-3 API request.
// It returns the payload as a string and the total number of request tokens.
func (g *GPT) CreatePayload(userMessage string) (string, int, error) {
	// Initialize history with system message and user message
	history := []map[string]string{
		{
			"role":    "system",
			"content": g.cfg.SystemMessage,
		},
		{
			"role":    "user",
			"content": userMessage,
		},
	}

	// Count the tokens in the system message and user message
	totalRequestTokens := CountTokens(userMessage, g.cfg.ModelName) + CountTokens(g.cfg.SystemMessage, g.cfg.ModelName)

	// If totalRequestTokens already exceeds MaxTotalTokens - MaxTokens, return an error
	if totalRequestTokens > g.cfg.MaxTotalTokens-g.cfg.MaxTokens {
		return "", 0, fmt.Errorf("system message and user message alone exceed the maximum allowed tokens for the request")
	}

	// If history is enabled, load the old history and append it to the current history
	if g.cfg.History {
		oldHistory, err := loadHistory()
		if err != nil {
			return "", 0, err
		}
		for i := len(oldHistory) - 1; i >= 0; i-- {
			historyTokens := CountTokens(oldHistory[i]["content"], g.cfg.ModelName)
			if totalRequestTokens+historyTokens <= g.cfg.MaxTotalTokens-g.cfg.MaxTokens {
				totalRequestTokens += historyTokens
				history = append([]map[string]string{oldHistory[i]}, history...)
			} else {
				break
			}
		}
	}

	// Convert the history to JSON
	historyJSON, err := json.Marshal(history)
	if err != nil {
		return "", 0, err
	}

	// Create the payload
	payload := fmt.Sprintf(`{
		"model": "%s",
		"messages": %s,
		"temperature": %f,
		"max_tokens": %d,
		"top_p": %f,
		"frequency_penalty": %f,
		"presence_penalty": %f,
		"stream": %t
	}`, g.cfg.ModelName, historyJSON, g.cfg.Temperature, g.cfg.MaxTokens, g.cfg.TopP, g.cfg.FrequencyPenalty, g.cfg.PresencePenalty, g.cfg.Stream)

	return payload, totalRequestTokens, nil
}

func (g *GPT) HandleResponse(resp *http.Response, startTime time.Time, totalRequestTokens int) (string, int, error) {
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	assistantMsg := ""

	// Add a variable to keep track of the total response tokens
	totalResponseTokens := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error reading response line: %v", err)
			return "", 0, err
		}
		if strings.HasPrefix(line, "data: ") {
			jsonData := line[6:]
			if strings.TrimSpace(jsonData) == "[DONE]" {
				continue
			}
			var event config.Event
			err = json.Unmarshal([]byte(jsonData), &event)
			if err != nil {
				log.Printf("Error unmarshalling event: %v", err)
				return "", 0, err
			}
			// Update the total response tokens
			totalResponseTokens += CountTokens(event.Choices[0].Delta.Content, g.cfg.ModelName)

			fmt.Print(event.Choices[0].Delta.Content)
			assistantMsg += event.Choices[0].Delta.Content
			if event.Choices[0].FinishReason == "stop" {
				fmt.Println()
				if g.cfg.PrintStats {
					fmt.Printf("📋 %d | ⌨️ %d | 📥 %d | ⏰ %.2fs\n", totalResponseTokens+totalRequestTokens, totalRequestTokens, totalResponseTokens, time.Since(startTime).Seconds())
				}
			}
		}
	}
	return assistantMsg, totalResponseTokens, nil
}

// GenerateCompletion function generates a completion using the GPT-3 API.
// It returns the generated completion as a string.
func (g *GPT) GenerateCompletion(userMessage string) (string, error) {
	// Get the start time
	startTime := time.Now()

	// Create the payload and count the total tokens in the request
	payload, totalRequestTokens, err := g.CreatePayload(userMessage)
	if err != nil {
		return "", err
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_SECRET_KEY"))

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	// Handle the response
	response, _, err := g.HandleResponse(resp, startTime, totalRequestTokens)

	if err != nil {
		return "", fmt.Errorf("Failed to handle response: %v", err)
	}

	return response, nil
}

func (g *GPT) AppendHistory(message map[string]string) error {
	historyFile, err := os.OpenFile("history.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer historyFile.Close()

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = historyFile.Write(messageJSON)
	if err != nil {
		return err
	}
	_, err = historyFile.WriteString("\n")
	return err
}

func CountTokens(text string, modelName string) int {
	tkm, err := tiktoken.EncodingForModel(modelName)
	if err != nil {
		log.Printf("EncodingForModel: %v", err)
		return 0
	}
	return len(tkm.Encode(text, nil, nil))
}

func loadHistory() ([]map[string]string, error) {
	historyFile, err := os.Open("history.json")
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]string{}, nil
		}
		return nil, err
	}
	defer historyFile.Close()

	history := make([]map[string]string, 0)
	scanner := bufio.NewScanner(historyFile)
	for scanner.Scan() {
		entry := make(map[string]string)
		err := json.Unmarshal([]byte(scanner.Text()), &entry)
		if err != nil {
			return nil, err
		}
		history = append(history, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return history, nil
}
