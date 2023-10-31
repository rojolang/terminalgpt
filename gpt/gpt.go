package gpt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/rojolang/terminalgpt/config"
	"github.com/rojolang/terminalgpt/helpers"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type GPT struct {
	cfg     *config.Config
	history []helpers.HistoryEntry
}

func (g *GPT) GetHistory() []helpers.HistoryEntry {
	return g.history
}

func New(cfg *config.Config) (*GPT, error) {
	history, err := helpers.LoadHistory(config.HistoryFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load history: %w", err)
	}
	return &GPT{
		cfg:     cfg,
		history: history,
	}, nil
}

func (g *GPT) CreatePayload(userMessage string) (string, int, int, error) {
	history := []helpers.HistoryEntry{
		{
			Role:    "system",
			Content: g.cfg.SystemMessage,
		},
		{
			Role:    "user",
			Content: userMessage,
		},
	}

	userMessageTokens, err := helpers.CountTokens(userMessage, g.cfg.ModelName)
	if err != nil {
		return "", 0, 0, err
	}

	systemMessageTokens, err := helpers.CountTokens(g.cfg.SystemMessage, g.cfg.ModelName)
	if err != nil {
		return "", 0, 0, err
	}

	totalRequestTokens := userMessageTokens + systemMessageTokens

	if totalRequestTokens > (g.cfg.MaxTotalTokens - g.cfg.MaxResponseTokens) {
		return "", 0, 0, fmt.Errorf("Request token count (%d) exceeds the maximum total token count (%d - %d = %d)", totalRequestTokens, g.cfg.MaxTotalTokens, g.cfg.MaxResponseTokens, (g.cfg.MaxTotalTokens - g.cfg.MaxResponseTokens))
	}

	if g.cfg.History {
		for i := len(g.history) - 1; i >= 0; i-- {
			historyTokens, err := helpers.CountTokens(g.history[i].Content, g.cfg.ModelName)
			if err != nil {
				return "", 0, 0, err
			}

			if totalRequestTokens+historyTokens <= g.cfg.MaxTotalTokens-g.cfg.MaxResponseTokens {
				totalRequestTokens += historyTokens
				history = append([]helpers.HistoryEntry{g.history[i]}, history...)
			} else {
				break
			}
		}
	}

	historyJSON, err := json.Marshal(history)
	if err != nil {
		return "", 0, 0, err
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
	}`, g.cfg.ModelName, historyJSON, g.cfg.Temperature, g.cfg.MaxResponseTokens, g.cfg.TopP, g.cfg.FrequencyPenalty, g.cfg.PresencePenalty, g.cfg.Stream)

	return payload, userMessageTokens, systemMessageTokens, nil
}

func (g *GPT) HandleResponse(resp *http.Response, startTime time.Time, totalRequestTokens int, userMessageTokens int, systemMessageTokens int) (string, int, int, int, int, error) {
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
			return "", 0, 0, 0, 0, err
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
				return "", 0, 0, 0, 0, fmt.Errorf("Failed to unmarshal event: %v", err)
			}

			responseTokens, err := helpers.CountTokens(event.Choices[0].Delta.Content, g.cfg.ModelName)
			if err != nil {
				return "", 0, 0, 0, 0, err
			}

			totalResponseTokens += responseTokens

			if isFirstChunk {
				fmt.Printf("\n%-*s ", maxLabelLength, boldBlue(responseLabel))
				isFirstChunk = false
			}

			// Apply tabbing to each chunk
			tabbedChunk := strings.ReplaceAll(event.Choices[0].Delta.Content, "\n", "\n\t")

			fmt.Print(blue(tabbedChunk))
			assistantMsg += tabbedChunk
		}
	}

	return assistantMsg, totalResponseTokens, userMessageTokens, systemMessageTokens, totalRequestTokens + totalResponseTokens, nil
}

func (g *GPT) GenerateCompletion(userMessage string) (string, int, int, int, int, error) {
	startTime := time.Now()

	payload, userMessageTokens, systemMessageTokens, err := g.CreatePayload(userMessage)
	if err != nil {
		return "", 0, 0, 0, 0, err
	}

	totalRequestTokens := userMessageTokens + systemMessageTokens

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return "", 0, 0, 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_SECRET_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, 0, 0, 0, fmt.Errorf("Failed to send HTTP request: %v", err)
	}

	response, responseTokens, userMessageTokens, systemMessageTokens, totalTokens, err := g.HandleResponse(resp, startTime, totalRequestTokens, userMessageTokens, systemMessageTokens)
	if err != nil {
		return "", 0, 0, 0, 0, fmt.Errorf("Failed to handle response: %v", err)
	}

	return response, responseTokens, userMessageTokens, systemMessageTokens, totalTokens, nil
}
