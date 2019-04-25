// Package vql implements a reflective query interface to traverse Go values.
// Given a query q and a value v, vql.Eval(q, v) returns the object described
// by the query starting from v, or an error.
//
// TODO: This package needs much more documentation.
//
// TODO: Implement additional interesting query operators.
package vql

import (
	"fmt"
	"reflect"
)

// Eval evaluates q starting from v, and returns the object reached.
func Eval(q Query, v interface{}) (interface{}, error) {
	result, err := q.eval(newValue(v))
	if err != nil {
		return nil, err
	}
	return result.val, nil
}

// A value carries a value through a query, encapsulating the value and the
// parent value from which it was produced.
type value struct {
	val    interface{}
	parent *value
}

// newValue constructs a value for obj with no parent.
func newValue(obj interface{}) *value { return &value{val: obj} }

// pushValue constructs a new value for obj with v as its parent.
func pushValue(v *value, obj interface{}) *value {
	return &value{val: obj, parent: v}
}

// A Query evalutes a query starting at the specified value, returning the
// resultant value reached by the query.
type Query interface {
	eval(*value) (*value, error)
}

// Self is query whose value is its input.
var Self selfQuery

type selfQuery struct{}

func (selfQuery) eval(v *value) (*value, error) { return v, nil }

// Const returns a Query whose value is the fixed constant obj.
func Const(obj interface{}) Query { return constQuery{newValue(obj)} }

type constQuery struct{ *value }

func (c constQuery) eval(v *value) (*value, error) { return c.value, nil }

// Seq is a Query that sequentially composes other Queries.  An empty Seq
// yields its input unmodified; otherwise the result from the first Query is
// recursively traversed by those remaining in left to right order.
type Seq []Query

func (s Seq) eval(v *value) (*value, error) {
	for _, elt := range s {
		next, err := elt.eval(v)
		if err != nil {
			return v, err
		}
		v = next
	}
	return v, nil
}

// Key returns a Query that returns the value of the specified field on a
// struct, or entry in a map with string keys, or nil if no such field or key
// exists. It is an error if the value type is not a struct or string-key map.
func Key(s string) Query { return keyQuery(s) }

// Keys is a convenient shorthand for a Seq of the specified keys.
func Keys(keys ...string) Query {
	q := make(Seq, len(keys))
	for i, key := range keys {
		q[i] = keyQuery(key)
	}
	return q
}

type keyQuery string

var stringType = reflect.TypeOf("string")

func (k keyQuery) eval(v *value) (*value, error) {
	rv := reflect.Indirect(reflect.ValueOf(v.val))
	var f reflect.Value
	if rv.Kind() == reflect.Struct {
		f = rv.FieldByName(string(k))
	} else if rv.Kind() == reflect.Map && rv.Type().Key() == stringType {
		f = rv.MapIndex(reflect.ValueOf(string(k)))
	} else {
		return nil, fmt.Errorf("value of type %T is not a struct or string map", v.val)
	}
	if !f.IsValid() {
		return pushValue(v, nil), nil
	}
	return pushValue(v, f.Interface()), nil
}

// Each returns a Query that applies v to each element of an array or slice,
// and yields a slice (of type []interface{}) containing the resulting values.
func Each(v Query) Query { return mapQuery{v} }

type mapQuery struct{ Query }

func (m mapQuery) eval(v *value) (*value, error) {
	var vs []interface{}
	err := forEach(v.val, func(obj interface{}) error {
		next, err := m.Query.eval(pushValue(v, obj))
		if err == nil {
			vs = append(vs, next.val)
		}
		return err
	})
	return pushValue(v, vs), err
}

// Select returns a Query that applies f to result of evaluating q for each
// entry in an array or slice, and yields a slice of concrete type
// []interface{} containing the values for which f reports true.
func Select(q Query, f func(interface{}) bool) Query { return selectQuery{q, f} }

type selectQuery struct {
	Query
	keep func(interface{}) bool
}

func (s selectQuery) eval(v *value) (*value, error) {
	var vs []interface{}
	err := forEach(v.val, func(obj interface{}) error {
		v, err := s.Query.eval(pushValue(v, obj))
		if err != nil {
			return err
		} else if s.keep(v.val) {
			vs = append(vs, obj)
		}
		return nil
	})
	return pushValue(v, vs), err
}

func forEach(v interface{}, f func(interface{}) error) error {
	rv := reflect.ValueOf(v)
	if k := rv.Kind(); k != reflect.Array && k != reflect.Slice {
		return fmt.Errorf("value of type %T is not an array or slice", v)
	}
	for i := 0; i < rv.Len(); i++ {
		if err := f(rv.Index(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

// Bind returns a Query that binds the values from the specified subqueries to
// the corresponding keys in a string-to-value map.  The concrete type of the
// result is map[string]interface{}. It is not an error for requested values to
// be missing; their corresponding values will be nil.
func Bind(m map[string]Query) Query { return bindQuery(m) }

type bindQuery map[string]Query

func (b bindQuery) eval(v *value) (*value, error) {
	result := make(map[string]interface{})
	for key, q := range b {
		val, err := q.eval(v)
		if err != nil {
			return nil, fmt.Errorf("evaluating subquery %q: %v", key, err)
		}
		result[key] = val.val
	}
	return pushValue(v, result), nil
}

// As returns a Query whose value is the result of applying f to the value of q.
func As(q Query, f func(interface{}) interface{}) Query { return asQuery{q, f} }

type asQuery struct {
	Query
	f func(interface{}) interface{}
}

func (a asQuery) eval(v *value) (*value, error) {
	result, err := a.Query.eval(v)
	if err != nil {
		return nil, err
	}
	return pushValue(v, a.f(result.val)), nil
}

// TODO: Nicer error messages.
