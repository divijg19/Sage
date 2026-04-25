package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/divijg19/sage/internal/entryflow"
	"github.com/divijg19/sage/internal/event"
)

func TestLoadChronicleDataCmd_MergesConfiguredTagsAndEvents(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setConfiguredTags([]string{"ops", "auth"}); err != nil {
		t.Fatalf("setConfiguredTags: %v", err)
	}

	s, err := openGlobalStore()
	if err != nil {
		t.Fatalf("openGlobalStore: %v", err)
	}

	base := time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)
	appendChronicleEvent(t, s, event.Event{
		ID:        "evt-1",
		Timestamp: base,
		Project:   "alpha",
		Kind:      event.RecordKind,
		Title:     "Investigate auth cache",
		Content:   "Added context",
		Tags:      []string{"auth", "backend"},
	})
	appendChronicleEvent(t, s, event.Event{
		ID:        "evt-2",
		Timestamp: base.Add(1 * time.Hour),
		Project:   "beta",
		Kind:      event.DecisionKind,
		Title:     "Choose sqlite wal",
		Content:   "Durability first",
		Tags:      []string{"qa"},
	})

	msg := loadChronicleDataCmdWithHighlight(2)().(chronicleDataLoadedMsg)
	if msg.err != nil {
		t.Fatalf("unexpected load error: %v", msg.err)
	}
	if msg.highlight != 2 {
		t.Fatalf("expected highlight 2, got %d", msg.highlight)
	}
	if len(msg.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(msg.events))
	}

	wantTags := []string{"ops", "auth", "backend", "qa"}
	if strings.Join(msg.tags, ",") != strings.Join(wantTags, ",") {
		t.Fatalf("unexpected union tags\nwant: %#v\n got: %#v", wantTags, msg.tags)
	}
}

func TestFinishQuickEntry_SavedReloadsWithHighlight(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	prepared := entryflow.PrepareInitialBuffer("Chronicle note", "record", "", "")
	m := newChronicleModel(chronicleOptions{})
	m.showQuick = true
	m.pending = &chroniclePendingEditor{
		launch: &editorLaunch{tempPath: writeEditorTempFile(t, prepared.Body+"\nAdded context.\n")},
		request: entryflow.FinalizeRequest{
			Title:        "Chronicle note",
			ExplicitKind: "record",
			SeedKind:     prepared.SeedKind,
			InitialBody:  prepared.Body,
			Project:      "alpha",
			Tags:         []string{"auth"},
		},
	}

	model, cmd := m.finishQuickEntry(nil)
	after := model.(chronicleModel)
	if after.status != "Entry recorded" {
		t.Fatalf("unexpected save status: %q", after.status)
	}
	if after.statusTone != chronicleStatusSuccess {
		t.Fatalf("expected success tone, got %q", after.statusTone)
	}
	if after.showQuick {
		t.Fatalf("expected quick entry overlay to close after save")
	}

	msg := cmd().(chronicleDataLoadedMsg)
	if msg.err != nil {
		t.Fatalf("unexpected reload error: %v", msg.err)
	}
	if msg.highlight != 1 {
		t.Fatalf("expected highlight for saved seq 1, got %d", msg.highlight)
	}
	if len(msg.events) != 1 {
		t.Fatalf("expected one saved event after reload, got %d", len(msg.events))
	}
	if msg.events[0].Title != "Chronicle note" {
		t.Fatalf("unexpected saved event: %#v", msg.events[0])
	}
	if !strings.Contains(msg.events[0].Content, "Added context.") {
		t.Fatalf("expected saved content to include edited notes, got %#v", msg.events[0])
	}
}

func TestFinishQuickEntry_OutcomeStatuses(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(t *testing.T)
		initialBody string
		edited      string
		execErr     error
		wantStatus  string
		wantTone    chronicleStatusTone
	}{
		{
			name:        "canceled editor",
			initialBody: entryflow.PrepareInitialBuffer("Canceled note", "record", "", "").Body,
			execErr:     &exec.ExitError{},
			wantStatus:  "Editor canceled",
			wantTone:    chronicleStatusWarn,
		},
		{
			name:        "unchanged body",
			initialBody: entryflow.PrepareInitialBuffer("Unchanged note", "record", "", "").Body,
			edited:      entryflow.PrepareInitialBuffer("Unchanged note", "record", "", "").Body,
			wantStatus:  "No changes recorded",
			wantTone:    chronicleStatusInfo,
		},
		{
			name:        "empty content",
			initialBody: entryflow.PrepareInitialBuffer("Empty note", "record", "", "").Body + "\nplaceholder",
			edited:      entryflow.PrepareInitialBuffer("Empty note", "record", "", "").Body,
			wantStatus:  "Entry was empty",
			wantTone:    chronicleStatusWarn,
		},
		{
			name:        "duplicate latest entry",
			initialBody: entryflow.PrepareInitialBuffer("Duplicate note", "record", "", "").Body,
			edited:      entryflow.PrepareInitialBuffer("Duplicate note", "record", "", "").Body + "\nAdded context.\n",
			setupStore: func(t *testing.T) {
				_, _, cleaned := entryflow.ExtractMetaAndBodyFromEditor(entryflow.PrepareInitialBuffer("Duplicate note", "record", "", "").Body + "\nAdded context.\n")
				s, err := openGlobalStore()
				if err != nil {
					t.Fatalf("openGlobalStore: %v", err)
				}
				appendChronicleEvent(t, s, event.Event{
					ID:        "evt-existing",
					Timestamp: time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
					Project:   "alpha",
					Kind:      event.RecordKind,
					Title:     "Duplicate note",
					Content:   cleaned,
					Tags:      []string{"auth"},
				})
			},
			wantStatus: "Duplicate entry skipped",
			wantTone:   chronicleStatusWarn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			if tt.setupStore != nil {
				tt.setupStore(t)
			}

			prepared := entryflow.PrepareInitialBuffer("Test note", "record", "", "")
			initialBody := tt.initialBody
			if initialBody == "" {
				initialBody = prepared.Body
			}

			m := newChronicleModel(chronicleOptions{})
			m.showQuick = true
			m.pending = &chroniclePendingEditor{
				launch: &editorLaunch{tempPath: writeEditorTempFile(t, tt.edited)},
				request: entryflow.FinalizeRequest{
					Title:        strings.TrimSpace(strings.TrimPrefix(strings.Split(initialBody, "\n")[1], "title:")),
					ExplicitKind: "record",
					SeedKind:     prepared.SeedKind,
					InitialBody:  initialBody,
					Project:      "alpha",
					Tags:         []string{"auth"},
				},
			}

			model, _ := m.finishQuickEntry(tt.execErr)
			after := model.(chronicleModel)
			if after.status != tt.wantStatus {
				t.Fatalf("unexpected status\nwant: %q\n got: %q", tt.wantStatus, after.status)
			}
			if after.statusTone != tt.wantTone {
				t.Fatalf("unexpected status tone\nwant: %q\n got: %q", tt.wantTone, after.statusTone)
			}
			if after.showQuick {
				t.Fatalf("expected quick entry overlay to close")
			}
		})
	}
}

func appendChronicleEvent(t *testing.T, s entryflow.Store, e event.Event) {
	t.Helper()
	if err := s.Append(e); err != nil {
		t.Fatalf("Append: %v", err)
	}
}

func writeEditorTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chronicle-editor.md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}
