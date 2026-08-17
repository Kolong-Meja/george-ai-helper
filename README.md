# George — local AI assistant (skeleton)

Skeleton CLI untuk George: asisten AI lokal yang jalan 100% di mesin kamu sendiri lewat
[Ollama](https://ollama.com), tanpa API berbayar pihak ke-3. Default Bahasa Indonesia,
bisa switch ke English.

## Struktur project

```
george/
├── main.go                  # entry point, CLI trigger, chat loop
├── internal/
│   ├── ollama/client.go     # HTTP client ke Ollama REST API (localhost:11434)
│   ├── config/config.go     # persona, system prompt, default bahasa & model
│   └── router/router.go     # rule-based command (file search) — skip LLM sama sekali
└── go.mod
```

Sudah dicoba build & jalan pakai Go 1.22 di sandbox — kompatibel dengan Go 1.26.5 kamu
karena cuma pakai standard library (net/http, encoding/json, regexp, dst), nggak ada
dependency eksternal yang perlu di-download.

## Cara pakai

Prasyarat: Ollama sudah jalan (`ollama serve` biasanya otomatis jalan sebagai service
setelah install) dan model `qwen2.5:3b` sudah di-pull.

```bash
go build -o george .
sudo mv george /usr/local/bin/    # biar bisa dipanggil dari mana aja

# panggil tanpa argumen -> masuk mode chat interaktif
george

# panggil dengan argumen -> one-shot command
george cariin file ABC.txt
george halo, lagi ngapain?

# switch bahasa untuk sesi itu
george --lang en how are you doing?
```

## Supaya "hello george" juga bisa dipanggil

Karena shell motong per kata, `hello george` itu artinya command `hello` dengan
argumen `george`. Tambahin function ini ke `~/.bashrc` (atau `~/.zshrc`) biar
frasa itu ke-forward ke binary George:

```bash
hello() {
  if [ "$1" = "george" ]; then
    shift
    george "$@"
  else
    command hello "$@" 2>/dev/null || echo "hello: command not found"
  fi
}
```

Reload dengan `source ~/.bashrc`, terus tinggal ketik `hello george` di terminal
atau tilix.

## Extension point yang belum digarap (sengaja, biar skeleton-nya jelas dulu)

- **Persist pilihan bahasa** — sekarang `--lang` cuma berlaku per sesi. Tinggal simpan
  ke file kecil (misal `~/.config/george/config.json`) dan baca di `config.Default()`.
- **Systemd service** — biar George standby dan nggak cold-start tiap dipanggil.
  Contoh unit file ada di `george.service`.
- **Streaming response** — sekarang nunggu jawaban penuh (`Stream: false`). Kalau mau
  George "ngetik" kata per kata, ganti ke `Stream: true` dan baca response sebagai
  NDJSON stream di `ollama/client.go`.
- **Router pattern** — `searchPattern` di `router.go` baru nangkep satu pola. Tambah
  regex/intent baru di file yang sama seiring kamu nambah command deterministik lain.
