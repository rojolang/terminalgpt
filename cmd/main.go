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
	runMode := flag.String("mode", "", "What mode to run in. (Default or empty: your config.json SystemMessage)")
	workingDirectory := flag.String("dir", "", "What directory to run in. (Default or empty: current directory)")

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
		err = config.SaveConfig(cfg)
		if err != nil {
			color.Red("Failed to save default config file: %v\n", err)
			return
		}
	}

	// if runMode is set, use that instead of the config.SystemMessage
	if *runMode != "" {
		// make sure run mode is either "laravel" or "go" or exit with error
		if *runMode != "laravel" && *runMode != "go" {
			color.Red("Invalid run mode: %s\n", *runMode)
			return
		}
		cfg.SystemMessage = config.GetRunModeSystemMessage(*runMode, *workingDirectory)
	}

	g := gpt.New(&cfg)

	_, entries, err := g.GetHistoryLength()
	if err != nil {
		color.Red("Failed to get history length: %v\n", err)
		return
	}

	color.Cyan("Model: %s | Max Response Tokens: %d | Max Total Tokens: %d | Temperature: %.2f | History Length: %d | System Message: %s\n\n",
		cfg.ModelName, cfg.MaxResponseTokens, cfg.MaxTotalTokens, cfg.Temperature, entries, cfg.SystemMessage)

	// print in orange last user message
	if cfg.LastUserMessage != "" {
		color.Cyan("Last User Message: ")
		color.Yellow("%s\n\n", cfg.LastUserMessage)
	}

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
		pink.Printf("--config, --clear, --exit, or...  type a prompt (note: *.php, *.go will auto inject file content): ")
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)

		// Use ANSI escape code to move cursor up and clear line
		fmt.Print("\033[1A\033[2K")

		// if userMessage is empty, set userMessage to the last user message
		if userMessage == "" {
			userMessage = cfg.LastUserMessage
		}

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

		// store last user message in config so if they rerun it will be pre-populated
		cfg.LastUserMessage = userMessage
		config.SaveConfig(cfg)

		// Modify user message if running in runMode "laravel"... parse out anything that is *.php and inject file content
		if *runMode == "laravel" || *runMode == "go" {
			// Split userMessage into array of strings
			userMessageArray := strings.Split(userMessage, " ")

			// build a dictionary/mapping of filename => filecontent
			fileContentMap := make(map[string]string)

			// loop through userMessageArray and find any *.php files
			for _, potentialCodeFileName := range userMessageArray {
				if strings.HasSuffix(potentialCodeFileName, ".php") || strings.HasSuffix(potentialCodeFileName, ".go") {

					codeFilePath, err := config.FindFile(potentialCodeFileName, *workingDirectory)
					if err != nil {
						panic(err)
					}

					// read file content
					fileContent, err := os.ReadFile(codeFilePath)
					if err != nil {
						color.Red("Failed to read file content: %v\n", err)
						continue
					}

					// add file content to fileContentMap
					fileContentMap[potentialCodeFileName] = string(fileContent)
				}
			}

			// loop through fileContentMap and append file content to userMessage
			for filePath, fileContent := range fileContentMap {
				// append file content with a prefix of "my current {filename} is: "
				userMessage = userMessage + "\n\nMy  " + filePath + " file is:\n==\n" + fileContent + "\n==\n"
			}

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
