package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"log"
	"net/http"
)

func main() {
	var (
		listen     = flag.String("listen", "127.0.0.1:7000", "REST API, e.g. 127.0.0.1:7000")
		dbFile     = flag.String("dbfile", "db.gob", "DB file")
		dataDir    = flag.String("datadir", "./data", "Data directory")
		readToken  = flag.String("readtoken", "", "Read-only token")
		writeToken = flag.String("writetoken", "", "Write token")
	)
	flag.Parse()

	db := NewDatabase(*dbFile, *dataDir)
	restapi := NewRestAPI(*readToken, *writeToken, db)
	go func() {
		log.Printf("Start RestAPI listening on %q", *listen)
		if err := http.ListenAndServe(*listen, restapi); err != nil {
			log.Fatalf("HTTP Server crashed: %q", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)

	s := <-c
	log.Printf("Received signal %q, shut down gracefully\n", s)
	err := db.SaveAndShutdown()
	if err != nil {
		log.Printf("Error while shutting down database: %q", err)
	}

	log.Printf("Graceful shutdown complete")
}
