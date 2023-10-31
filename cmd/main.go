package main

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/rojolang/terminalgpt/common"
	"github.com/rojolang/terminalgpt/config"
	"github.com/rojolang/terminalgpt/helpers"
	"os"
	"strings"
)

func main() {
	configFlag, clearFlag, runMode, workingDirectory := helpers.HandleFlags()

	cfg := helpers.LoadConfig(configFlag)

	helpers.HandleRunMode(runMode, workingDirectory, cfg)

	helpers.HandleClearFlag(clearFlag)

	reader := bufio.NewReader(os.Stdin)

	for {
		pink := color.New(color.FgHiMagenta)
		pink.Printf("--config, --clear, --exit, or...  type a prompt (note: *.php will auto inject file content): ")
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)

		fmt.Print("\033[1A\033[2K")

		if userMessage == "" {
			userMessage = cfg.LastUserMessage
		}

		if userMessage == "--exit" || userMessage == "--quit" {
			break
		}

		if userMessage == "--config" {
			err := config.InteractiveConfigure()
			if err != nil {
				continue
			}
			tempCfg, err := config.LoadConfig(config.ConfigFile)
			if err != nil {
				continue
			}
			cfg = &tempCfg
			continue
		}

		if userMessage == "--clear" {
			err := helpers.ClearHistory(config.HistoryFile)
			if err != nil {
				continue
			}
			continue
		}

		cfg.LastUserMessage = userMessage
		config.SaveConfig(*cfg)

		if *runMode == "laravel" {
			userMessage = helpers.HandleLaravelMode(userMessage, *workingDirectory)
		} else if *runMode == "go" {
			userMessage = helpers.HandleGoMode(userMessage, *workingDirectory)
		}

		fmt.Printf("Prompt: %s\n", userMessage)
		fmt.Print("Response: ")

		response, userMessageTokens, systemMessageTokens, responseTokens, historyTokens, err := common.GenerateCompletion(cfg, userMessage)
		if err != nil {
			continue
		}

		totalTokens := responseTokens + userMessageTokens + systemMessageTokens + historyTokens

		fmt.Printf("\n📥 %d | 📋 %d | ⌨️ %d | 📜 %d\n", responseTokens, totalTokens, userMessageTokens, historyTokens)

		err = helpers.AppendHistory(helpers.HistoryEntry{
			Role:    "user",
			Content: userMessage,
		}, config.HistoryFile)
		if err != nil {
			continue
		}

		err = helpers.AppendHistory(helpers.HistoryEntry{
			Role:    "assistant",
			Content: response,
		}, config.HistoryFile)
		if err != nil {
			continue
		}

		history, err := helpers.GetHistory(config.HistoryFile)
		if err != nil {
			continue
		}
		entries := len(history)

		historyTokens = 0
		for _, entry := range history {
			tokenCount, err := helpers.CountTokens(entry.Content, "gpt-4")
			if err != nil {
				fmt.Println("Error counting tokens:", err)
				continue
			}
			historyTokens += tokenCount
		}
		fmt.Printf("History Length: %d, History Tokens: %d\n\n", entries, historyTokens)

	}
}
