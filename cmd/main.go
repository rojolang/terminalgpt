package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/rojolang/terminalgpt/config"
	"github.com/rojolang/terminalgpt/gpt"
	"os"
	"strings"
)

func main() {
	configFlag := flag.Bool("config", false, "Configure settings")
	clearFlag := flag.Bool("clear", false, "Clear history")
	flag.Parse()

	_, err := os.Stat(config.ConfigFile)
	if os.IsNotExist(err) || *configFlag {
		err := config.InteractiveConfigure()
		if err != nil {
			color.Red("Failed to configure settings: %v\n", err)
			return
		}
	}

	cfg, err := config.LoadConfig(config.ConfigFile)
	if err != nil {
		color.Red("Failed to load config file, using default settings: %v\n", err)
		cfg = config.GetDefaultConfig()
		err = config.SaveConfig(config.ConfigFile, cfg)
		if err != nil {
			color.Red("Failed to save default config file: %v\n", err)
			return
		}
	}

	g := gpt.New(&cfg)

	_, entries, err := g.GetHistoryLength()
	if err != nil {
		color.Red("Failed to get history length: %v\n", err)
		return
	}

	color.Cyan("Model: %s | Max Tokens: %d | Max Total Tokens: %d | Temperature: %.2f | History Length: %d | System Message: %s\n",
		cfg.ModelName, cfg.MaxTokens, cfg.MaxTotalTokens, cfg.Temperature, entries, cfg.SystemMessage)

	if *clearFlag {
		err := g.ClearHistory()
		if err != nil {
			color.Red("Failed to clear history: %v\n", err)
			return
		}
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		// Create a new color object for pink
		pink := color.New(color.FgHiMagenta)

		// Use the color object to print the text
		pink.Printf("--config, --clear, --exit, or...  type a prompt: ")
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)

		// Use ANSI escape code to move cursor up and clear line
		fmt.Print("\033[1A\033[2K")

		if userMessage == "--exit" || userMessage == "--quit" {
			break
		}

		if userMessage == "--config" {
			err := config.InteractiveConfigure()
			if err != nil {
				color.Red("Failed to configure settings: %v\n", err)
				continue
			}
			cfg, err = config.LoadConfig(config.ConfigFile)
			if err != nil {
				color.Red("Failed to load config file: %v\n", err)
				continue
			}
			g = gpt.New(&cfg)
			continue
		}

		if userMessage == "--clear" {
			err := g.ClearHistory()
			if err != nil {
				color.Red("Failed to clear history: %v\n", err)
				continue
			}
			color.Blue("History cleared.")
			continue
		}

		// Define max function inline
		max := func(a, b int) int {
			if a > b {
				return a
			}
			return b
		}

		// Print the prompt immediately after the user presses enter
		promptLabel := "Prompt:"
		responseLabel := "Response:"
		maxLabelLength := max(len(promptLabel), len(responseLabel))
		fmt.Printf("%-*s %s\n", maxLabelLength, color.GreenString(promptLabel), userMessage)

		_, err = g.GenerateCompletion(userMessage)
		if err != nil {
			color.Red("Failed to generate completion: %v\n", err)
			continue
		}

		err = g.AppendHistory(map[string]string{
			"role":    "user",
			"content": userMessage,
		})
		if err != nil {
			color.Red("Failed to append user message to history: %v\n", err)
			continue
		}

		// Add print statement to check if history is updated
		_, entries, err := g.GetHistoryLength()
		if err != nil {
			color.Red("Failed to get history length: %v\n", err)
			return
		}

		fmt.Printf("%-*s %s\n", maxLabelLength, "", color.CyanString("History Length: %d\n\n", entries))
	}

}
