package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/Vong3432/go-web-url-summarizer/internal/handler"
	"github.com/Vong3432/go-web-url-summarizer/internal/scraper"
	"github.com/Vong3432/go-web-url-summarizer/internal/summarizer"
)

type application struct {
	port           string
	summarizer     *summarizer.Summarizer
	maxUrlsAllowed int
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(httprate.LimitByIP(1, 10*time.Second))
	r.Post("/summarize", handler.NewSummarizeHandler(scraper.Fetch, app.summarizer, app.maxUrlsAllowed).ServeHTTP)
	// r.Get("/swagger/*", httpSwagger.WrapHandler)
	return r
}

func (app *application) run(h http.Handler) error {
	srv := &http.Server{
		Addr:    ":" + app.port,
		Handler: h,
	}

	log.Printf("listening on :%s", app.port)
	return srv.ListenAndServe()
}
