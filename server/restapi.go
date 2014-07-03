package main

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
)

type RestAPI struct {
	router     *httprouter.Router
	readToken  string
	writeToken string
	db         *Database
}

func NewRestAPI(readToken string, writeToken string, db *Database) *RestAPI {
	r := &RestAPI{
		router:     httprouter.New(),
		readToken:  readToken,
		writeToken: writeToken,
		db:         db,
	}
	r.registerEndpoints()
	return r
}

func (a *RestAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin,Authorization,Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
	} else {
		a.router.ServeHTTP(w, r)
	}
}

func (a *RestAPI) registerEndpoints() {
	a.router.Handle("GET", "/repos/:namespace/:repository/releases/:channel/latest", a.handleGETRelease)
}

func (a *RestAPI) handleGETRelease(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	release, err := a.db.LatestRelease(params.ByName("namespace"), params.ByName("repository"), params.ByName("channel"))
	if err != nil {
		log.Printf("Get latest release for %s failed: %s", params, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	err = json.NewEncoder(w).Encode(release)
	if err != nil {
		log.Printf("Could not encode release %s to json: %s", release, err)
		w.WriteHeader(500)
		return
	}
}
