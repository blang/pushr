package main

import (
	"encoding/json"
	"fmt"
	"github.com/blang/methodr"
	"github.com/blang/pushr"
	"github.com/blang/semver"
	"github.com/gorilla/mux"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type RestAPI struct {
	router     *mux.Router
	readToken  string
	writeToken string
	ds         *DataStore
}

func NewRestAPI(readToken string, writeToken string, ds *DataStore) *RestAPI {
	r := &RestAPI{
		router:     mux.NewRouter(),
		readToken:  readToken,
		writeToken: writeToken,
		ds:         ds,
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
	a.router.Handle("/ping", http.HandlerFunc(a.handlePing))
	a.router.Handle("/releases/{name}", methodr.GET(a.readAccess(http.HandlerFunc(a.handleReleaseList))))
	a.router.Handle("/releases/{name}/{version}", methodr.GET(a.readAccess(http.HandlerFunc(a.handleGetRelease))))
	a.router.Handle("/releases/{name}/{version}/{filename}", methodr.POST(a.writeAccess(http.HandlerFunc(a.handlePostRelease))))
}

func (a *RestAPI) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (a *RestAPI) handleReleaseList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name, found := vars["name"]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	a.ds.RLock()
	defer a.ds.RUnlock()
	release, found := a.ds.releases[name]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(release)
}

func (a *RestAPI) handleGetRelease(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name, found := vars["name"]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	versionStr, found := vars["version"]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	a.ds.RLock()
	defer a.ds.RUnlock()

	release, found := a.ds.releases[name]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	version, found := release.Versions[versionStr]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		json.NewEncoder(w).Encode(version)
	} else {
		filename := a.ds.Filepath(version)
		w.Header().Set("Content-Type", version.ContentType)
		w.Header().Set("Content-Disposition", "attachment; filename=\""+version.Filename+"\"")
		http.ServeFile(w, r, filename)
	}
}

func (a *RestAPI) handlePostRelease(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name, found := vars["name"]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	versionStr, found := vars["version"]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_, err := semver.New(versionStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error processing version: %s", err)
		return
	}

	filename, found := vars["filename"]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	a.ds.Lock()
	defer a.ds.Unlock()

	release, found := a.ds.releases[name]
	if !found {
		release = pushr.NewRelease()
		a.ds.releases[name] = release
	}

	version, found := release.Versions[versionStr]
	if found {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, "Error: Version already found: %s", version)
		return
	}
	fileext := filepath.Ext(filename)
	if fileext == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: No File extension on %q found", filename)
		return
	}
	newFilename := name + "-" + versionStr + fileext

	version = pushr.NewVersion()
	version.Filename = newFilename
	version.ContentType = mime.TypeByExtension(fileext)
	filePath := a.ds.Filepath(version)

	o, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %s", err)
		return
	}
	defer o.Close()
	defer r.Body.Close()
	written, err := io.Copy(o, r.Body)
	if err != nil || written == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: Written %d, %s", written, err)
		os.Remove(o.Name())
		return
	}
	version.Size = written
	release.Versions[versionStr] = version
	w.WriteHeader(http.StatusCreated)
}

func (a *RestAPI) readAccess(handler http.Handler) http.Handler {
	// Allow access if no readToken is set
	if a.readToken == "" {
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-PUSHR-TOKEN") == a.readToken || r.FormValue("token") == a.readToken {
			handler.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})

}

func (a *RestAPI) writeAccess(handler http.Handler) http.Handler {
	if a.writeToken == "" {
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-PUSHR-TOKEN") == a.writeToken || r.FormValue("token") == a.writeToken {
			handler.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
}
