package main

import (
	"flag"
	"fmt"
	"github.com/rojolang/terminalgpt/config"
	"github.com/rojolang/terminalgpt/gpt"
	"os"
	"strings"
)

func main() {
	configFlag := flag.Bool("config", false, "Configure settings")
	flag.Parse()

	_, err := os.Stat(config.ConfigFile)
	if os.IsNotExist(err) || *configFlag {
		err := config.InteractiveConfigure()
		if err != nil {
			fmt.Println("Failed to configure settings.")
			return
		}
	}

	cfg, err := config.LoadConfig(config.ConfigFile)
	if err != nil {
		fmt.Println("Failed to load config file, using default settings.")
		cfg = config.GetDefaultConfig()
		err = config.SaveConfig(config.ConfigFile, cfg)
		if err != nil {
			fmt.Printf("Failed to save default config file: %v\n", err)
			return
		}
	}

	g := gpt.New(&cfg) // Pass a pointer to cfg

	userMessage := strings.Join(flag.Args(), " ")

	response, err := g.GenerateCompletion(userMessage) // Remove startTime
	if err != nil {
		fmt.Printf("Failed to generate completion: %v\n", err)
		return
	}

	err = g.AppendHistory(map[string]string{
		"role":    "user",
		"content": userMessage,
	})
	if err != nil {
		fmt.Printf("Failed to append user message to history: %v\n", err)
		return
	}

	err = g.AppendHistory(map[string]string{
		"role":    "assistant",
		"content": response,
	})
	if err != nil {
		fmt.Printf("Failed to append assistant response to history: %v\n", err)
		return
	}
}
