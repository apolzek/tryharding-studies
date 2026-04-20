package main

import "fmt"

func main() {
	ed := NewEditor("")
	hist := NewHistory(ed, 20)

	hist.Save()
	ed.Write("hello")
	hist.Save()
	ed.Write(" world")
	hist.Save()
	ed.Write("!")

	fmt.Printf("estado atual : %q (cursor=%d)\n", ed.Content(), ed.Cursor())

	if err := hist.Undo(); err != nil {
		fmt.Println("undo:", err)
	}
	fmt.Printf("após undo    : %q\n", ed.Content())

	if err := hist.Undo(); err != nil {
		fmt.Println("undo:", err)
	}
	fmt.Printf("após 2x undo : %q\n", ed.Content())

	if err := hist.Redo(); err != nil {
		fmt.Println("redo:", err)
	}
	fmt.Printf("após redo    : %q\n", ed.Content())

	// nova edição descarta redos anteriores
	ed.Write(" there")
	hist.Save()
	fmt.Printf("nova edição  : %q\n", ed.Content())
	if err := hist.Redo(); err != nil {
		fmt.Println("redo esperado falhar:", err)
	}
}
