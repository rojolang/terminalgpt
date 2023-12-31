package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/rojolang/terminalgpt/helpers"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

const LanguageModel = "gpt-4"

// Add a function to detect code blocks and color them yellow
func colorCodeBlocks(text string) string {
	languages := []string{"1c", "abnf", "accesslog", "actionscript", "ada", "angelscript", "apache", "applescript", "arcade", "arduino", "armasm", "asciidoc", "aspectj", "autohotkey", "autoit", "avrasm", "awk", "axapta", "bash", "basic", "bnf", "brainfuck", "c", "cal", "capnproto", "ceylon", "clean", "clojure-repl", "clojure", "cmake", "coffeescript", "coq", "cos", "cpp", "crmsh", "crystal", "csharp", "csp", "css", "d", "dart", "delphi", "diff", "django", "dns", "dockerfile", "dos", "dsconfig", "dts", "dust", "ebnf", "elixir", "elm", "erb", "erlang-repl", "erlang", "excel", "fix", "flix", "fortran", "fsharp", "gams", "gauss", "gcode", "gherkin", "glsl", "gml", "go", "golo", "html", "gradle", "graphql", "groovy", "haml", "handlebars", "haskell", "haxe", "hsp", "http", "hy", "inform7", "ini", "irpf90", "isbl", "java", "javascript", "jboss-cli", "json", "julia-repl", "julia", "kotlin", "lasso", "latex", "ldif", "leaf", "less", "lisp", "livecodeserver", "livescript", "llvm", "lsl", "lua", "makefile", "markdown", "mathematica", "matlab", "maxima", "mel", "mercury", "mipsasm", "mizar", "mojolicious", "monkey", "moonscript", "n1ql", "nestedtext", "nginx", "nim", "nix", "node-repl", "nsis", "objectivec", "ocaml", "openscad", "oxygene", "parser3", "perl", "pf", "pgsql", "php-template", "php", "plaintext", "pony", "powershell", "processing", "profile", "prolog", "properties", "protobuf", "puppet", "purebasic", "python-repl", "python", "q", "qml", "r", "reasonml", "rib", "roboconf", "routeros", "rsl", "ruby", "ruleslanguage", "rust", "sas", "scala", "scheme", "scilab", "scss", "shell", "smali", "smalltalk", "sml", "sqf", "sql", "stan", "stata", "step21", "stylus", "subunit", "swift", "taggerscript", "tap", "tcl", "thrift", "tp", "twig", "typescript", "vala", "vbnet", "vbscript-html", "vbscript", "verilog", "vhdl", "vim", "wasm", "wren", "x86asm", "xl", "xml", "xquery", "yaml", "zephir"}
	yellow := "\033[33m"
	reset := "\033[0m"

	for _, lang := range languages {
		prefix := "```" + lang
		if strings.HasPrefix(text, prefix) {
			text = strings.TrimPrefix(text, prefix)
			text = strings.TrimSuffix(text, "```")
			return yellow + text + reset
		}
	}
	return text
}

func GenerateCompletion(userMessage, systemMessage, azureURL, azureAuthKey, modelName string, maxTokens int32, topP, temperature, frequencyPenalty, presencePenalty float32, timeout time.Duration, history []helpers.HistoryEntry) (string, int, int, int, int, error) {
	userMessageTokens, err := helpers.CountTokens(userMessage, LanguageModel)
	if err != nil {
		return "", 0, 0, 0, 0, err
	}

	systemMessageTokens, err := helpers.CountTokens(systemMessage, LanguageModel)
	if err != nil {
		return "", 0, 0, 0, 0, err
	}

	historyTokens := 0
	for _, entry := range history {
		count, err := helpers.CountTokens(entry.Content, LanguageModel)
		if err != nil {
			return "", 0, 0, 0, 0, err
		}
		historyTokens += count
	}
	ctx := context.Background()

	keyCredential, err := azopenai.NewKeyCredential(azureAuthKey)
	if err != nil {
		logrus.WithError(err).Error("Failed to create key credential")
		return "", 0, 0, 0, 0, err
	}

	client, err := azopenai.NewClientWithKeyCredential(azureURL, keyCredential, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to create client with key credential")
		return "", 0, 0, 0, 0, err
	}

	messages := []azopenai.ChatMessage{
		{Role: to.Ptr(azopenai.ChatRoleSystem), Content: to.Ptr(systemMessage)},
		{Role: to.Ptr(azopenai.ChatRoleUser), Content: to.Ptr(userMessage)},
	}

	for _, entry := range history {
		messages = append([]azopenai.ChatMessage{
			{Role: to.Ptr(azopenai.ChatRole(entry.Role)), Content: to.Ptr(entry.Content)},
		}, messages...)
	}

	resp, err := client.GetChatCompletionsStream(ctx, azopenai.ChatCompletionsOptions{
		Messages:         messages,
		N:                to.Ptr[int32](1),
		Deployment:       modelName,
		Temperature:      to.Ptr(temperature),
		TopP:             to.Ptr(topP),
		MaxTokens:        to.Ptr(maxTokens),
		FrequencyPenalty: to.Ptr(frequencyPenalty),
		PresencePenalty:  to.Ptr(presencePenalty),
	}, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to get chat completions stream")
		return "", 0, 0, 0, 0, err
	}
	defer resp.ChatCompletionsStream.Close()

	responseTokens := 0

	for {
		_, cancel := context.WithTimeout(ctx, timeout)
		chatCompletions, err := resp.ChatCompletionsStream.Read()
		cancel()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.WithError(err).Error("Failed to read from chat completions stream")
			return "", 0, 0, 0, 0, err
		}

		for _, choice := range chatCompletions.Choices {
			text := ""
			if choice.Delta.Content != nil {
				text = *choice.Delta.Content
			}
			if text == "" {
				continue
			}

			// Color the code blocks if they match any of the given languages
			coloredText := colorCodeBlocks(text)
			print(coloredText)

			tokens, err := helpers.CountTokens(text, LanguageModel)
			if err != nil {
				return "", 0, 0, 0, 0, err
			}
			responseTokens += tokens
		}
	}

	return "", userMessageTokens, systemMessageTokens, responseTokens, historyTokens, nil
}
