package vql_test

import (
	"fmt"
	"log"

	"bitbucket.org/creachadair/vql"
)

func ExampleEval() {
	type Person struct {
		Name  string
		Title string
		Age   int
	}
	type Company struct {
		Name   string
		People []*Person
	}

	input := Company{
		Name: "Stuff, Inc.",
		People: []*Person{
			{Name: "Alice", Title: "CEO", Age: 35},
			{Name: "Bob", Title: "MGR", Age: 38},
			{Name: "Carol", Title: "MGR", Age: 19},
			{Name: "Dave", Title: "CFO", Age: 49},
			{Name: "Eve", Title: "EMP", Age: 27},
			{Name: "Frank", Title: "EMP", Age: 31},
		},
	}

	// A query to select the names of executives, identified by their title
	// being "CEO" or "CFO".
	execNames := vql.Seq{
		vql.Key("People"),
		vql.Select(vql.Key("Title"), func(obj interface{}) bool {
			s := obj.(string)
			return s == "CEO" || s == "CFO"
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
