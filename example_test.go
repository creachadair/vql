package vql_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/creachadair/vql"
)

func ExampleEval_variant() {
	// For this example we consider types that have a similar structure, both
	// have a Name field and a Boolean property but are not identical.
	type Animal struct {
		Name  string
		Alive bool
	}
	type Person struct {
		Name     string
		Age      int
		Employed bool
	}

	inputs := []interface{}{
		Animal{Name: "aardvark", Alive: true},
		Person{Name: "alice", Age: 25, Employed: false},
		Person{Name: "bob", Age: 38, Employed: true},
		Animal{Name: "boar", Alive: false},
	}

	// Select values whose bool-containing field is true, and pull out the
	// corresponding name.
	query := vql.Seq{
		vql.Select(vql.Or{vql.Key("Alive"), vql.Key("Employed")}),
		vql.Each(vql.Key("Name")),
	}

	res, err := vql.Eval(query, inputs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	// Output:
	// [aardvark bob]
}

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

	isExec := func(s string) bool {
		return len(s) == 3 && s[0] == 'C' && s[2] == 'O'
	}

	// A query to select the names of executives, identified by their title
	// being "CxO".
	execNames := vql.Seq{
		vql.Key("People"),
		vql.Select(vql.Key("Title"), vql.Func(isExec)),
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
	isPresidential := func(age int) bool { return age > 35 }

	res, err := vql.Eval(vql.Select(
		vql.Key("age"),
		vql.Func(isPresidential),
	), []map[string]int{
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

func ExampleMap() {
	type Client struct {
		Name string
		Port int
		IP   string
	}
	res, err := vql.Eval(vql.Map{
		"address": vql.Key("IP"),
		"port":    vql.Key("Port"),
	}, Client{Name: "X", Port: 75, IP: "129.170.16.50"})
	if err != nil {
		log.Fatal(err)
	}
	m := res.(vql.Values)
	fmt.Printf("Address %s, port %d\n", m["address"], m["port"])
	// Output:
	// Address 129.170.16.50, port 75
}

func ExampleFunc() {
	cleanString := func(s string) string {
		return strings.Join(strings.Fields(s), " ")
	}
	res, err := vql.Eval(vql.Func(cleanString), " a messy\n \t string\n\n")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	// Output:
	// a messy string
}

func ExampleIndex() {
	res, err := vql.Eval(vql.Index(2), []int{2, 3, 5, 7, 11, 13})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	// Output:
	// 5
}

func ExampleOr() {
	res, err := vql.Eval(vql.Or{
		vql.Key("cheese"), // wrong type, not selected
		vql.Index(1),      // match
		vql.Index(-1),     // match but not evaluted
	}, []string{"some", "settling", "may", "occur"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	// Output:
	// settling
}

func ExampleList() {
	res, err := vql.Eval(vql.List{
		vql.Key("mice"),
		vql.Key("men"),
	}, map[string]bool{"mice": true, "men": false, "cows": true})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	// Output:
	// [true false]
}

func ExampleCat() {
	res, err := vql.Eval(vql.Cat{
		vql.Key("xyz"),
		vql.Key("pdq"),
	}, map[string]interface{}{
		"xyz": "some",
		"pdq": []string{"assembly", "required"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	// Output:
	// [some assembly required]
}
