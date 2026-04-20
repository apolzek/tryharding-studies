package main

import (
	"errors"
	"strings"
	"time"
)

// Memento captura o estado interno do Editor. O conteúdo é opaco para o Caretaker.
type Memento struct {
	content   string
	cursor    int
	createdAt time.Time
}

// CreatedAt expõe metadados úteis para listagens no histórico.
func (m Memento) CreatedAt() time.Time { return m.createdAt }

// Editor é o Originator: conhece os detalhes internos e produz/restaura Mementos.
type Editor struct {
	content string
	cursor  int
	clock   func() time.Time
}

// NewEditor cria um editor com conteúdo inicial.
func NewEditor(initial string) *Editor {
	return &Editor{content: initial, cursor: len(initial), clock: time.Now}
}

// Content devolve o texto atual.
func (e *Editor) Content() string { return e.content }

// Cursor devolve a posição atual do cursor.
func (e *Editor) Cursor() int { return e.cursor }

// Write insere texto na posição do cursor, avançando-o.
func (e *Editor) Write(text string) {
	var sb strings.Builder
	sb.WriteString(e.content[:e.cursor])
	sb.WriteString(text)
	sb.WriteString(e.content[e.cursor:])
	e.content = sb.String()
	e.cursor += len(text)
}

// MoveCursor posiciona o cursor com clamping nos limites.
func (e *Editor) MoveCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(e.content) {
		pos = len(e.content)
	}
	e.cursor = pos
}

// Delete remove n runas (bytes) antes do cursor.
func (e *Editor) Delete(n int) {
	if n <= 0 || e.cursor == 0 {
		return
	}
	if n > e.cursor {
		n = e.cursor
	}
	e.content = e.content[:e.cursor-n] + e.content[e.cursor:]
	e.cursor -= n
}

// Snapshot produz um memento do estado atual.
func (e *Editor) Snapshot() Memento {
	return Memento{content: e.content, cursor: e.cursor, createdAt: e.clock()}
}

// Restore volta o editor para o estado de um memento.
func (e *Editor) Restore(m Memento) {
	e.content = m.content
	e.cursor = m.cursor
}

// ErrHistoryEmpty indica que não há snapshot disponível na direção pedida.
var ErrHistoryEmpty = errors.New("histórico vazio")

// History é o Caretaker: guarda os mementos em pilhas de undo/redo.
type History struct {
	editor *Editor
	undos  []Memento
	redos  []Memento
	limit  int
}

// NewHistory cria um caretaker com limite opcional (0 = ilimitado).
func NewHistory(e *Editor, limit int) *History {
	return &History{editor: e, limit: limit}
}

// Save captura o estado atual e zera a pilha de redo.
func (h *History) Save() {
	snap := h.editor.Snapshot()
	h.undos = append(h.undos, snap)
	if h.limit > 0 && len(h.undos) > h.limit {
		// descarta o mais antigo
		h.undos = h.undos[1:]
	}
	h.redos = nil
}

// Undo volta ao memento anterior, empilhando o estado atual em redo.
func (h *History) Undo() error {
	if len(h.undos) == 0 {
		return ErrHistoryEmpty
	}
	last := len(h.undos) - 1
	current := h.editor.Snapshot()
	h.editor.Restore(h.undos[last])
	h.undos = h.undos[:last]
	h.redos = append(h.redos, current)
	return nil
}

// Redo reaplica o próximo estado previamente desfeito.
func (h *History) Redo() error {
	if len(h.redos) == 0 {
		return ErrHistoryEmpty
	}
	last := len(h.redos) - 1
	current := h.editor.Snapshot()
	h.editor.Restore(h.redos[last])
	h.redos = h.redos[:last]
	h.undos = append(h.undos, current)
	return nil
}

// Len devolve o tamanho das pilhas (undo, redo).
func (h *History) Len() (int, int) { return len(h.undos), len(h.redos) }
