package repository

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"subscription-billing-api/internal/subscription/model"
)

func newTestRepo(t *testing.T) *JSONRepository {
	t.Helper()
	repo, err := NewJSONRepository(filepath.Join(t.TempDir(), "subscriptions.json"))
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	return repo
}

func testSubscription(id string) model.Subscription {
	return model.Subscription{
		ID:              id,
		Name:            "Netflix",
		Amount:          39.9,
		Status:          model.StatusActive,
		NextRenewalDate: time.Now().AddDate(0, 0, 10).Format(model.DateLayout),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
}

func TestJSONRepositoryCRUDAndPersist(t *testing.T) {
	repo := newTestRepo(t)
	first := testSubscription("first")

	if err := repo.Create(first); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByID("first")
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Name != "Netflix" {
		t.Fatalf("unexpected subscription: %#v", got)
	}

	got.Name = "Updated"
	got.Status = model.StatusPaused
	if err := repo.Update(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := repo.Delete("first"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetByID("first"); err != ErrNotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}

	reloaded, err := NewJSONRepository(repo.path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, err := reloaded.GetByID("first"); err != ErrNotFound {
		t.Fatalf("expected delete to persist, got %v", err)
	}
}

func TestJSONRepositoryListReturnsCopy(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.Create(testSubscription("copy")); err != nil {
		t.Fatalf("create: %v", err)
	}

	items, err := repo.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	items[0].Name = "changed-by-caller"

	again, err := repo.List()
	if err != nil {
		t.Fatalf("list again: %v", err)
	}
	if again[0].Name != "Netflix" {
		t.Fatalf("repository state was mutated by caller: %#v", again[0])
	}
}

func TestJSONRepositoryConcurrentCreateAndList(t *testing.T) {
	repo := newTestRepo(t)
	const writerCount = 16
	const writesPerWriter = 12
	const readerCount = 8

	var wg sync.WaitGroup
	for i := 0; i < writerCount; i++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for j := 0; j < writesPerWriter; j++ {
				sub := testSubscription(fmt.Sprintf("writer-%d-item-%d", seed, j))
				if err := repo.Create(sub); err != nil {
					t.Errorf("concurrent create failed: %v", err)
					return
				}
			}
		}(i)
	}
	for i := 0; i < readerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				if _, err := repo.List(); err != nil {
					t.Errorf("concurrent list failed: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	items, err := repo.List()
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	if len(items) != writerCount*writesPerWriter {
		t.Fatalf("expected %d items, got %d", writerCount*writesPerWriter, len(items))
	}
}
