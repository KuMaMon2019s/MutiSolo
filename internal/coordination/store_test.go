package coordination

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestStoreWithStateSerializesConcurrentUpdates(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "state.json"))

	var wg sync.WaitGroup
	for _, id := range []string{"task-a", "task-b"} {
		wg.Add(1)
		go func(taskID string) {
			defer wg.Done()
			err := store.WithState(func(state *State) (bool, error) {
				_, err := CreateTask(state, taskID, []string{"code"})
				return err == nil, err
			})
			if err != nil {
				t.Errorf("WithState(%s) returned error: %v", taskID, err)
			}
		}(id)
	}
	wg.Wait()

	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if findTaskIndex(state, "task-a") < 0 {
		t.Fatal("task-a was not saved")
	}
	if findTaskIndex(state, "task-b") < 0 {
		t.Fatal("task-b was not saved")
	}
}
