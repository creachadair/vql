package vql_test

import (
	"fmt"
	"log"

	"bitbucket.org/creachadair/vql"
)

func ExampleEval() {
	type Person struct {
		Name string
		Exec bool
		Age  int
	}
	type Company struct {
		Name   string
		People []*Person
	}

	input := Company{
		Name: "Stuff, Inc.",
		People: []*Person{
			{Name: "Alice", Exec: true, Age: 35},
			{Name: "Bob", Exec: false, Age: 38},
			{Name: "Carol", Exec: false, Age: 19},
			{Name: "Dave", Exec: true, Age: 49},
			{Name: "Eve", Exec: false, Age: 27},
			{Name: "Frank", Exec: false, Age: 31},
		},
	}

	// A query to select the names of executives, identified by having their
	// Exec field set true.
	execNames := vql.Seq{
		vql.Key("People"),
		vql.Select(vql.Key("Exec"), func(obj interface{}) bool {
			return obj.(bool)
		}),
		vql.Each(vql.Key("Name")),
	}

	res, err := vql.Eval(execNames, input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%q\n", res)
	// Output:
	// ["Alice" "Dave"]
}
