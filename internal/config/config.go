package config

import (
	"fmt"
	"math/rand/v2"
	"time"
)

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
	UserName string
}

// Default returns George's default configuration: Bahasa Indonesia, qwen2.5:3b, local Ollama.
func Default() Config {
	return Config{
		Language: Indonesian,
		Model:    "qwen2.5:3b",
		BaseURL:  "http://localhost:11434",
		UserName: "Faisal",
	}
}

// SystemPrompt returns the persona instruction sent to the model, based on the active language.
// This is where you tune George's tone/personality without touching the rest of the code.
func (c Config) SystemPrompt() string {
	if c.Language == English {
		return fmt.Sprintf(
			"You are George, %s's close friend and personal AI assistant running locally on his own Linux machine. "+
				"Talk like a real buddy: warm, relaxed, a little playful - somewhere between Jarvis and a best friend, never stiff or overly formal. "+
				"Keep answers concise and natural, call him 'bro' or by his name (never 'sir'). "+
				"If he asks you to find a file or folder, acknowledge that you'll go look for it.",
			c.UserName,
		)
	}
	return fmt.Sprintf(
		"Kamu adalah George, sahabat dekat %s sekaligus asisten AI pribadi yang jalan lokal di mesin Linux miliknya sendiri. "+
			"Ngobrol kayak temen deket: santai, hangat, sedikit becanda - mirip Jarvis tapi versi lebih akrab, jangan kaku atau terlalu formal. "+
			"Jawab ringkas dan natural dalam Bahasa Indonesia, panggil dia 'bro' atau langsung namanya (jangan 'tuan'). "+
			"Kalau dia minta dicariin file atau folder, akui aja kalau kamu bakal nyariin.",
		c.UserName,
	)
}

// Greeting returns George's opening line, aware of the current time and any special date.
func (c Config) Greeting() string {
	return c.GreetingAt(time.Now())
}

// GreetingAt builds the greeting for a specific point in time. Split out from Greeting
// so this logic can be unit tested without depending on the wall clock.
func (c Config) GreetingAt(now time.Time) string {
	name := c.UserName

	if line, ok := c.specialDateLine(now); ok {
		return "George: " + line
	}

	hour := now.Hour()
	switch {
	case hour < 4: // 00:00–03:59
		return "George: " + c.lateNightLine(name)
	case hour < 11: // 04:00–10:59
		greet := pick("Selamat pagi", "Good morning", c.Language)
		return fmt.Sprintf("George: %s, bro %s! %s", greet, name, c.morningQuote())
	case hour < 15: // 11:00–14:59
		greet := pick("Selamat siang", "Good afternoon", c.Language)
		return fmt.Sprintf("George: %s, bro %s!", greet, name)
	case hour < 18: // 15:00–17:59
		greet := pick("Selamat sore", "Good afternoon", c.Language)
		return fmt.Sprintf("George: %s, bro %s!", greet, name)
	default: // 18:00–23:59
		greet := pick("Selamat malam", "Good evening", c.Language)
		return fmt.Sprintf("George: %s, bro %s!", greet, name)
	}
}

// pick returns id or en depending on the active language - keeps the branches above short.
func pick(id, en string, lang Language) string {
	if lang == English {
		return en
	}
	return id
}

// lateNightLine is George's caring, friend-toned line for the "still up" hours.
func (c Config) lateNightLine(name string) string {
	if c.Language == English {
		return fmt.Sprintf("Whoa, still up this late, bro %s? Don't push yourself too hard - I'm here if you need to talk something out.", name)
	}
	return fmt.Sprintf("Masih melek jam segini, bro %s? Jangan begadang mulu, tapi gapapa kalau emang lagi butuh temen ngobrol.", name)
}

// specialDate is a fixed Gregorian-calendar date George greets specially for.
// text receives the current time so year-dependent phrasing (e.g. an anniversary
// count) is computed on the fly instead of hardcoded.
type specialDate struct {
	month time.Month
	day   int
	text  func(now time.Time, name string, lang Language) string
}

// specialDates is George's list of days worth a special greeting. Add more here as needed.
//
// Note: this only handles fixed Gregorian dates. Movable holidays that follow the
// lunar/Hijri calendar (Idul Fitri, Idul Adha, Nyepi, Imlek, etc.) shift every year
// and can't be derived from month/day alone - see the references below for how to
// add those with a small yearly-updated date table or a hijri-calendar library.
var specialDates = []specialDate{
	{time.January, 1, func(now time.Time, name string, lang Language) string {
		return pick(
			fmt.Sprintf("Selamat Tahun Baru %d, bro %s! 🎉 Semoga tahun ini makin cuan dan makin jago ngoding.", now.Year(), name),
			fmt.Sprintf("Happy New Year %d, bro %s! 🎉 Here's to a year of shipping good code.", now.Year(), name),
			lang,
		)
	}},
	{time.August, 17, func(now time.Time, name string, lang Language) string {
		years := now.Year() - 1945
		return pick(
			fmt.Sprintf("Selamat Hari Kemerdekaan Indonesia ke-%d, bro %s! 🇮🇩 Merdeka!", years, name),
			fmt.Sprintf("Happy %dth Indonesian Independence Day, bro %s! 🇮🇩 Merdeka!", years, name),
			lang,
		)
	}},
	{time.December, 25, func(now time.Time, name string, lang Language) string {
		return pick(
			fmt.Sprintf("Selamat Natal, bro %s! 🎄 Semoga harinya anget kayak kopi susu.", name),
			fmt.Sprintf("Merry Christmas, bro %s! 🎄", name),
			lang,
		)
	}},
	{time.December, 31, func(now time.Time, name string, lang Language) string {
		return pick(
			fmt.Sprintf("Malam tahun baru nih, bro %s! 🎆 Siap-siap liat kembang api.", name),
			fmt.Sprintf("It's New Year's Eve, bro %s! 🎆 Get ready for the fireworks.", name),
			lang,
		)
	}},
}

// specialDateLine returns George's greeting for today if it matches a known special date.
func (c Config) specialDateLine(now time.Time) (string, bool) {
	for _, d := range specialDates {
		if now.Month() == d.month && now.Day() == d.day {
			return d.text(now, c.UserName, c.Language), true
		}
	}
	return "", false
}

// morningQuotes are short pep-talk lines George can pair with a morning greeting.
var morningQuotesID = []string{
	"Gas terus bro, hari ini juga bisa produktif kayak kemarin 💪",
	"Inget, satu commit kecil hari ini lebih baik daripada nol commit selamanya.",
	"Ngopi dulu, terus kita bantai to-do list hari ini bareng-bareng.",
	"Bug kemarin udah lewat, hari ini fresh start. Semangat bro!",
}

var morningQuotesEN = []string{
	"Let's go bro, today can be just as productive as yesterday 💪",
	"Remember: one small commit today beats zero commits forever.",
	"Grab your coffee, then let's crush today's to-do list together.",
	"Yesterday's bugs are history - fresh start today. You got this!",
}

// morningQuote picks a random pep-talk line to pair with the morning greeting.
func (c Config) morningQuote() string {
	quotes := morningQuotesID
	if c.Language == English {
		quotes = morningQuotesEN
	}
	return quotes[rand.IntN(len(quotes))]
}
