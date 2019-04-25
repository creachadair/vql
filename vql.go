// Package vql implements a reflective query interface to traverse Go values.
// A vql.Query q describes a sequence of steps in the structure of a compound
// value (struct, slice, map). For a value v, vql.Eval(q, v) performs the steps
// described by q starting at v, and reports the value obtained.
//
// Queries
//
// To fetch a named field from a struct, or the value from a map, use vql.Key.
// Compound lookups can be chained with vql.Keys.
//
// To index into a slice of values, use vql.Index.
//
// To walk sequentially into the structure of a value, use vql.Seq.
//
// To apply a subquery to the elements of a slice, use vql.Each.
//
// To filter the elements of a slice based on a subquery, use vql.Select.
//
// To extract subqueries from a value, use vql.Bind.
//
// To apply a functional transformation to a value, use vql.As.
//
// To construct a list of subquery values, use vql.List.
//
// To select one of a sequence of subqueries to apply, use vql.Or.
//
// TODO: Add more descriptive errors.
package vql

import (
	"fmt"
	"reflect"
)

// Eval evaluates q starting from v, and returns the object described.
func Eval(q Query, v interface{}) (interface{}, error) {
	result, err := q.eval(newValue(v))
	if err != nil {
		return nil, err
	}
	return result.val, nil
}

// A value carries a value through a query, encapsulating the current state of
// query expansion (val) and the parent value from which it was produced.  The
// initial input to a query has parent == nil.
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
// struct, or entry in a map, or nil if no such field or key exists. It is an
// error if the value type is not a struct or a map with a compatible key type.
func Key(key interface{}) Query { return keyQuery{key: key} }

// Keys is a convenience shorthand for a Seq of the specified keys.
func Keys(keys ...interface{}) Query {
	q := make(Seq, len(keys))
	for i, key := range keys {
		q[i] = Key(key)
	}
	return q
}

type keyQuery struct {
	key interface{}
}

func (k keyQuery) eval(v *value) (*value, error) {
	rv := reflect.Indirect(reflect.ValueOf(v.val))
	var f reflect.Value
	if rv.Kind() == reflect.Struct {
		if s, ok := k.key.(string); ok {
			f = rv.FieldByName(s)
		} else {
			return nil, fmt.Errorf("value of type %T cannot be a field name", k.key)
		}
	} else if rv.Kind() == reflect.Map {
		if !reflect.TypeOf(k.key).AssignableTo(rv.Type().Key()) {
			return nil, fmt.Errorf("value of type %T cannot be a key in this map", k.key)
		}
		f = rv.MapIndex(reflect.ValueOf(k.key))
	} else {
		return nil, fmt.Errorf("value of type %T is not a struct or map", v.val)
	}
	if !f.IsValid() {
		return pushValue(v, nil), nil
	}
	return pushValue(v, f.Interface()), nil
}

// Each returns a Query that applies q to each element of an array or slice,
// and yields a slice of type []interface{} containing the resulting values.
func Each(q Query) Query { return mapQuery{q} }

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

// As returns a Query whose value is the result of applying f to its input.
func As(f Transform) Query { return asQuery(f) }

type asQuery Transform

func (a asQuery) eval(v *value) (*value, error) { return pushValue(v, a(v.val)), nil }

// Index returns a Query that selects the item at a specified offset in an
// array or slice. Offsets are 0-based, with negative offsets referring to
// offsets from the end of the sequence. An offset outside the range of the
// sequence report an error.
func Index(i int) Query { return indexQuery(i) }

type indexQuery int

func (q indexQuery) eval(v *value) (*value, error) {
	rv, err := seqValue(v.val)
	if err != nil {
		return nil, err
	}
	offset := int(q)
	if offset < 0 {
		offset += rv.Len()
	}
	if offset >= rv.Len() || offset < 0 {
		return nil, fmt.Errorf("index %d is out of range for 0..%d", offset, rv.Len())
	}
	return pushValue(v, rv.Index(offset).Interface()), nil
}

// Or is a Query that yields the first non-nil value among the given queries in
// left-to-right order. If no queries are given, the result is nil.  Errors in
// evaluating subqueries are ignored.
type Or []Query

func (o Or) eval(v *value) (*value, error) {
	for _, q := range o {
		next, err := q.eval(v)
		if err == nil && next.val != nil {
			return pushValue(v, next.val), nil
		}
	}
	return pushValue(v, nil), nil
}

// List is a Query that accumulates the values of the given queries in a slice
// of type []interface{}. If no queries are given, the slice is empty.
type List []Query

func (q List) eval(v *value) (*value, error) {
	var vs []interface{}
	for _, elt := range q {
		next, err := elt.eval(v)
		if err != nil {
			return nil, err
		}
		vs = append(vs, next.val)
	}
	return pushValue(v, vs), nil
}

func forEach(v interface{}, f func(interface{}) error) error {
	rv, err := seqValue(v)
	if err != nil {
		return err
	}
	for i := 0; i < rv.Len(); i++ {
		if err := f(rv.Index(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

func seqValue(v interface{}) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	if k := rv.Kind(); k != reflect.Array && k != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("value of type %T is not an array or slice", v)
	}
	return rv, nil
}
