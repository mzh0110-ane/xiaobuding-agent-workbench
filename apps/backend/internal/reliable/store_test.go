package reliable

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestThreeRoundsSurviveRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Create(&Task{ID: "task", Project: "project", State: "auditing"}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		got, err := s.StartRound("task")
		if err != nil {
			t.Fatalf("round %d: %v", i, err)
		}
		if got.Round != i {
			t.Fatalf("got round %d, want %d", got.Round, i)
		}
		if _, err := s.Update("task", func(t *Task) error { t.State = "auditing"; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if _, err := s.StartRound("task"); err == nil {
		t.Fatal("fourth round was admitted after restart")
	}
	got, err := s.Get("task")
	if err != nil || got.Round != 3 {
		t.Fatalf("round counter was reset: %v %v", got, err)
	}
}

func TestConcurrentRoundAdmissionHasOneWinner(t *testing.T) {
	s, err := OpenStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if err := s.Create(&Task{ID: "task", Project: "project", State: "auditing"}); err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			if _, err := s.StartRound("task"); err == nil {
				wins.Add(1)
			}
		})
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("got %d admissions, want 1", wins.Load())
	}
}

func TestRestartFinishesPendingCancellationWithoutResettingRounds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Create(&Task{ID: "task", Project: "project", State: stateCancelling, Round: 2}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if err = s.Recover(); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("task")
	if err != nil || got.State != stateCancelled || got.Round != 2 {
		t.Fatalf("pending cancellation was not recovered safely: %+v %v", got, err)
	}
}
