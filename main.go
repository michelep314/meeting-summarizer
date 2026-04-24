package main

import (
	"bytes"
	"encoding/json"
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
func main() {
	audioFile := "riunione.wav"

	// Registra 60 secondi (modificabile)
	if err := recordAudio(audioFile, 60); err != nil {
		fmt.Fprintf(os.Stderr, "Errore registrazione: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n📝 Trascrizione in corso...")
	transcription, err := transcribeAudio(audioFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Errore: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Trascrizione completata:")
	fmt.Println(transcription)

	fmt.Println("\n🤖 Generazione sintesi...")
	prompt := buildSummaryPrompt(transcription)
	summary, err := askOllama(prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Errore: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n📋 SINTESI RIUNIONE")
	fmt.Println("==================")
	fmt.Println(summary)
}