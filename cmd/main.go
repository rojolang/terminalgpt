package main

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/rojolang/terminalgpt/common"
	"github.com/rojolang/terminalgpt/config"
	"github.com/rojolang/terminalgpt/helpers"
	"log"
	"os"
	"strings"
)

func main() {
	configFlag, clearFlag, runMode, workingDirectory, prompt := helpers.HandleFlags()

	// if working directory is empty then set it to the current directory
	if *workingDirectory == "" {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		*workingDirectory = wd
	}

	cfg := helpers.LoadConfig(configFlag)

	helpers.HandleRunMode(runMode, workingDirectory, cfg)

	helpers.HandleClearFlag(clearFlag)

	reader := bufio.NewReader(os.Stdin)

	// Define a function to handle the prompt and response
	handlePrompt := func(userMessage string) {
		pink := color.New(color.FgHiMagenta)
		orange := color.New(color.FgHiYellow)
		orange.Printf("Working Directory: %s\n", *workingDirectory)

		// if run mode is not empty, print it out
		if *runMode != "" {
			orange.Printf("Run Mode: %s\n", *runMode)
		}

		// If userMessage is empty, ask for input
		if userMessage == "" {
			pink.Printf("--config, --clear, --exit, or...  type a prompt (note: *.php will auto inject file content): ")
			userMessage, _ = reader.ReadString('\n')
			userMessage = strings.TrimSpace(userMessage)
		}

		fmt.Print("\033[1A\033[2K")

		if userMessage == "" {
			userMessage = cfg.LastUserMessage
		}

		if userMessage == "--exit" || userMessage == "--quit" {
			os.Exit(0)
		}

		if userMessage == "--config" {
			err := config.InteractiveConfigure()
			if err != nil {
				return
			}
			tempCfg, err := config.LoadConfig(config.ConfigFile)
			if err != nil {
				return
			}
			cfg = &tempCfg
			return
		}

		if userMessage == "--clear" {
			err := helpers.ClearHistory(config.HistoryFile)
			if err != nil {
				return
			}
			return
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
			return
		}

		totalTokens := responseTokens + userMessageTokens + systemMessageTokens + historyTokens

		fmt.Printf("\n📥 %d | 📋 %d | ⌨️ %d | 📜 %d\n", responseTokens, totalTokens, userMessageTokens, historyTokens)

		err = helpers.AppendHistory(helpers.HistoryEntry{
			Role:    "user",
			Content: userMessage,
		}, config.HistoryFile)
		if err != nil {
			return
		}

		err = helpers.AppendHistory(helpers.HistoryEntry{
			Role:    "assistant",
			Content: response,
		}, config.HistoryFile)
		if err != nil {
			return
		}

		history, err := helpers.GetHistory(config.HistoryFile)
		if err != nil {
			return
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

	if len(prompt) > 0 {
		userMessage := strings.Join(prompt, " ")
		handlePrompt(userMessage)
		os.Exit(0)
	}

	// If prompt flag is not set, keep looping
	for {
		handlePrompt("")
	}
}
