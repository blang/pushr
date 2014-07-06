package main

import (
	"flag"
	"github.com/blang/pushr"
	"github.com/blang/semver"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type DataStore struct {
	sync.RWMutex
	dataDir  string
	releases map[string]*pushr.Release
}

func (d *DataStore) Filepath(version *pushr.Version) string {
	return filepath.Join(d.dataDir, version.Filename)
}

func main() {
	var (
		listen     = flag.String("listen", "127.0.0.1:7000", "REST API, e.g. 127.0.0.1:7000")
		dataDir    = flag.String("datadir", "./data", "Data directory")
		readToken  = flag.String("readtoken", "", "Read-only token")
		writeToken = flag.String("writetoken", "", "Write token")
	)
	flag.Parse()

	ds, err := buildDataStore(*dataDir)
	if err != nil {
		log.Fatalf("Could not read data dir: %s", err)
	}

	restapi := NewRestAPI(*readToken, *writeToken, ds)
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
	log.Printf("Graceful shutdown complete")
}

func buildDataStore(dataDir string) (*DataStore, error) {
	files, err := ioutil.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}
	ds := &DataStore{
		dataDir:  dataDir,
		releases: make(map[string]*pushr.Release),
	}
	for _, f := range files {
		parts := strings.SplitN(f.Name(), "-", 2)
		if len(parts) != 2 {
			log.Printf("Could not parse filename %s\n", f.Name())
			continue
		}
		ext := filepath.Ext(f.Name())
		if ext == "" {
			log.Printf("Could not find extension of file %s\n", f.Name())
			continue
		}
		versionStr := strings.TrimSuffix(parts[1], ext)
		_, err := semver.New(versionStr)
		if err != nil {
			log.Printf("Could not parse version of file %s\n", f.Name())
			continue
		}
		//TODO: Check filename for invalid chars

		var r *pushr.Release
		r, found := ds.releases[parts[0]]
		if !found {
			r = &pushr.Release{
				Versions: make(map[string]*pushr.Version),
			}
			ds.releases[parts[0]] = r
		}

		_, found = r.Versions[versionStr]
		if found {
			log.Printf("Duplicate version of file %s: %s\n", f.Name(), versionStr)
			continue
		}
		v := &pushr.Version{}
		v.Size = f.Size()
		v.ContentType = mime.TypeByExtension(ext)
		v.Filename = f.Name()
		r.Versions[versionStr] = v
		log.Printf("Read %s: Release %q, Version: %q, Content-Type: %q, Size: %dB", f.Name(), parts[0], versionStr, v.ContentType, v.Size)
	}

	return ds, nil
}
