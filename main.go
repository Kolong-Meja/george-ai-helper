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

// maxHistoryTurns caps how many user+assistant exchanges George keeps in memory
// during one chat session. Keeps the request sent to Ollama small and fast on an
// 8GB CPU-only machine instead of growing unbounded the longer you chat.
const maxHistoryTurns = 12

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

	client := ollama.New(cfg.BaseURL, cfg.Model, cfg.Temperature, cfg.ContextSize)

	// No remaining args: `george` alone -> greet, then drop into interactive chat.
	if len(args) == 0 {
		greeting := cfg.Greeting()
		fmt.Println(greeting)

		// Seed history with the persona + the greeting George just gave, so if
		// you reply to it ("makasih buat ucapannya"), George actually remembers
		// saying it instead of drawing a blank on the next turn.
		history := []ollama.Message{
			{Role: "system", Content: cfg.SystemPrompt()},
			{Role: "assistant", Content: strings.TrimPrefix(greeting, "George: ")},
		}
		chatLoop(cfg, client, history)
		return
	}

	// Remaining args present: `george <perintah>` or `hello george <perintah>` -> one-shot.
	input := strings.Join(args, " ")
	history := []ollama.Message{{Role: "system", Content: cfg.SystemPrompt()}}
	handleInput(cfg, client, history, input)
}

func chatLoop(cfg config.Config, client *ollama.Client, history []ollama.Message) {
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
		history = handleInput(cfg, client, history, input)
	}
}

// handleInput processes one turn and returns the updated history so the caller's
// next call keeps the full conversation. Deterministic commands (file search, etc.)
// still skip the LLM entirely, but their exchange is now recorded too, so a later
// LLM call still has the full picture of what's already been said.
func handleInput(cfg config.Config, client *ollama.Client, history []ollama.Message, input string) []ollama.Message {
	if res := router.TryHandle(input, cfg); res.Handled {
		fmt.Println("George:", res.Output)
		return trimHistory(append(history,
			ollama.Message{Role: "user", Content: input},
			ollama.Message{Role: "assistant", Content: res.Output},
		))
	}

	history = append(history, ollama.Message{Role: "user", Content: input})

	reply, err := client.Chat(trimHistory(history))
	if err != nil {
		fmt.Println("George: Waduh bro, ada masalah nyambungin ke model AI:", err)
		// Drop the user turn we just added - it never got a real reply, so keeping
		// it would duplicate the input if you retry.
		return history[:len(history)-1]
	}

	fmt.Println("George:", reply)
	return trimHistory(append(history, ollama.Message{Role: "assistant", Content: reply}))
}

// trimHistory keeps the system message plus only the most recent maxHistoryTurns
// exchanges, so the request sent to Ollama stays small and fast no matter how long
// the chat session runs.
func trimHistory(history []ollama.Message) []ollama.Message {
	maxLen := 1 + maxHistoryTurns*2 // system message + (user, assistant) per turn
	if len(history) <= maxLen {
		return history
	}
	trimmed := make([]ollama.Message, 0, maxLen)
	trimmed = append(trimmed, history[0])
	trimmed = append(trimmed, history[len(history)-(maxLen-1):]...)
	return trimmed
}
