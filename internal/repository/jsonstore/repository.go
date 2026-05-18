package jsonstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"workout-app/backend/internal/domain/model"
)

type Repository struct {
	mu   sync.Mutex
	path string
	data model.State
}

func New(path string) (*Repository, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	repo := &Repository{path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		repo.data = SeedState()
		return repo, repo.saveLocked()
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		repo.data = SeedState()
		return repo, repo.saveLocked()
	}
	if err := json.Unmarshal(raw, &repo.data); err != nil {
		return nil, err
	}
	if repo.data.Sessions == nil {
		repo.data.Sessions = map[string]int64{}
	}
	return repo, nil
}

func (r *Repository) View(ctx context.Context, fn func(state model.State) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return fn(cloneState(r.data))
}

func (r *Repository) Update(ctx context.Context, fn func(state *model.State) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := fn(&r.data); err != nil {
		return err
	}
	return r.saveLocked()
}

func (r *Repository) saveLocked() error {
	raw, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func cloneState(state model.State) model.State {
	raw, err := json.Marshal(state)
	if err != nil {
		return state
	}
	var cloned model.State
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return state
	}
	return cloned
}
