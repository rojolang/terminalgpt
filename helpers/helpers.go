package helpers

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/pkoukk/tiktoken-go"
	"github.com/rojolang/terminalgpt/config"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type HistoryEntry struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	TokenCount int    `json:"tokenCount"`
}

func AppendHistory(entry HistoryEntry, historyFile string) error {
	entry.TokenCount, _ = CountTokens(entry.Content, "gpt-4")

	history, err := LoadHistory(historyFile)
	if err != nil {
		return err
	}

	history = append(history, entry)

	file, err := os.OpenFile(historyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	historyJSON, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("Failed to marshal history: %v", err)
	}

	_, err = file.Write(historyJSON)
	if err != nil {
		return err
	}

	return nil
}

func ClearHistory(historyFile string) error {
	err := os.Remove(historyFile)
	if err != nil {
		return fmt.Errorf("Failed to clear history: %v", err)
	}
	return nil
}

func GetHistoryLength(history []map[string]string, modelName string) (int, int, error) {
	tokenSize := 0
	entries := len(history)

	if entries == 0 {
		return tokenSize, entries, nil
	}

	for _, message := range history {
		tokens, err := CountTokens(message["content"], modelName)
		if err != nil {
			return 0, 0, err
		}
		tokenSize += tokens
	}

	return tokenSize, entries, nil
}

func LoadHistory(historyFile string) ([]HistoryEntry, error) {
	file, err := os.Open(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	history := []HistoryEntry{}
	err = json.NewDecoder(file).Decode(&history)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode history: %v", err)
	}

	return history, nil
}

func CountTokens(text string, modelName string) (int, error) {
	tkm, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		return 0, fmt.Errorf("EncodingForModel: %v", err)
	}
	return len(tkm.Encode(text, nil, nil)), nil
}

// New functions...
func HandleFlags() (*bool, *bool, *string, *string, *bool) {
	configFlag := flag.Bool("config", false, "Configure settings")
	clearFlag := flag.Bool("clear", false, "Clear history")
	runMode := flag.String("mode", "", "What mode to run in. (Default or empty: your config.json SystemMessage)")
	workingDirectory := flag.String("dir", "", "What directory to run in. (Default or empty: current directory)")
	promptFlag := flag.Bool("prompt", false, "Run once and exit")

	flag.Parse()

	return configFlag, clearFlag, runMode, workingDirectory, promptFlag
}

func LoadConfig(configFlag *bool) *config.Config {
	_, err := os.Stat(config.ConfigFile)
	if os.IsNotExist(err) || *configFlag {
		err := config.InteractiveConfigure()
		if err != nil {
			color.Red("Failed to configure settings: %v\n", err)
			os.Exit(1)
		}
	}

	cfg, err := config.LoadConfig(config.ConfigFile)
	if err != nil {
		color.Red("Failed to load config file, using default settings: %v\n", err)
		cfg = config.GetDefaultConfig()
		err = config.SaveConfig(cfg)
		if err != nil {
			color.Red("Failed to save default config file: %v\n", err)
			os.Exit(1)
		}
	}

	return &cfg
}

func HandleRunMode(runMode *string, workingDirectory *string, cfg *config.Config) {
	// if runMode is set, use that instead of the config.SystemMessage
	if *runMode != "" {
		cfg.SystemMessage = config.GetRunModeSystemMessage(*runMode, *workingDirectory)
	}
}

func HandleClearFlag(clearFlag *bool) {
	if *clearFlag {
		err := ClearHistory(config.HistoryFile) // Use helper function
		if err != nil {
			color.Red("Failed to clear history: %v\n", err)
			os.Exit(1)
		}
	}
}

func GetHistory(historyFile string) ([]HistoryEntry, error) {
	history, err := LoadHistory(historyFile)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func HandleLaravelMode(userMessage string, workingDirectory string) string {
	userMessageArray := strings.Split(userMessage, " ")
	fileContentMap := make(map[string]string)

	for _, potentialFileName := range userMessageArray {
		if strings.HasSuffix(potentialFileName, ".php") {
			codeFilePaths, err := config.FindFiles(potentialFileName, workingDirectory)
			if err != nil {
				fmt.Println(err)
				continue
			}

			if len(codeFilePaths) == 0 {
				fmt.Println("No files named", potentialFileName, "found.")
				continue
			}

			var codeFilePath string
			if len(codeFilePaths) == 1 {
				fmt.Println("No duplicates found, using the only", potentialFileName, "file.")
				codeFilePath = codeFilePaths[0]
			} else {
				codeFilePath, err = selectFile(codeFilePaths)
				if err != nil || codeFilePath == "" {
					fmt.Println("Failed to select a file:", err)
					continue
				}
			}

			fileContent, err := ioutil.ReadFile(codeFilePath)
			if err != nil {
				fmt.Println("Failed to read file content:", err)
				continue
			}

			fileContentMap[potentialFileName] = string(fileContent)
		}
	}

	for filePath, fileContent := range fileContentMap {
		userMessage += "\n\nMy " + filePath + " file is:\n==\n" + fileContent + "\n==\n"
	}

	return userMessage
}

func HandleGoMode(userMessage string, workingDirectory string) string {
	userMessageArray := strings.Split(userMessage, " ")
	fileContentMap := make(map[string]string)

	for _, potentialFileName := range userMessageArray {
		if strings.HasSuffix(potentialFileName, ".go") {
			codeFilePaths, err := config.FindFiles(potentialFileName, workingDirectory)
			if err != nil {
				fmt.Println(err)
				continue
			}

			if len(codeFilePaths) == 0 {
				fmt.Println("No files named", potentialFileName, "found.")
				continue
			}

			var codeFilePath string
			if len(codeFilePaths) == 1 {
				fmt.Println("No duplicates found, using the only", potentialFileName, "file.")
				codeFilePath = codeFilePaths[0]
			} else {
				codeFilePath, err = selectFile(codeFilePaths)
				if err != nil || codeFilePath == "" {
					fmt.Println("Failed to select a file:", err)
					continue
				}
			}

			fileContent, err := ioutil.ReadFile(codeFilePath)
			if err != nil {
				fmt.Println("Failed to read file content:", err)
				continue
			}

			fileContentMap[potentialFileName] = string(fileContent)
		}
	}

	for filePath, fileContent := range fileContentMap {
		userMessage += "\n\nMy " + filePath + " file is:\n==\n" + fileContent + "\n==\n"
	}

	return userMessage
}

// selectFile prompts the user to select a file from multiple files with the same name.
// It shows the file size, last modified time, and the directory of each file for the user to make a decision.
func selectFile(paths []string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	for i, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		fmt.Printf("%d: %s (size: %d bytes, last modified: %s)\n", i+1, path, info.Size(), info.ModTime().Format(time.RFC3339))
	}

	fmt.Println("Enter the number of the file you want to use, or 'e' to exit:")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	if input == "e" {
		return "", nil
	}

	i, err := strconv.Atoi(input)
	if err != nil || i < 1 || i > len(paths) {
		fmt.Println("Invalid input, please try again.")
		return selectFile(paths)
	}

	return paths[i-1], nil
}
