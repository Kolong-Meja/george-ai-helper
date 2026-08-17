package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Kolong-Meja/george/internal/config"
	"github.com/Kolong-Meja/george/internal/ollama"
	"github.com/Kolong-Meja/george/internal/router"
)

func main() {
	cfg := config.Default()
	args := os.Args[1:]

	// Optional language override: `george --lang en ...`
	if len(args) >= 2 && args[0] == "--lang" {
		switch args[1] {
		case "en":
			cfg.Language = config.English
		case "id":
			cfg.Language = config.Indonesian
		default:
			fmt.Fprintf(os.Stderr, "Bahasa tidak dikenal: %s (pakai 'id' atau 'en')\n", args[1])
			os.Exit(1)
		}
		args = args[2:]
	}

	client := ollama.New(cfg.BaseURL, cfg.Model)

	// No remaining args: `george` alone -> greet, then drop into interactive chat.
	if len(args) == 0 {
		fmt.Println(cfg.Greeting())
		chatLoop(cfg, client)
		return
	}

	// Remaining args present: `george <perintah>` or `hello george <perintah>` -> one-shot.
	input := strings.Join(args, " ")
	handleInput(cfg, client, input)
}

func chatLoop(cfg config.Config, client *ollama.Client) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			return
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "keluar" {
			return
		}
		handleInput(cfg, client, input)
	}
}

func handleInput(cfg config.Config, client *ollama.Client, input string) {
	// Deterministic commands (file search, etc.) skip the LLM entirely.
	if res := router.TryHandle(input); res.Handled {
		fmt.Println("George:", res.Output)
		return
	}

	reply, err := client.Generate(input, cfg.SystemPrompt())
	if err != nil {
		fmt.Println("George: Maaf tuan, ada masalah menghubungi model AI:", err)
		return
	}
	fmt.Println("George:", reply)
}