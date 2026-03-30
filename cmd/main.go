// Package main is the entry point for the URL summarizer API.
//
//	@title			URL Summarizer API
//	@version		1.0.0
//	@description	Accepts a list of URLs, scrapes their content, and returns a ~500-character AI-generated summary for each one.
//
//	@contact.name	Vong3432
//
//	@host		localhost:8080
//	@BasePath	/
package main

import (
	"log"
	"os"
	"strconv"

	"github.com/Vong3432/go-web-url-summarizer/internal/summarizer"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	maxUrlsAllowed := os.Getenv("MAX_ALLOWED_URLS")
	if maxUrlsAllowed == "" {
		log.Fatal("MAX_ALLOWED_URLS environment variable is required")
	}
	maxUrlsAllowedInt, err := strconv.Atoi(maxUrlsAllowed)
	if err != nil {
		log.Fatal("MAX_ALLOWED_URLS environment variable must be int")
	}

	app := &application{
		port:           port,
		summarizer:     summarizer.NewSummarizer(apiKey),
		maxUrlsAllowed: maxUrlsAllowedInt,
	}

	router := app.mount()
	if err := app.run(router); err != nil {
		log.Println("Server has been shutdown due to error %s", err)
		os.Exit(1)
	}
}
