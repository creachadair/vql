package vql_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/creachadair/vql"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
	sm := map[string]string{
		"oh":   "bother",
		"said": "pooh",
	}
	zm := map[int]string{
		10: "ten",
		12: "twelve",
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
		{vql.Seq{vql.Key("C"), vql.Func(vql.IsNil)}, t1, true},
		{vql.Seq{vql.Key("C"), vql.Func(vql.NotNil)}, t1, false},

		{vql.Key("oh"), sm, "bother"},
		{vql.Key("piglet"), sm, nil},
		{vql.Key(10), zm, "ten"},
		{vql.Key(11), zm, nil},

		{vql.Seq(nil), "whatever", "whatever"},
		{vql.Seq{}, "whatever", "whatever"},
		{vql.Seq{vql.Const(1)}, "whatever", 1},
		{vql.Seq{vql.Key("T"), vql.Key("A")}, t1, "bar"},
		{vql.Seq{vql.Key("T"), vql.Key("B")}, t1, 25},
		{vql.Seq{vql.Key("T"), vql.Key("C")}, t1, nil},
		{vql.Seq{vql.Key("T"), vql.Key("T")}, t1, (*thingy)(nil)},
		{vql.Key("T", "A"), t1, "bar"},
		{vql.Key("T", "B"), t1, 25},
		{vql.Key("T", "C"), t1, nil},
		{vql.Key("T", "T"), t1, (*thingy)(nil)},

		{vql.Each(vql.Key("A")), []*thingy{&t1, t2}, []interface{}{"foo", "bar"}},
		{vql.Each(vql.Key("Key")), map[string]bool{"ok": true}, []interface{}{"ok"}},
		{vql.Each(vql.Key("Value")), map[string]bool{"ok": true}, []interface{}{true}},
		{vql.Seq{
			vql.Select(vql.Key("Value"), vql.Eq(4)),
			vql.Each(vql.Key("Key")),
		}, map[string]int{"yes": 4, "sí": 3, "да": 2, "はい": 1}, []interface{}{"yes"}},

		// Order comparisons.
		{vql.Lt(25), 16, true},
		{vql.Gt(25), 16, false},
		{vql.Le(25), 16, true},
		{vql.Ge(25), 16, false},
		{vql.Lt(25), 25, false},
		{vql.Gt(25), 25, false},
		{vql.Le(25), 25, true},
		{vql.Ge(25), 25, true},
		{vql.Lt(25), 35, false},
		{vql.Gt(25), 35, true},
		{vql.Le(25), 35, false},
		{vql.Ge(25), 35, true},

		{vql.Seq{vql.Key("T", "S"), vql.Index(-1)}, t1, "pie"},
		{vql.Seq{vql.Key("S"), vql.Index(1)}, t1, "plum"},

		{vql.Seq{
			vql.Key("S"),
			vql.Select(vql.Func(func(s string) bool {
				return strings.HasPrefix(s, "p")
			})),
		}, t1, []interface{}{"pear", "plum"}},

		{vql.Map{
			"first":  vql.Key("B"),
			"second": vql.Seq{vql.Key("T"), vql.Key("B")},
		}, t1, vql.Values{"first": 17, "second": 25}},

		{vql.Each(vql.Seq{vql.Key("B"), vql.Func(func(v int) bool {
			return v > 20
		})}), []*thingy{&t1, t2}, []interface{}{false, true}},

		{vql.Or{
			vql.Index(10),     // error, ignored
			vql.Const(nil),    // nil value, ignored
			vql.Index(1),      // non-nil value, selected
			vql.Const("whee"), // unevaluated
		}, []string{"all", "bears", "chug", "diesel"}, "bears"},

		{vql.Seq{
			vql.Key("S"),
			vql.Or{
				// Verify that a two-result function with an error gets propagated
				// when it fails.
				vql.Func(func(ss []string) (string, error) {
					return "", errors.New("I say no thank you sir")
				}),
				vql.Index(2),
			},
		}, t1, "cherry"},

		{vql.List(nil), t1, []interface{}(nil)},
		{vql.List{}, t1, []interface{}(nil)},
		{vql.List{
			vql.Key("T", "A"),
			vql.Key("B"),
			vql.Seq{vql.Key("S"), vql.Index(1)},
		}, t1, []interface{}{"bar", 17, "plum"}},
		{vql.List{
			vql.Key("T", "A"),
			vql.Key("T", "S"),
			vql.Key("B"),
		}, t1, []interface{}{"bar", []string{"apple", "pie"}, 17}},

		{vql.Cat(nil), "whatever", []interface{}{}},
		{vql.Cat{}, "whatever", []interface{}{}},
		{vql.Cat{vql.Const("x")}, "whatever", []interface{}{"x"}},
		{vql.Cat{vql.Self}, "x", []interface{}{"x"}},
		{vql.Cat{vql.Self}, []interface{}{"x"}, []interface{}{"x"}},
		{vql.Cat{vql.Self}, []string{"a", "b"}, []interface{}{"a", "b"}},
		{vql.Cat{
			vql.Key("A"),
			vql.Key("T", "B"),
			vql.Key("S"),
			vql.Key("T", "S"),
		}, t1, []interface{}{"foo", 25, "pear", "plum", "cherry", "apple", "pie"}},
	}
	for _, test := range tests {
		got, err := vql.Eval(test.query, test.input)
		if err != nil {
			t.Errorf("Eval(%v): unexpected error: %v", test.query, err)
		} else if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Eval(%v): (-want, +got)\n%s", test.query, diff)
		}
	}
}

// TODO: Add tests for error conditions.
