package main

import (
	"strings"
	"testing"
)

func buildSample(t *testing.T) *Department {
	t.Helper()
	root := NewDepartment("Company")
	dev := NewDepartment("Dev")
	ops := NewDepartment("Ops")
	if err := dev.Add(
		&Employee{FullName: "A", Role: "Jr", Salary: 5000},
		&Employee{FullName: "B", Role: "Sr", Salary: 15000},
	); err != nil {
		t.Fatalf("add dev: %v", err)
	}
	if err := ops.Add(&Employee{FullName: "C", Role: "SRE", Salary: 12000}); err != nil {
		t.Fatalf("add ops: %v", err)
	}
	if err := root.Add(dev, ops); err != nil {
		t.Fatalf("add root: %v", err)
	}
	return root
}

func TestComposite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		node   func(*testing.T) OrgNode
		want   float64
		people int
	}{
		{
			name:   "single employee leaf",
			node:   func(*testing.T) OrgNode { return &Employee{FullName: "A", Role: "x", Salary: 1000} },
			want:   1000,
			people: 1,
		},
		{
			name:   "empty department is zero",
			node:   func(*testing.T) OrgNode { return NewDepartment("empty") },
			want:   0,
			people: 0,
		},
		{
			name:   "nested company total",
			node:   func(t *testing.T) OrgNode { return buildSample(t) },
			want:   32000,
			people: 3,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			n := tc.node(t)
			if got := n.TotalSalary(); got != tc.want {
				t.Errorf("total=%.2f want %.2f", got, tc.want)
			}
			if got := n.Headcount(); got != tc.people {
				t.Errorf("headcount=%d want %d", got, tc.people)
			}
		})
	}
}

func TestAverageSalary(t *testing.T) {
	t.Parallel()
	root := buildSample(t)
	got := AverageSalary(root)
	want := 32000.0 / 3.0
	if got != want {
		t.Errorf("avg=%.4f want %.4f", got, want)
	}

	if AverageSalary(nil) != 0 {
		t.Errorf("nil should be 0")
	}
	if AverageSalary(NewDepartment("empty")) != 0 {
		t.Errorf("empty dept should be 0")
	}
}

func TestDepartment_AddValidation(t *testing.T) {
	t.Parallel()
	d := NewDepartment("X")
	if err := d.Add(nil); err == nil {
		t.Errorf("expected nil rejection")
	}
	if err := d.Add(d); err == nil {
		t.Errorf("expected self-add rejection")
	}
}

func TestPrintContainsNames(t *testing.T) {
	t.Parallel()
	root := buildSample(t)
	out := root.Print(0)
	for _, want := range []string{"Company", "Dev", "Ops", "A", "B", "C"} {
		if !strings.Contains(out, want) {
			t.Errorf("print missing %q:\n%s", want, out)
		}
	}
}

func TestChildrenIsDefensiveCopy(t *testing.T) {
	t.Parallel()
	d := NewDepartment("X")
	_ = d.Add(&Employee{FullName: "E", Salary: 1})
	kids := d.Children()
	kids[0] = &Employee{FullName: "Mutated", Salary: 999}
	if d.Children()[0].Name() == "Mutated" {
		t.Errorf("Children() must return a defensive copy")
	}
}
