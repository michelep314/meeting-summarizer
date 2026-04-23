package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

// Queste due struct rispecchiano esattamente il JSON che
// mandiamo e riceviamo dall'API di Ollama

type OllamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
}

type OllamaResponse struct {
    Response string `json:"response"`
}

func askOllama(prompt string) (string, error) {
    // 1. Costruiamo il body della richiesta
    reqBody := OllamaRequest{
        Model:  "qwen2.5:7b",
        Prompt: prompt,
        Stream: false,
    }

    // 2. Serializziamo la struct in JSON
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return "", err
    }

    // 3. Facciamo la richiesta HTTP POST
    resp, err := http.Post(
        "http://localhost:11434/api/generate",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // 4. Leggiamo il body della risposta
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    // 5. Deserializziamo il JSON nella struct
    var ollamaResp OllamaResponse
    if err := json.Unmarshal(body, &ollamaResp); err != nil {
        return "", err
    }

    return ollamaResp.Response, nil
}

func main() {
    prompt := "Dimmi in 3 bullet point cosa sai fare."

    fmt.Println("Invio richiesta a Ollama...")
    response, err := askOllama(prompt)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Errore: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("\nRisposta:")
    fmt.Println(response)
}