package main

import (
	"log"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof/* on http.DefaultServeMux, served below on a loopback-only port
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"song-drill-backend/db"
	"song-drill-backend/handlers"
)

// logRequestStart logs the instant a request is received, before any
// middleware or handler work happens. gin's own request logger only logs
// once a response has actually been written, so a request that never
// completes — the exact "signal timed out, no matching access log line"
// pattern this was added to debug — leaves no trace at all. This turns
// "did the request even reach the Go process" from a guess into a fact:
// if the freeze happens again, check whether an [arrived] line exists for
// the stuck request. If it doesn't, the problem is upstream of Go entirely
// (the proxy, the browser). If it does, whatever's next in the log (or
// the goroutine dump below) says exactly where it got stuck.
func logRequestStart(c *gin.Context) {
	log.Printf("[arrived] %s %s", c.Request.Method, c.Request.URL.Path)
	c.Next()
}

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

	r := gin.New()
	r.Use(logRequestStart, gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: false,
	}))

	api := r.Group("/api/song-drill")
	api.Use(env.WithActiveUser)

	// Registered directly on api (not as sub-groups) for /songs and
	// /profiles, since both need a route matching the bare group path
	// itself ("/songs", "/profiles") alongside their sub-paths — simplest
	// to just spell out the full path on each route than rely on a group's
	// empty-suffix route.
	api.POST("/songs/ingest", env.IngestSong)
	api.GET("/songs", env.ListSongs)
	api.GET("/songs/:id", env.GetSong)
	api.DELETE("/songs/:id", env.DeleteSong)
	api.GET("/songs/:id/lines", env.GetSongLines)

	drill := api.Group("/drill")
	{
		drill.GET("/vocab", env.VocabDrillQueue)
		drill.POST("/vocab/more", env.AddMoreVocab)
		drill.GET("/lines", env.LineDrillQueue)
		drill.POST("/lines/more", env.AddMoreLines)
		drill.POST("/result", env.RecordDrillResult)
	}

	api.GET("/stats", env.GetStats)

	progress := api.Group("/progress")
	{
		progress.GET("/vocab", env.ListVocabProgress)
		progress.POST("/vocab/burn", env.BurnVocabProgress)
		progress.POST("/vocab/reset", env.ResetVocabProgress)
	}

	api.GET("/profiles", env.ListProfiles)
	api.POST("/profiles", env.CreateProfile)
	api.GET("/profiles/active", env.GetActiveProfile)
	api.POST("/profiles/active", env.SetActiveProfile)
	api.PATCH("/profiles/:id", env.UpdateProfile)
	api.DELETE("/profiles/:id", env.DeleteProfile)

	// Debug/profiling endpoints (net/http/pprof, registered on
	// http.DefaultServeMux by the blank import above) — bound to loopback
	// only, on its own port, so it's reachable from this machine but never
	// over Tailscale even though the main API is. If the app ever freezes
	// again, hit http://localhost:6060/debug/pprof/goroutine?debug=2 *while
	// it's stuck* — that dumps every goroutine's current stack trace, which
	// says definitively where something is blocked (waiting on a mutex, a
	// DB call, network I/O, etc.) instead of leaving it to guesswork.
	go func() {
		log.Println("pprof debug server listening on 127.0.0.1:6060")
		log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

	// http.ListenAndServe's bare form uses an http.Server with every timeout
	// at its zero value — in particular no IdleTimeout, so a keep-alive
	// connection the frontend's dev/preview proxy is holding open can sit
	// idle forever from the server's own point of view. If that connection
	// ever goes stale for any reason (however rare), nothing on the Go side
	// forces it closed, so the next request that happens to reuse it from
	// the proxy's pool just hangs — indistinguishable from the backend
	// itself being slow, except no request ever actually lands (see the
	// "signal timed out" reports with no matching access log line). Explicit
	// timeouts bound how long an idle connection survives, so a wedged one
	// gets torn down and the proxy is forced to open a fresh one instead of
	// reusing it indefinitely.
	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("song-drill API listening on %s (db: %s)", addr, dbPath)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
