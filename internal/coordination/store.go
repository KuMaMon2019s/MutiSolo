package coordination

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type Store struct {
	path string
}

func NewStore(path string) Store {
	return Store{path: path}
}

func DefaultStatePath() string {
	if path := os.Getenv("OPENCLAW_STATE"); path != "" {
		return path
	}
	return ".openclaw/state.json"
}

func (s Store) Load() (State, error) {
	data, err := os.ReadFile(s.path)
	if err == nil {
		var state State
		if err := json.Unmarshal(data, &state); err != nil {
			return State{}, fmt.Errorf("decode state: %w", err)
		}
		return state, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return State{}, fmt.Errorf("read state: %w", err)
	}

	state := InitialState()
	if err := s.Save(state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s Store) WithState(fn func(*State) (bool, error)) error {
	unlock, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock()

	state, err := s.Load()
	if err != nil {
		return err
	}
	changed, err := fn(&state)
	if err != nil {
		if changed {
			if saveErr := s.Save(state); saveErr != nil {
				return saveErr
			}
		}
		return err
	}
	if changed {
		return s.Save(state)
	}
	return nil
}

func (s Store) Save(state State) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

func (s Store) lock() (func(), error) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	file, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open state lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock state: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}
