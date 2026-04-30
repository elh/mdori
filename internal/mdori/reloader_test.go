package mdori

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestReloaderMatchesOnlyTargetFileChanges(t *testing.T) {
	t.Parallel()

	r := &reloader{baseName: "note.md"}

	tests := []struct {
		name  string
		event fsnotify.Event
		want  bool
	}{
		{
			name:  "write to target",
			event: fsnotify.Event{Name: "/tmp/note.md", Op: fsnotify.Write},
			want:  true,
		},
		{
			name:  "rename target",
			event: fsnotify.Event{Name: "/tmp/note.md", Op: fsnotify.Rename},
			want:  true,
		},
		{
			name:  "different file",
			event: fsnotify.Event{Name: "/tmp/other.md", Op: fsnotify.Write},
			want:  false,
		},
		{
			name:  "unsupported op",
			event: fsnotify.Event{Name: "/tmp/note.md", Op: fsnotify.Chmod},
			want:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := r.matches(tt.event); got != tt.want {
				t.Fatalf("matches(%v) = %v, want %v", tt.event, got, tt.want)
			}
		})
	}
}
