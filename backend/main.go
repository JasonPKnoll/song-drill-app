package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"song-drill-backend/db"
	"song-drill-backend/handlers"
)

func main() {
	dbPath := os.Getenv("SONG_DRILL_DB")
	if dbPath == "" {
		dbPath = "song-drill.db"
	}

	addr := os.Getenv("SONG_DRILL_ADDR")
	if addr == "" {
		addr = ":30001"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	env := handlers.NewEnv(database)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}))

	r.Route("/api/song-drill", func(r chi.Router) {
		r.Route("/songs", func(r chi.Router) {
			r.Post("/ingest", env.IngestSong)
			r.Get("/", env.ListSongs)
			r.Get("/{id}", env.GetSong)
			r.Delete("/{id}", env.DeleteSong)
			r.Get("/{id}/lines", env.GetSongLines)
		})
		r.Route("/drill", func(r chi.Router) {
			r.Get("/vocab", env.VocabDrillQueue)
			r.Get("/lines", env.LineDrillQueue)
			r.Post("/result", env.RecordDrillResult)
		})
		r.Get("/stats", env.GetStats)
	})

	log.Printf("song-drill API listening on %s (db: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
