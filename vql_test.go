package vql_test

import (
	"strings"
	"testing"

	"bitbucket.org/creachadair/vql"
	"github.com/kylelemons/godebug/pretty"
)

func TestQueries(t *testing.T) {
	type thingy struct {
		A string
		B int
		S []string
		T *thingy
	}
	t2 := &thingy{ // N.B. pointer
		A: "bar",
		B: 25,
		S: []string{"apple", "pie"},
		T: nil,
	}
	t1 := thingy{ // N.B. non-pointer
		A: "foo",
		B: 17,
		S: []string{"pear", "plum", "cherry"},
		T: t2,
	}
	m := map[string]string{
		"oh":   "bother",
		"said": "pooh",
	}

	tests := []struct {
		query       vql.Query
		input, want interface{}
	}{
		{vql.Self, "whatever", "whatever"},
		{vql.Self, nil, nil},

		{vql.Const(true), nil, true},
		{vql.Const(true), "whatever", true},
		{vql.Const(125), []string{"a", "b", "c"}, 125},

		{vql.Key("A"), t1, "foo"},
		{vql.Key("B"), t1, 17},
		{vql.Key("S"), t1, []string{"pear", "plum", "cherry"}},
		{vql.Key("C"), t1, nil},
		{vql.Key("oh"), m, "bother"},
		{vql.Key("piglet"), m, nil},

		{vql.Seq(nil), "whatever", "whatever"},
		{vql.Seq{}, "whatever", "whatever"},
		{vql.Seq{vql.Const(1)}, "whatever", 1},
		{vql.Seq{vql.Key("T"), vql.Key("A")}, t1, "bar"},
		{vql.Seq{vql.Key("T"), vql.Key("B")}, t1, 25},
		{vql.Seq{vql.Key("T"), vql.Key("C")}, t1, nil},
		{vql.Seq{vql.Key("T"), vql.Key("T")}, t1, (*thingy)(nil)},
		{vql.Keys("T", "A"), t1, "bar"},
		{vql.Keys("T", "B"), t1, 25},
		{vql.Keys("T", "C"), t1, nil},
		{vql.Keys("T", "T"), t1, (*thingy)(nil)},

		{vql.Each(vql.Key("A")), []*thingy{&t1, t2}, []interface{}{"foo", "bar"}},

		{vql.Seq{vql.Keys("T", "S"), vql.Index(-1)}, t1, "pie"},
		{vql.Seq{vql.Key("S"), vql.Index(1)}, t1, "plum"},

		{vql.Seq{
			vql.Key("S"),
			vql.Select(vql.Self, func(obj interface{}) bool {
				s, ok := obj.(string)
				return ok && strings.HasPrefix(s, "p")
			}),
		}, t1, []interface{}{"pear", "plum"}},

		{vql.Bind(map[string]vql.Query{
			"first":  vql.Key("B"),
			"second": vql.Seq{vql.Key("T"), vql.Key("B")},
		}), t1, map[string]interface{}{"first": 17, "second": 25}},

		{vql.Each(vql.With(vql.Key("B"), func(obj interface{}) interface{} {
			return obj.(int) > 20
		})), []*thingy{&t1, t2}, []bool{false, true}},

		{vql.Or{
			vql.Index(10),     // error, ignored
			vql.Const(nil),    // nil value, ignored
			vql.Index(1),      // non-nil value, selected
			vql.Const("whee"), // unevaluated
		}, []string{"all", "bears", "chug", "diesel"}, "bears"},

		{vql.List(nil), t1, []interface{}(nil)},
		{vql.List{}, t1, []interface{}(nil)},
		{vql.List{
			vql.Keys("T", "A"),
			vql.Key("B"),
			vql.Seq{vql.Key("S"), vql.Index(1)},
		}, t1, []interface{}{"bar", 17, "plum"}},
	}
	for _, test := range tests {
		got, err := vql.Eval(test.query, test.input)
		if err != nil {
			t.Errorf("Eval(%v): unexpected error: %v", test.query, err)
		} else if diff := pretty.Compare(got, test.want); diff != "" {
			t.Errorf("Eval(%v): (-got, +want)\n%s", test.query, diff)
		}
	}
}

// TODO: Add tests for error conditions.
