# TerminalGPT 🤖

TerminalGPT is a command-line interface (CLI) tool that leverages OpenAI's GPT-4 model to interactively generate text completions. It's perfect for debugging, generating code, figuring out bottlenecks in your code or adding in better error handling
## Todo
- Project Tech Stack (libraries and packages versions, OS version, etc)
- Terminal context (from program errors)
- Source code as context (code context)
- Vector integration 
- Multiple agent chain 
- Add doc retrieval / web navigation tools
  
## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Contributing](#contributing)

## Features
- Interactive mode: Communicate with GPT-4 in real-time right from your terminal.
- History: Keeps track of your interaction history with the AI.
- Customizable: You can modify various parameters to suit your needs.

## Installation

1. **Install Go**

   If Go is not installed on your system, you can download it [here](https://golang.org/dl/). Follow the instructions according to your operating system.

2. **Clone the Repository**

```
git clone https://github.com/rojolang/terminalgpt.git
```

3. **Navigate to the Project Directory**

```
cd terminalgpt
```

4. **Build the Project**

```
go build -o terminalgpt
```

5. **Move the Executable**

   Move the executable to a directory in your PATH. For example, `/usr/local/bin`:

```
sudo mv terminalgpt /usr/local/bin
```

6. **Set up an Alias**

   For convenience, you can set up an alias to run the program. Here's how you can do this:

   - In `bash`, you can echo the alias command into your `.bashrc` or `.bash_profile`:

     ```
     echo "alias gpt='terminalgpt'" >> ~/.bashrc
     source ~/.bashrc
     ```

   - In `zsh`, you can echo the alias command into your `.zshrc`:

     ```
     echo "alias gpt='terminalgpt'" >> ~/.zshrc
     source ~/.zshrc
     ```

   Now you can start the program with the command `gpt`.

## Usage

1. **Set the OpenAI Secret Key**

   You need to set your OpenAI secret key as an environment variable:

```
export OPENAI_SECRET_KEY=your-secret-key
```

   Replace `your-secret-key` with your actual OpenAI secret key.

2. **Run TerminalGPT**

   After installation, you can start the program with:

```
terminalgpt
```

   Or if you've set up the alias:

```
gpt
```

   You can then interact with the GPT-4 model directly from your terminal. To exit, type `--exit` or `--quit`.

## Configuration

TerminalGPT allows you to customize various settings. You can change these settings by running:

```
terminalgpt --config
```

This will launch an interactive configuration process where you can change the model, temperature, max tokens, etc.

## Contributing

Contributions to improve TerminalGPT are welcomed. Feel free to create a PR or raise an issue.

## License

TerminalGPT is open-source software licensed under the MIT license.

Enjoy your AI-powered terminal experience! 🚀

Check out our other project [cntrl.ai](https://cntrl.ai/) for more AI-powered tools.

