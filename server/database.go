package main

import (
	"errors"
	"github.com/blang/pushr"
	"sync"
)

type Database struct {
	sync.RWMutex
	repos map[string]*Repository
}

type Repository struct {
	releases []*pushr.Release
}

func NewDatabase(dbfile string, dataDir string) *Database {
	return &Database{}
}

func (d *Database) SaveAndShutdown() error {
	return nil
}

func (d *Database) LatestRelease(namespace, repository, channel string) (*pushr.Release, error) {
	d.RLock()
	defer d.RUnlock()

	repo, found := d.repos[namespace+"/"+repository]
	if !found {
		return nil, errors.New("Repository not found")
	}
	r := repo.releases[len(repo.releases)]
	//TODO: Handle channel
	return r, nil
}
