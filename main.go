package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// --- Ollama ---

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func askOllama(prompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:  "qwen2.5:7b",
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(
		"http://localhost:11434/api/generate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", err
	}

	return ollamaResp.Response, nil
}

// --- Whisper ---

// Chiama transcribe.py come sottoprocesso e ritorna il testo trascritto
func transcribeAudio(audioFile string) (string, error) {
	// exec.Command crea un comando da eseguire
	cmd := exec.Command("python3", "transcribe.py", audioFile)

	// Output() lo esegue e aspetta che finisca, ritornando stdout
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("errore trascrizione: %w", err)
	}

	// Estraiamo solo le righe della trascrizione (quelle dopo "---")
	text := string(output)
	parts := strings.Split(text, "--- TRASCRIZIONE ---")
	if len(parts) < 2 {
		return "", fmt.Errorf("formato trascrizione non riconosciuto")
	}

	return strings.TrimSpace(parts[1]), nil
}

// --- Prompt ---

func buildSummaryPrompt(transcription string) string {
	return fmt.Sprintf(`Sei un assistente che sintetizza riunioni in italiano.

Ecco la trascrizione della riunione:
---
%s
---

Genera una sintesi strutturata con queste sezioni:
1. **Punti principali discussi**
2. **Decisioni prese**
3. **Azioni da fare** (con responsabile se menzionato)

Sii conciso e usa il bullet point per ogni voce.`, transcription)
}

// recordAudio registra dal microfono per `seconds` secondi e salva in `outputFile`
func recordAudio(outputFile string, seconds int) error {
	fmt.Printf("🎙️  Registrazione per %d secondi... (parla ora)\n", seconds)

	cmd := exec.Command(
		"ffmpeg",
		"-y",                        // sovrascrive il file se esiste
		"-f", "pulse",               // usa PulseAudio (Ubuntu)
		"-i", "default",             // microfono di default
		"-t", fmt.Sprintf("%d", seconds), // durata in secondi
		"-ar", "16000",              // 16kHz - il sample rate che Whisper preferisce
		"-ac", "1",                  // mono - sufficiente per il parlato
		outputFile,
	)

	// Colleghiamo stderr di ffmpeg al nostro stderr
	// così vedi eventuali errori di ffmpeg direttamente nel terminale
	cmd.Stderr = os.Stderr

	return cmd.Run()
}


// --- Main ---

func usage() {
	fmt.Fprintln(os.Stderr, "Uso:")
	fmt.Fprintln(os.Stderr, "  go run . -s <sec>  [-r]     registra (e se non -r, trascrive e sintetizza)")
	fmt.Fprintln(os.Stderr, "  go run . -m <min>  [-r]     come sopra, in minuti")
	fmt.Fprintln(os.Stderr, "  go run . -t <file.wav>      trascrive e sintetizza un audio esistente")
	flag.PrintDefaults()
}

func summarize(transcription string) error {
	fmt.Println("\n🤖 Generazione sintesi...")
	prompt := buildSummaryPrompt(transcription)
	summary, err := askOllama(prompt)
	if err != nil {
		return err
	}

	fmt.Println("\n📋 SINTESI RIUNIONE")
	fmt.Println("==================")
	fmt.Println(summary)
	return nil
}

func transcribeAndSummarize(audioFile string) error {
	fmt.Println("\n📝 Trascrizione in corso...")
	transcription, err := transcribeAudio(audioFile)
	if err != nil {
		return err
	}

	fmt.Println("✅ Trascrizione completata:")
	fmt.Println(transcription)

	return summarize(transcription)
}

func main() {
	seconds := flag.Int("s", 0, "durata registrazione in secondi")
	minutes := flag.Int("m", 0, "durata registrazione in minuti")
	recordOnly := flag.Bool("r", false, "registra soltanto, senza trascrivere")
	transcribeFile := flag.String("t", "", "trascrivi (e sintetizza) un audio esistente")
	flag.Usage = usage
	flag.Parse()

	if *transcribeFile != "" {
		if *seconds > 0 || *minutes > 0 || *recordOnly {
			fmt.Fprintln(os.Stderr, "Errore: -t non si combina con -s/-m/-r")
			os.Exit(1)
		}
		if err := transcribeAndSummarize(*transcribeFile); err != nil {
			fmt.Fprintf(os.Stderr, "Errore: %v\n", err)
			os.Exit(1)
		}
		return
	}

	duration := *seconds + *minutes*60
	if duration <= 0 {
		usage()
		os.Exit(1)
	}

	audioFile := "riunione.wav"
	if err := recordAudio(audioFile, duration); err != nil {
		fmt.Fprintf(os.Stderr, "Errore registrazione: %v\n", err)
		os.Exit(1)
	}

	if *recordOnly {
		fmt.Printf("Registrazione salvata in %s\n", audioFile)
		return
	}

	if err := transcribeAndSummarize(audioFile); err != nil {
		fmt.Fprintf(os.Stderr, "Errore: %v\n", err)
		os.Exit(1)
	}
}