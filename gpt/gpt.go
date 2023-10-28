// Package gpt provides a set of functions and types to handle the interaction with the GPT model.
// It includes functions to create a payload, create a request, handle the response, and generate a completion.
//
// Functions:
// - CreatePayload: Creates a payload for the GPT model.
// - CreateRequest: Creates a request for the GPT model.
// - HandleResponse: Handles the response from the GPT model.
// - GenerateCompletion: Generates a completion using the GPT model.
package gpt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tiktoken "github.com/pkoukk/tiktoken-go"
	"github.com/rojolang/terminalgpt/config"
)

// CreatePayload function creates a payload for the GPT model.
// It takes a Config struct and a user message as input and returns a payload string.
func CreatePayload(config config.Config, userMessage string) string {
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
	}`, config.ModelName, config.SystemMessage, userMessage, config.Temperature, config.MaxTokens, config.TopP, config.FrequencyPenalty, config.PresencePenalty, config.Stream)
}

// CreateRequest function creates a request for the GPT model.
// It takes a Config struct and a user message as input and returns an http.Request and an error.
func CreateRequest(config config.Config, userMessage string) (*http.Request, error) {
	payload := strings.NewReader(CreatePayload(config, userMessage))
	req, err := http.NewRequest("POST", config.CompletionAPIURL, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.AuthorizationKey))
	return req, nil
}

// HandleResponse function handles the response from the GPT model.
// It takes an http.Response, a Tiktoken, the number of user tokens, and a boolean indicating whether to print stats as input.
// It returns an error.
func HandleResponse(resp *http.Response, tkm *tiktoken.Tiktoken, userTokens int, printStats bool) error {
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
			var event config.Event
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
						fmt.Printf("\n[🪙 %d ( 👤 %d | 💻 %d) ⏰ %ds]\n", userTokens+assistantTokens, userTokens, assistantTokens, int(time.Since(config.StartTime).Seconds()))
					}
				}
			}
		}
	}
	return nil
}

// GenerateCompletion function generates a completion using the GPT model.
// It takes a user message as input and returns a completion and an error.
func GenerateCompletion(userMessage string) (string, error) {
	config, err := config.ReadConfig(config.ConfigFile)
	if err != nil {
		return "", err
	}

	req, err := CreateRequest(config, userMessage)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	tkm, err := config.GetTokenCount(userMessage, config.ModelName)
	if err != nil {
		return "", err
	}
	userTokens := len(tkm.Encode(userMessage, nil, nil))

	if err := HandleResponse(resp, tkm, userTokens, config.PrintStats); err != nil {
		return "", err
	}

	return "", nil
}
