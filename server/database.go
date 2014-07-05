package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/blang/pushr"
	"log"
	"os"
	"sync"
)

type Database struct {
	sync.RWMutex
	dbFile  string
	dataDir string
	store   *Store
}

type Store struct {
	repos map[string]*Repository
}

type Repository struct {
	releases []*pushr.Release
}

func newRepository() *Repository {
	return &Repository{}
}

func newStore() *Store {
	return &Store{
		repos: make(map[string]*Repository),
	}
}

func NewDatabase(dbFile string, dataDir string) *Database {
	db := &Database{
		dbFile:  dbFile,
		dataDir: dataDir,
	}

	store, err := restoreStore(dbFile)
	if err != nil {
		log.Printf("Error while reading db from file %s: %s", dbFile, err)
		db.store = newStore()
	} else {
		db.store = store
	}
	return db
}

func restoreStore(filename string) (*Store, error) {
	if filename == "" {
		return nil, errors.New("Store file could not be empty")
	}
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return nil, fmt.Errorf("Could not open store file %q: %q", filename, err)
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var store Store
	err = dec.Decode(&store)
	if err != nil {
		return nil, fmt.Errorf("Could not decode store from file %q: %q", filename, err)
	}
	return &store, nil
}

func (s *Store) saveStore(filename string) error {
	if filename == "" {
		return errors.New("Store file could not be empty")
	}
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return fmt.Errorf("Could not open store file %q: %q", filename, err)
	}
	defer f.Close()
	dec := gob.NewEncoder(f)
	err = dec.Encode(s)
	if err != nil {
		return fmt.Errorf("Could not encode store to file %q: %q", filename, err)
	}
	return nil
}

func (d *Database) SaveAndShutdown() error {
	d.Lock()
	defer d.Unlock()
	return d.store.saveStore(d.dbFile)
}

func (d *Database) Releases(namespace, repository, channel string) ([]*pushr.Release, error) {
	d.RLock()
	defer d.RUnlock()

	repo, found := d.store.repos[namespace+"/"+repository]
	if !found {
		return nil, errors.New("Repository not found")
	}
	return repo.releases, nil
}

func (d *Database) AddRelease(namespace, repository, channel string, release *Release) error {
	if release == nil {
		return errors.New("Release nil")
	}
	d.Lock()
	var repo *Repository
	repo, found := d.store.repos[namespace+"/"+repository]
	if !found {
		repo = newRepository()
		d.store.repos[namespace+"/"+repository] = repo
	}

	repo.releases = append(repo.releases, release)
	return nil
}
