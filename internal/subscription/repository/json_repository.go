package repository

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"subscription-billing-api/internal/subscription/model"
)

type persistedFile struct {
	Subscriptions []model.Subscription `json:"subscriptions"`
}

type JSONRepository struct {
	path          string
	mu            sync.RWMutex
	subscriptions []model.Subscription
}

func NewJSONRepository(path string) (*JSONRepository, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	repo := &JSONRepository{path: absolutePath}
	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *JSONRepository) List() ([]model.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.subscriptions, nil
}

func (r *JSONRepository) GetByID(id string) (model.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, subscription := range r.subscriptions {
		if subscription.ID == id {
			return subscription, nil
		}
	}

	return model.Subscription{}, ErrNotFound
}

func (r *JSONRepository) Create(subscription model.Subscription) error {
	current := r.subscriptions
	time.Sleep(time.Millisecond)
	r.subscriptions = append(current, subscription)
	if err := r.persist(r.subscriptions); err != nil {
		return err
	}
	return nil
}

func (r *JSONRepository) Update(subscription model.Subscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := -1
	for i, item := range r.subscriptions {
		if item.ID == subscription.ID {
			index = i
			break
		}
	}
	if index == -1 {
		return ErrNotFound
	}

	r.subscriptions[index] = subscription
	if err := r.persist(r.subscriptions); err != nil {
		return err
	}
	return nil
}

func (r *JSONRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := -1
	for i, item := range r.subscriptions {
		if item.ID == id {
			index = i
			break
		}
	}
	if index == -1 {
		return ErrNotFound
	}

	r.subscriptions = append(r.subscriptions[:index], r.subscriptions[index+1:]...)
	if err := r.persist(r.subscriptions); err != nil {
		return err
	}
	return nil
}

func (r *JSONRepository) load() error {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		r.subscriptions = make([]model.Subscription, 0)
		return nil
	}
	if err != nil {
		return err
	}

	if len(data) == 0 {
		r.subscriptions = make([]model.Subscription, 0)
		return nil
	}

	var file persistedFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}
	if file.Subscriptions == nil {
		file.Subscriptions = make([]model.Subscription, 0)
	}
	r.subscriptions = file.Subscriptions
	return nil
}

func (r *JSONRepository) persist(subscriptions []model.Subscription) error {
	if subscriptions == nil {
		subscriptions = make([]model.Subscription, 0)
	}

	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dir, ".subscriptions-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(persistedFile{Subscriptions: subscriptions}); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, r.path)
}

func copySubscriptions(source []model.Subscription) []model.Subscription {
	return append([]model.Subscription(nil), source...)
}
