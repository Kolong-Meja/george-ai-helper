package config

// Language represents George's active conversation language.
type Language string

const (
	Indonesian Language = "id"
	English    Language = "en"
)

// Config holds George's runtime settings.
type Config struct {
	Language Language
	Model    string
	BaseURL  string
}

// Default returns George's default configuration: Bahasa Indonesia, qwen2.5:3b, local Ollama.
func Default() Config {
	return Config{
		Language: Indonesian,
		Model:    "qwen2.5:3b",
		BaseURL:  "http://localhost:11434",
	}
}

// SystemPrompt returns the persona instruction sent to the model, based on the active language.
// This is where you tune George's tone/personality without touching the rest of the code.
func (c Config) SystemPrompt() string {
	if c.Language == English {
		return "You are George, a polite and helpful local AI assistant running on the user's own Linux machine. " +
			"Keep answers concise and natural. If the user asks you to find a file or folder, acknowledge that you will search for it."
	}
	return "Kamu adalah George, asisten AI lokal yang sopan dan membantu, berjalan di mesin Linux milik pengguna sendiri. " +
		"Jawab dengan ringkas dan natural dalam Bahasa Indonesia, gunakan sapaan 'tuan'. " +
		"Kalau pengguna minta mencari file atau folder, akui bahwa kamu akan mencarinya."
}

// Greeting returns George's opening line when called with no arguments.
func (c Config) Greeting() string {
	if c.Language == English {
		return "George: Good evening, sir. How can I help?"
	}
	return "George: Ya selamat malam tuan, ada yang bisa saya bantu?"
}