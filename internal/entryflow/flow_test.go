package entryflow

import (
	"testing"
	"time"

	"github.com/divijg19/sage/internal/event"
)

type stubStore struct {
	events []event.Event
}

func (s *stubStore) Append(e event.Event) error {
	e.Seq = int64(len(s.events) + 1)
	s.events = append(s.events, e)
	return nil
}

func (s *stubStore) Latest() (*event.Event, error) {
	if len(s.events) == 0 {
		return nil, nil
	}
	e := s.events[len(s.events)-1]
	return &e, nil
}

func (s *stubStore) LatestByProject(project string) (*event.Event, error) {
	for i := len(s.events) - 1; i >= 0; i-- {
		if s.events[i].Project == project {
			e := s.events[i]
			return &e, nil
		}
	}
	return nil, nil
}

func TestFinalize_Saved(t *testing.T) {
	store := &stubStore{}
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	initial := PrepareInitialBuffer("Chronicle note", "record", "", "")

	result, err := Finalize(FinalizeRequest{
		Title:        "Chronicle note",
		ExplicitKind: "record",
		SeedKind:     initial.SeedKind,
		InitialBody:  initial.Body,
		Edited:       initial.Body + "\nAdded context.\n",
		Project:      "alpha",
		Tags:         []string{"auth"},
	}, Dependencies{
		Store: store,
		ResolveKind: func(explicit string, suggested string) (event.EntryKind, error) {
			return event.RecordKind, nil
		},
		Now:   func() time.Time { return now },
		NewID: func() string { return "evt-1" },
	})
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if result.Status != StatusSaved {
		t.Fatalf("expected saved status, got %s", result.Status)
	}
	if result.Event == nil || result.Event.Seq != 1 {
		t.Fatalf("expected saved event with seq 1, got %#v", result.Event)
	}
}

func TestFinalize_UnchangedAndDuplicate(t *testing.T) {
	initial := PrepareInitialBuffer("Chronicle note", "record", "", "")
	edited := initial.Body + "\nAdded context.\n"
	_, _, storedBody := ExtractMetaAndBodyFromEditor(edited)

	store := &stubStore{
		events: []event.Event{
			{
				Seq:       1,
				ID:        "evt-1",
				Timestamp: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC),
				Project:   "alpha",
				Kind:      event.RecordKind,
				Title:     "Chronicle note",
				Content:   storedBody,
				Tags:      []string{"auth"},
			},
		},
	}

	unchanged, err := Finalize(FinalizeRequest{
		Title:        "Chronicle note",
		ExplicitKind: "record",
		SeedKind:     initial.SeedKind,
		InitialBody:  initial.Body,
		Edited:       initial.Body,
		Project:      "alpha",
		Tags:         []string{"auth"},
	}, Dependencies{
		Store: store,
		ResolveKind: func(explicit string, suggested string) (event.EntryKind, error) {
			return event.RecordKind, nil
		},
	})
	if err != nil {
		t.Fatalf("Finalize unchanged: %v", err)
	}
	if unchanged.Status != StatusUnchanged {
		t.Fatalf("expected unchanged status, got %s", unchanged.Status)
	}

	duplicate, err := Finalize(FinalizeRequest{
		Title:        "Chronicle note",
		ExplicitKind: "record",
		SeedKind:     initial.SeedKind,
		InitialBody:  initial.Body,
		Edited:       edited,
		Project:      "alpha",
		Tags:         []string{"auth"},
	}, Dependencies{
		Store: store,
		ResolveKind: func(explicit string, suggested string) (event.EntryKind, error) {
			return event.RecordKind, nil
		},
	})
	if err != nil {
		t.Fatalf("Finalize duplicate: %v", err)
	}
	if duplicate.Status != StatusDuplicate {
		t.Fatalf("expected duplicate status, got %s", duplicate.Status)
	}
}
