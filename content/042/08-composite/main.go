package main

import "fmt"

func main() {
	engineering := NewDepartment("Engenharia")
	backend := NewDepartment("Backend")
	frontend := NewDepartment("Frontend")

	_ = backend.Add(
		&Employee{FullName: "Ana", Role: "Staff", Salary: 22000},
		&Employee{FullName: "Bruno", Role: "Senior", Salary: 15000},
	)
	_ = frontend.Add(
		&Employee{FullName: "Carla", Role: "Pleno", Salary: 9000},
	)
	_ = engineering.Add(backend, frontend, &Employee{FullName: "Diego", Role: "VP Eng", Salary: 40000})

	company := NewDepartment("ACME")
	_ = company.Add(engineering)

	fmt.Println(company.Print(0))
	fmt.Printf("\nTotal payroll: R$%.2f\n", company.TotalSalary())
	fmt.Printf("Headcount: %d\n", company.Headcount())
	fmt.Printf("Média salarial: R$%.2f\n", AverageSalary(company))
}
