package gpt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/pkoukk/tiktoken-go"
	"github.com/rojolang/terminalgpt/config"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type GPT struct {
	cfg     *config.Config
	history []map[string]string
}

func New(cfg *config.Config) *GPT {
	history, _ := loadHistory()
	return &GPT{
		cfg:     cfg,
		history: history,
	}
}

func (g *GPT) CreatePayload(userMessage string) (string, int, error) {
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

	userMessageTokens, err := CountTokens(userMessage, g.cfg.ModelName)
	if err != nil {
		return "", 0, err
	}

	systemMessageTokens, err := CountTokens(g.cfg.SystemMessage, g.cfg.ModelName)
	if err != nil {
		return "", 0, err
	}

	totalRequestTokens := userMessageTokens + systemMessageTokens

	if totalRequestTokens > g.cfg.MaxTotalTokens-g.cfg.MaxTokens {
		return "", 0, fmt.Errorf("system message and user message alone exceed the maximum allowed tokens for the request")
	}

	if g.cfg.History {
		for i := len(g.history) - 1; i >= 0; i-- {
			historyTokens, err := CountTokens(g.history[i]["content"], g.cfg.ModelName)
			if err != nil {
				return "", 0, err
			}

			if totalRequestTokens+historyTokens <= g.cfg.MaxTotalTokens-g.cfg.MaxTokens {
				totalRequestTokens += historyTokens
				history = append([]map[string]string{g.history[i]}, history...)
			} else {
				break
			}
		}
	}

	historyJSON, err := json.Marshal(history)
	if err != nil {
		return "", 0, err
	}

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
	totalResponseTokens := 0
	isFirstChunk := true
	boldBlue := color.New(color.FgBlue, color.Bold).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}

	promptLabel := "Prompt:"
	responseLabel := "Response:"
	maxLabelLength := max(len(promptLabel), len(responseLabel))

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
				return "", 0, fmt.Errorf("Failed to unmarshal event: %v", err)
			}

			responseTokens, err := CountTokens(event.Choices[0].Delta.Content, g.cfg.ModelName)
			if err != nil {
				return "", 0, err
			}

			totalResponseTokens += responseTokens

			if isFirstChunk {
				fmt.Printf("%-*s ", maxLabelLength, boldBlue(responseLabel))
				isFirstChunk = false
			}

			// Apply tabbing to each chunk
			tabbedChunk := strings.ReplaceAll(event.Choices[0].Delta.Content, "\n", "\n\t")

			fmt.Print(blue(tabbedChunk))
			assistantMsg += tabbedChunk

			if event.Choices[0].FinishReason == "stop" {
				fmt.Println()
				if g.cfg.PrintStats {
					fmt.Printf("%-*s %s\n", maxLabelLength, "", fmt.Sprintf("\n\t📋 %d | ⌨️ %d | 📥 %d | ⏰ %.2fs\n", totalResponseTokens+totalRequestTokens, totalRequestTokens, totalResponseTokens, time.Since(startTime).Seconds()))
				}
			}
		}
	}

	return assistantMsg, totalResponseTokens, nil
}

func (g *GPT) GenerateCompletion(userMessage string) (string, error) {
	startTime := time.Now()

	payload, totalRequestTokens, err := g.CreatePayload(userMessage)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_SECRET_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to send HTTP request: %v", err)
	}

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
		return fmt.Errorf("Failed to marshal message: %v", err)
	}
	_, err = historyFile.Write(messageJSON)
	if err != nil {
		return err
	}
	_, err = historyFile.WriteString("\n")
	return err
}

func (g *GPT) ClearHistory() error {
	err := os.Remove("history.json")
	if err != nil {
		return fmt.Errorf("Failed to clear history: %v", err)
	}
	return nil
}

// GetHistoryLength returns the total token size of all the history and the number of entries
func (g *GPT) GetHistoryLength() (int, int, error) {
	tokenSize := 0
	entries := len(g.history)

	if entries == 0 {
		return tokenSize, entries, nil
	}

	for _, message := range g.history {
		tokens, err := CountTokens(message["content"], g.cfg.ModelName)
		if err != nil {
			return 0, 0, err
		}
		tokenSize += tokens
	}

	return tokenSize, entries, nil
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
			return nil, fmt.Errorf("Failed to unmarshal entry: %v", err)
		}
		history = append(history, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return history, nil
}

func CountTokens(text string, modelName string) (int, error) {
	tkm, err := tiktoken.EncodingForModel(modelName)
	if err != nil {
		return 0, fmt.Errorf("EncodingForModel: %v", err)
	}
	return len(tkm.Encode(text, nil, nil)), nil
}
