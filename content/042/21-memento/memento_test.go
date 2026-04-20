package main

import (
	"errors"
	"testing"
	"time"
)

func TestEditorEditing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ops     func(e *Editor)
		want    string
		cursor  int
	}{
		{
			name: "write simples",
			ops: func(e *Editor) {
				e.Write("oi")
			},
			want:   "oi",
			cursor: 2,
		},
		{
			name: "write insere na posição do cursor",
			ops: func(e *Editor) {
				e.Write("abcxyz")
				e.MoveCursor(3)
				e.Write("_")
			},
			want:   "abc_xyz",
			cursor: 4,
		},
		{
			name: "delete retroage cursor",
			ops: func(e *Editor) {
				e.Write("abcdef")
				e.Delete(2)
			},
			want:   "abcd",
			cursor: 4,
		},
		{
			name: "delete com n maior que cursor",
			ops: func(e *Editor) {
				e.Write("abc")
				e.Delete(10)
			},
			want:   "",
			cursor: 0,
		},
		{
			name: "delete no início não faz nada",
			ops: func(e *Editor) {
				e.Write("abc")
				e.MoveCursor(0)
				e.Delete(5)
			},
			want:   "abc",
			cursor: 0,
		},
		{
			name: "movecursor clampa",
			ops: func(e *Editor) {
				e.Write("ola")
				e.MoveCursor(-5)
				e.Write("!")
				e.MoveCursor(999)
			},
			want:   "!ola",
			cursor: 4,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := NewEditor("")
			tc.ops(e)
			if e.Content() != tc.want {
				t.Fatalf("content=%q, quer=%q", e.Content(), tc.want)
			}
			if e.Cursor() != tc.cursor {
				t.Fatalf("cursor=%d, quer=%d", e.Cursor(), tc.cursor)
			}
		})
	}
}

func TestHistoryUndoRedo(t *testing.T) {
	t.Parallel()

	e := NewEditor("")
	h := NewHistory(e, 0)

	h.Save()
	e.Write("A")
	h.Save()
	e.Write("B")
	h.Save()
	e.Write("C")

	if e.Content() != "ABC" {
		t.Fatalf("setup: %q", e.Content())
	}

	if err := h.Undo(); err != nil {
		t.Fatalf("undo1: %v", err)
	}
	if e.Content() != "AB" {
		t.Fatalf("após undo1: %q", e.Content())
	}
	if err := h.Undo(); err != nil {
		t.Fatalf("undo2: %v", err)
	}
	if e.Content() != "A" {
		t.Fatalf("após undo2: %q", e.Content())
	}
	if err := h.Redo(); err != nil {
		t.Fatalf("redo1: %v", err)
	}
	if e.Content() != "AB" {
		t.Fatalf("após redo1: %q", e.Content())
	}

	// nova gravação descarta redos
	e.Write("Z")
	h.Save()
	if err := h.Redo(); !errors.Is(err, ErrHistoryEmpty) {
		t.Fatalf("redo após save devia vazar ErrHistoryEmpty, got %v", err)
	}
}

func TestHistoryEmpty(t *testing.T) {
	t.Parallel()
	h := NewHistory(NewEditor(""), 0)
	if err := h.Undo(); !errors.Is(err, ErrHistoryEmpty) {
		t.Fatalf("undo vazio: %v", err)
	}
	if err := h.Redo(); !errors.Is(err, ErrHistoryEmpty) {
		t.Fatalf("redo vazio: %v", err)
	}
}

func TestHistoryLimit(t *testing.T) {
	t.Parallel()
	e := NewEditor("")
	h := NewHistory(e, 2)
	for i := 0; i < 5; i++ {
		h.Save()
		e.Write("x")
	}
	u, _ := h.Len()
	if u != 2 {
		t.Fatalf("esperava limite=2 no undo, got %d", u)
	}
}

func TestSnapshotMetadata(t *testing.T) {
	t.Parallel()
	e := NewEditor("oi")
	fixed := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	e.clock = func() time.Time { return fixed }
	m := e.Snapshot()
	if !m.CreatedAt().Equal(fixed) {
		t.Fatalf("CreatedAt=%v, quer=%v", m.CreatedAt(), fixed)
	}
}

func TestMainDoesNotPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("main panic: %v", r)
		}
	}()
	main()
}

func TestRestoreReplacesState(t *testing.T) {
	t.Parallel()
	e := NewEditor("abc")
	snap := e.Snapshot()
	e.Write("XYZ")
	e.Restore(snap)
	if e.Content() != "abc" || e.Cursor() != 3 {
		t.Fatalf("restore falhou: content=%q cursor=%d", e.Content(), e.Cursor())
	}
}
