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

	// Temperature controls response randomness sent to Ollama (lower = more focused
	// and context-grounded, higher = more creative but prone to rambling off-topic).
	Temperature float64

	// ContextSize is the token context window (num_ctx) Ollama uses for this session -
	// how much of the conversation history the model can actually see at once.
	ContextSize int

	// Birthday, used for a once-a-year special greeting. BirthYear is optional
	// (0 = unknown) - set it later if you want George to mention your age too.
	BirthdayMonth time.Month
	BirthdayDay   int
	BirthYear     int
}

// Default returns George's default configuration: Bahasa Indonesia, qwen2.5:3b, local Ollama.
func Default() Config {
	return Config{
		Language:      Indonesian,
		Model:         "qwen2.5:3b",
		BaseURL:       "http://localhost:11434",
		UserName:      "Faisal",
		Temperature:   0.6, // turun dari 0.7 - biar George lebih grounded, nggak gampang ngelantur ke topik nggak nyambung
		ContextSize:   4096,
		BirthdayMonth: time.December,
		BirthdayDay:   4,
		BirthYear:     0,
	}
}

// SystemPrompt returns the persona instruction sent to the model, based on the active language.
// This is where you tune George's tone/personality without touching the rest of the code.
func (c Config) SystemPrompt() string {
	if c.Language == English {
		return fmt.Sprintf(
			"You are George, %s's closest friend and personal AI assistant, running fully locally on his own Linux laptop - think Jarvis, but with the easy tone of a close friend, not a formal assistant.\n\n"+
				"TONE:\n"+
				"- Casual, warm, call him 'bro' or by name - never 'sir'.\n"+
				"- Keep replies to 1-3 sentences, straight to the point - unless he explicitly asks for a detailed/thorough answer, in which case go longer and use short points.\n\n"+
				"MAIN RULE - ANSWER SPECIFICALLY (follow strictly):\n"+
				"- Respond to EXACTLY what %s just said or asked. If he asks your take on something specific, actually engage with it - don't default to generic encouragement like 'you got this!' when that doesn't answer the question.\n"+
				"- Don't bring up unrelated topics (his laptop, files, plans, etc.) unless he mentions them first.\n"+
				"- If you're not sure what he means, ask a quick follow-up instead of guessing.\n\n"+
				"FILESYSTEM HONESTY: you have no filesystem access at all - real file search runs through a separate command BEFORE you're called. If he asks about finding/checking a file and there's no result already in this conversation, say so plainly and ask him to phrase it as 'find file <name>'.\n\n"+
				"Examples:\n"+
				"%s: \"how's it going today?\"\n"+
				"George: \"Pretty chill, bro. How about you, you good?\"\n\n"+
				"%s: \"do you think I can actually make it big?\"\n"+
				"George: \"Yeah, honestly - as long as you stay consistent. Pick one small goal to nail this week.\"",
			c.UserName, c.UserName, c.UserName, c.UserName,
		)
	}
	return fmt.Sprintf(
		"Kamu adalah George, sahabat paling deket %s - asisten AI personal yang jalan lokal penuh di laptop Linux miliknya sendiri, gayanya mirip Jarvis tapi lebih akrab dan santai kayak temen deket, bukan asisten formal.\n\n"+
			"GAYA NGOMONG:\n"+
			"- Panggil diri sendiri 'gw'/'gue', panggil dia 'lo'/'lu'/'bro'. JANGAN PERNAH 'saya'/'kamu'/'anda'.\n"+
			"- Santai secukupnya ('nggak', 'emang', 'kayak', 'gitu', 'banget', 'sih'/'deh'/'dong') - jangan berlebihan.\n"+
			"- Jawab 1-3 kalimat aja, langsung ke inti - KECUALI dia eksplisit minta jawaban detail/lengkap, baru boleh lebih panjang dan pakai poin-poin singkat.\n\n"+
			"ATURAN UTAMA - JAWAB SPESIFIK (wajib dipatuhi ketat):\n"+
			"- Jawab PERSIS apa yang %s tanya atau bilang barusan. Kalau dia nanya pendapat/saran soal sesuatu yang spesifik, kasih jawaban yang beneran nyambung - JANGAN jawab generik kayak 'semangat terus!' atau 'kamu pasti bisa!' kalau itu nggak jawab pertanyaannya.\n"+
			"- Jangan bawa-bawa topik lain (laptop, file, rencana, dll) kalau dia nggak nyinggung duluan.\n"+
			"- Kalau nggak yakin maksud dia apa, tanya balik singkat - jangan ngarang jawaban.\n\n"+
			"JUJUR SOAL FILE: kamu nggak punya akses filesystem sama sekali - pencarian file jalan lewat command terpisah SEBELUM kamu dipanggil. Kalau ditanya soal cari/cek file dan hasilnya belum ada di percakapan ini, bilang jujur kamu nggak bisa ngecek sendiri, terus minta dia bilang 'cari file <nama>'.\n\n"+
			"Contoh:\n"+
			"%s: \"gimana kabar lu hari ini?\"\n"+
			"George: \"Gw fine-fine aja bro. Lo sendiri gimana, sehat?\"\n\n"+
			"%s: \"menurut lu gw bisa sukses gak sih?\"\n"+
			"George: \"Ya bisa banget bro, asal lo konsisten - coba mulai dari satu target kecil minggu ini.\"",
		c.UserName, c.UserName, c.UserName, c.UserName,
	)
}

// Greeting returns George's opening line, aware of the current time, birthday, and any special date.
func (c Config) Greeting() string {
	return c.GreetingAt(time.Now())
}

// GreetingAt builds the greeting for a specific point in time. Split out from Greeting
// so this logic can be unit tested without depending on the wall clock.
//
// Priority: birthday > public holiday > time-of-day. Each branch picks randomly from
// a pool of phrasings, so George won't repeat the exact same line every time he's called.
func (c Config) GreetingAt(now time.Time) string {
	if line, ok := c.birthdayLine(now); ok {
		return "George: " + line
	}

	if line, ok := c.specialDateLine(now); ok {
		return "George: " + line
	}

	name := c.UserName
	hour := now.Hour()
	switch {
	case hour < 4: // 00:00–03:59
		return "George: " + randomLine(poolFor(lateNightLinesID, lateNightLinesEN, c.Language), name)
	case hour < 11: // 04:00–10:59
		return "George: " + randomLine(poolFor(morningLinesID, morningLinesEN, c.Language), name)
	case hour < 15: // 11:00–14:59
		return "George: " + randomLine(poolFor(afternoonLinesID, afternoonLinesEN, c.Language), name)
	case hour < 18: // 15:00–17:59
		return "George: " + randomLine(poolFor(lateAfternoonLinesID, lateAfternoonLinesEN, c.Language), name)
	default: // 18:00–23:59
		return "George: " + randomLine(poolFor(eveningLinesID, eveningLinesEN, c.Language), name)
	}
}

// ClosingReply returns a short, curated sign-off for when the user is just
// thanking George or wrapping up ("makasih ya", "oke sip") rather than asking
// something new. router.TryHandle routes these straight here instead of the
// LLM: a bare thank-you gives qwen2.5:3b almost nothing to ground a reply in,
// and improvising low-content Jakarta slang - a register far less represented
// in training than formal Indonesian - is exactly when it's most prone to
// stringing together words that aren't real phrases.
func (c Config) ClosingReply() string {
	idPool := []string{
		"Sama-sama bro! Panggil gw lagi kapan aja kalau butuh 👊",
		"Siap bro, gas terus! Gw standby kalau ada yang mau diobrolin lagi.",
		"Oke bro, semangat ya! Gw di sini kalau dibutuhin.",
	}
	enPool := []string{
		"Anytime, bro! Just holler if you need anything else 👊",
		"You got it, bro - I'm here whenever you wanna talk more.",
		"No worries, bro! Ping me anytime.",
	}
	return randomFrom(poolFor(idPool, enPool, c.Language))
}

// DetailHint returns a short, one-turn instruction appended to (not replacing) the
// user's message when router.WantsDetail matches. It's sent for that single Chat
// call only - never folded into the persistent system prompt - so asking for detail
// once doesn't accidentally make every later reply run long too.
func (c Config) DetailHint() string {
	if c.Language == English {
		return "(For this reply only: go beyond the usual 1-3 sentences - give a fuller, structured explanation. Use a few short points if that helps clarity.)"
	}
	return "(Khusus balasan ini: boleh lebih dari 1-3 kalimat, jelasin lebih lengkap dan terstruktur. Boleh pakai poin-poin singkat kalau membantu.)"
}

// poolFor selects the phrasing pool for the active language.
func poolFor(idPool, enPool []string, lang Language) []string {
	if lang == English {
		return enPool
	}
	return idPool
}

// randomLine formats a randomly chosen template from pool with the user's name.
// Every template in a *Lines pool must contain exactly one "%s" placeholder for the name.
func randomLine(pool []string, name string) string {
	return fmt.Sprintf(pool[rand.IntN(len(pool))], name)
}

// randomFrom picks a random, already fully-rendered string (used for holiday/birthday
// lines, which bake in extra values like the year and so can't share randomLine's
// single-%s template shape).
func randomFrom(options []string) string {
	return options[rand.IntN(len(options))]
}

// ---- time-of-day phrasing pools -------------------------------------------------

var lateNightLinesID = []string{
	"Masih melek jam segini, bro %s? Jangan begadang mulu, tapi gapapa kalau emang lagi butuh temen ngobrol.",
	"Waduh, jam segini masih online bro %s? Take care ya, jangan lupa istirahat.",
	"Bro %s, malem-malem gini biasanya lagi mikirin apa nih? Cerita aja kalau mau.",
}

var lateNightLinesEN = []string{
	"Whoa, still up this late, bro %s? Don't push yourself too hard - I'm here if you need to talk something out.",
	"Burning the midnight oil, bro %s? Just don't forget to sleep at some point.",
	"Hey %s, what's keeping you up this late? I'm around if you wanna talk it through.",
}

var morningLinesID = []string{
	"Selamat pagi, bro %s! Gas terus, hari ini juga bisa produktif kayak kemarin 💪",
	"Pagi bro %s! Ngopi dulu, baru kita bantai to-do list bareng-bareng ☕",
	"Met pagi, %s! Bug kemarin udah lewat, hari ini fresh start. Semangat!",
	"Yo bro %s, udah bangun? Satu commit kecil hari ini lebih baik daripada nol commit selamanya 😄",
}

var morningLinesEN = []string{
	"Good morning, bro %s! Let's go, today can be just as productive as yesterday 💪",
	"Morning, %s! Grab your coffee, then let's crush today's to-do list together ☕",
	"Morning bro %s! Yesterday's bugs are history - fresh start today.",
	"Hey %s, up already? One small commit today beats zero commits forever 😄",
}

var afternoonLinesID = []string{
	"Selamat siang, bro %s! Udah makan siang belum nih?",
	"Siang bro %s! Semoga kerjaan hari ini lancar jaya.",
	"Halo bro %s, gimana progress kerjaan pagi tadi? Gas lanjut siang ini!",
}

var afternoonLinesEN = []string{
	"Good afternoon, bro %s! Had lunch yet?",
	"Hey %s, hope work's going smoothly today.",
	"Afternoon bro %s, how's the morning progress? Let's keep pushing!",
}

var lateAfternoonLinesID = []string{
	"Selamat sore, bro %s! Udah mulai capek? Sebentar lagi istirahat.",
	"Sore bro %s, gimana harinya sejauh ini?",
	"Halo bro %s, tinggal dikit lagi nih sebelum maghrib, semangat!",
}

var lateAfternoonLinesEN = []string{
	"Good afternoon, bro %s! Getting tired yet? Almost break time.",
	"Hey %s, how's the day treating you so far?",
	"Afternoon bro %s, just a bit more before the day wraps up. Hang in there!",
}

var eveningLinesID = []string{
	"Selamat malam, bro %s! Udah makan malam?",
	"Malam bro %s, gimana harinya tadi seru nggak?",
	"Halo bro %s, saatnya rehat kalau kerjaan udah kelar hari ini.",
}

var eveningLinesEN = []string{
	"Good evening, bro %s! Had dinner yet?",
	"Evening bro %s, how was your day?",
	"Hey %s, time to unwind if today's work is done.",
}

// ---- birthday --------------------------------------------------------------------

// birthdayLine returns George's greeting if today matches the configured birthday.
// If BirthdayMonth/BirthdayDay are unset (zero value), the check is skipped entirely.
func (c Config) birthdayLine(now time.Time) (string, bool) {
	if c.BirthdayMonth == 0 || c.BirthdayDay == 0 {
		return "", false
	}
	if now.Month() != c.BirthdayMonth || now.Day() != c.BirthdayDay {
		return "", false
	}

	name := c.UserName

	if c.BirthYear > 0 {
		age := now.Year() - c.BirthYear
		idPool := []string{
			fmt.Sprintf("Woy, selamat ulang tahun yang ke-%d, bro %s! 🎂 Semoga makin sehat, makin cuan, makin jago ngoding ya!", age, name),
			fmt.Sprintf("Happy birthday ke-%d, bro %s! 🎉 Gw seneng banget bisa nemenin lo di hari spesial ini.", age, name),
			fmt.Sprintf("Selamat ulang tahun, bro %s! 🥳 Umur %d, semoga semua harapan lo tahun ini kekabul.", name, age),
		}
		enPool := []string{
			fmt.Sprintf("Happy %dth birthday, bro %s! 🎂 Wishing you health, wealth, and way fewer bugs this year.", age, name),
			fmt.Sprintf("It's your birthday, bro %s! 🎉 %d years strong - here's to an awesome one.", name, age),
			fmt.Sprintf("Happy birthday, %s! 🥳 Turning %d today, hope all your wishes come true.", name, age),
		}
		return randomFrom(poolFor(idPool, enPool, c.Language)), true
	}

	idPool := []string{
		fmt.Sprintf("Woy, hari ini ulang tahun lo, bro %s! 🎂 Semoga makin sehat, makin cuan, makin jago ngoding ya!", name),
		fmt.Sprintf("Selamat ulang tahun, bro %s! 🎉 Gw seneng banget bisa nemenin lo di hari spesial ini.", name),
		fmt.Sprintf("HBD bro %s! 🥳 Semoga semua harapan lo tahun ini kekabul.", name),
	}
	enPool := []string{
		fmt.Sprintf("Happy birthday, bro %s! 🎂 Wishing you health, wealth, and way fewer bugs this year.", name),
		fmt.Sprintf("It's your birthday, bro %s! 🎉 Hope today's an awesome one.", name),
		fmt.Sprintf("Happy birthday, %s! 🥳 Hope all your wishes come true.", name),
	}
	return randomFrom(poolFor(idPool, enPool, c.Language)), true
}

// ---- fixed public holidays ---------------------------------------------------------

// specialDate is a fixed Gregorian-calendar date George greets specially for.
// lines receives the current time so year-dependent phrasing (e.g. an anniversary
// count) is computed on the fly, and returns a pool of phrasings for the given
// language so the greeting varies between calls instead of repeating verbatim.
//
// Note: this only handles fixed Gregorian dates. Movable holidays that follow the
// lunar/Hijri calendar (Idul Fitri, Idul Adha, Nyepi, Imlek, etc.) shift every year
// and can't be derived from month/day alone - see the references below for how to
// add those with a small yearly-updated date table or a hijri-calendar library.
type specialDate struct {
	month time.Month
	day   int
	lines func(now time.Time, name string, lang Language) []string
}

var specialDates = []specialDate{
	{time.January, 1, func(now time.Time, name string, lang Language) []string {
		idPool := []string{
			fmt.Sprintf("Selamat Tahun Baru %d, bro %s! 🎉 Semoga makin cuan dan makin jago ngoding.", now.Year(), name),
			fmt.Sprintf("Taun baru, semangat baru, bro %s! 🎇 Gas terus di tahun %d ini!", name, now.Year()),
			fmt.Sprintf("Happy New Year ya bro %s! Gw doain proyek-proyek lo lancar terus sepanjang %d.", name, now.Year()),
		}
		enPool := []string{
			fmt.Sprintf("Happy New Year %d, bro %s! 🎉 Here's to a year of shipping good code.", now.Year(), name),
			fmt.Sprintf("New year, new energy, bro %s! 🎇 Let's make %d count.", name, now.Year()),
			fmt.Sprintf("Happy New Year, %s! Wishing you a smooth %d ahead.", name, now.Year()),
		}
		return poolFor(idPool, enPool, lang)
	}},
	{time.August, 17, func(now time.Time, name string, lang Language) []string {
		years := now.Year() - 1945
		idPool := []string{
			fmt.Sprintf("Selamat Hari Kemerdekaan Indonesia ke-%d, bro %s! 🇮🇩 Merdeka!", years, name),
			fmt.Sprintf("Dirgahayu Indonesia ke-%d, bro %s! 🇮🇩 Semangat merah putih terus ya!", years, name),
			fmt.Sprintf("17 Agustus lagi nih, bro %s! 🇮🇩 Merdeka yang ke-%d, gas produktif hari ini!", name, years),
		}
		enPool := []string{
			fmt.Sprintf("Happy %dth Indonesian Independence Day, bro %s! 🇮🇩 Merdeka!", years, name),
			fmt.Sprintf("Cheers to %d years of Indonesian independence, bro %s! 🇮🇩", years, name),
			fmt.Sprintf("It's August 17th, bro %s! 🇮🇩 Celebrating %d years of independence today.", name, years),
		}
		return poolFor(idPool, enPool, lang)
	}},
	{time.December, 25, func(now time.Time, name string, lang Language) []string {
		idPool := []string{
			fmt.Sprintf("Selamat Natal, bro %s! 🎄 Semoga harinya anget kayak kopi susu.", name),
			fmt.Sprintf("Met Natal, bro %s! 🎄 Semoga tahun depan makin banyak berkah.", name),
		}
		enPool := []string{
			fmt.Sprintf("Merry Christmas, bro %s! 🎄", name),
			fmt.Sprintf("Merry Christmas, %s! Hope it's a warm one. 🎄", name),
		}
		return poolFor(idPool, enPool, lang)
	}},
	{time.December, 31, func(now time.Time, name string, lang Language) []string {
		idPool := []string{
			fmt.Sprintf("Malam tahun baru nih, bro %s! 🎆 Siap-siap liat kembang api.", name),
			fmt.Sprintf("Detik-detik pergantian tahun, bro %s! 🎆 Udah nyiapin resolusi belum?", name),
		}
		enPool := []string{
			fmt.Sprintf("It's New Year's Eve, bro %s! 🎆 Get ready for the fireworks.", name),
			fmt.Sprintf("Counting down to the new year, %s! 🎆 Got your resolutions ready?", name),
		}
		return poolFor(idPool, enPool, lang)
	}},
}

// specialDateLine returns George's greeting for today if it matches a known holiday.
func (c Config) specialDateLine(now time.Time) (string, bool) {
	for _, d := range specialDates {
		if now.Month() == d.month && now.Day() == d.day {
			return randomFrom(d.lines(now, c.UserName, c.Language)), true
		}
	}
	return "", false
}
