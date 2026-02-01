package main

import (
	"log"
	"net/http"
	"os"

	"github.com/hubenschmidt/go-fissio"
)

func main() {
	client := fissio.NewUnifiedClient(fissio.UnifiedConfig{
		OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
		AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		OllamaURL:    getEnvOr("OLLAMA_URL", "http://localhost:11434/v1"),
	})

	srv := fissio.NewServer(fissio.ServerConfig{
		Client:    client,
		OllamaURL: getEnvOr("OLLAMA_URL", "http://localhost:11434"),
	})

	// In dev mode (DEV=1), serve only API - client runs separately on :3001
	// In prod mode, serve embedded editor at /
	handler := srv.Handler()
	if os.Getenv("DEV") == "" {
		mux := http.NewServeMux()
		mux.Handle("/", fissio.EditorHandler())
		mux.Handle("/api/", http.StripPrefix("/api", srv.Handler()))
		handler = mux
	}

	addr := getEnvOr("ADDR", ":8000")
	log.Printf("Starting fissio server on http://localhost:%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
