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

	// Executing a query on the input returns the matching results.
	all, err := vql.Eval(execNames, input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("All execs: %v\n", all)

	// Queries can be composed.
	res, err := vql.Eval(vql.Seq{execNames, vql.Index(0)}, input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("First exec: %s\n", res.(string))

	// Output:
	// All execs: [Alice Dave]
	// First exec: Alice
}

func ExampleKey() {
	res, err := vql.Eval(vql.Key("three"), map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	// Output:
	// 3
}

func ExampleEach() {
	res, err := vql.Eval(vql.Each(vql.Key(1)), []map[int]string{
		{0: "zero", 1: "one", 2: "two"},
		{0: "cero", 1: "uno", 2: "dos"},
		{0: "null", 1: "eins", 2: "zwei"},
		{0: "ゼロ", 1: "一", 2: "二"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	// Output:
	// [one uno eins 一]
}

func ExampleSelect() {
	res, err := vql.Eval(vql.Select(vql.Key("age"), func(obj interface{}) bool {
		return obj.(int) > 35
	}), []map[string]int{
		{"age": 19, "id": 10332, "height": 180},
		{"age": 39, "id": 10335, "height": 143},
		{"age": 34, "id": 92131, "height": 139},
		{"age": 65, "id": 7153, "height": 182},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, elt := range res.([]interface{}) {
		m := elt.(map[string]int)
		fmt.Printf("Age %d years, ID %d, height %d cm\n", m["age"], m["id"], m["height"])
	}
	// Output:
	// Age 39 years, ID 10335, height 143 cm
	// Age 65 years, ID 7153, height 182 cm
}
