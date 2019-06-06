// Package vql implements a reflective query interface to traverse Go values.
// A vql.Query q describes a sequence of steps in the structure of a compound
// value (struct, slice, map). For a value v, vql.Eval(q, v) performs the steps
// described by q starting at v, and reports the value obtained.
//
// Purpose
//
// Decoding loosely-structured data such as JSON or YAML often produces dynamic
// structures that are highly nested and can be inconvenient to traverse. A
// vql.Query makes it easier to pick out only the pieces of the value relevant
// to your particular task.
//
// This can be helpful even the type of the structure is statically known, as
// in the case of configuration scripts with a complex nesting structure.  It
// can also be used to safely inspect variant types with similar shapes.
//
// Queries
//
// To fetch a named field from a struct, or the value from a map, use vql.Key.
// You can supply multiple keys to do compound lookups.
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
// To construct a list of subquery values, use vql.List, or vql.Cat to flatten
// list-valued subqueries.
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

// Key returns a Query that returns the value of the specified sequence of
// field lookups on a struct, or entry in a map. The result is nil if no such
// field or key exists. It is an error if the value type is not a struct or a
// map with a compatible key type.
func Key(keys ...interface{}) Query {
	q := make(Seq, len(keys))
	for i, key := range keys {
		q[i] = keyQuery{key: key}
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

// Each returns a Query that applies q to each element of an array, slice, or
// map, and yields a slice of type []interface{} containing the resulting
// values. If the input value is a map, the selector is given inputs of
// concrete type Entry.
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

// Entry is the concrete type of input values to a selector query for a map.
type Entry struct {
	Key, Value interface{}
}

// Select returns a Query that evaluates q for each entry in an array, slice,
// or map, and yields a slice of concrete type []interface{} containing the
// entries for which the value of q on that entry is true. It is an error if q
// does not yield a bool. If the input value is a map, tne selector is given
// inputs of concrete type Entry.
func Select(q ...Query) Query { return selectQuery{Seq(q)} }

type selectQuery struct {
	Query
}

func (s selectQuery) eval(v *value) (*value, error) {
	var vs []interface{}
	err := forEach(v.val, func(obj interface{}) error {
		v, err := s.Query.eval(newValue(obj))
		if err != nil {
			return err
		} else if keep, ok := v.val.(bool); !ok {
			return fmt.Errorf("select query yielded %T, not bool", v.val)
		} else if keep {
			vs = append(vs, obj) // N.B. keep the subquery input, not the result
		}
		return nil
	})
	return pushValue(v, vs), err
}

// Values represents the values bound by application of a Map query.
type Values map[string]interface{}

// A Map is a Query that binds the values from the specified subqueries to the
// corresponding keys in a string-to-value map.  The concrete type of the
// result is vql.Values, and the concrete type of each value is whatever was
// expressed by the corresponding subquery. It is not an error for requested
// values to be missing; their corresponding values will be nil.
type Map map[string]Query

func (m Map) eval(v *value) (*value, error) {
	result := make(Values)
	for key, q := range m {
		val, err := q.eval(v)
		if err != nil {
			return nil, fmt.Errorf("evaluating subquery %q: %v", key, err)
		}
		result[key] = val.val
	}
	return pushValue(v, result), nil
}

// Func returns a Query whose value is the result of applying a function v to
// its input. The value of v must have one of the following signatures:
//
//     func(T) U
//     func(T) (U, error)
//
// Otherwise, Func will panic. If v has the second form and reports an error,
// that error is propagated through the query chain.
func Func(v interface{}) Query {
	fn := reflect.ValueOf(v)
	t := fn.Type()
	switch {
	case t.Kind() != reflect.Func:
		panic("func: value is not a function")
	case t.NumIn() != 1:
		panic("func: wrong number of arguments")
	case t.NumOut() < 1, t.NumOut() > 2:
		panic("func: wrong number of returns")
	case t.NumOut() == 2 && t.Out(1) != errType:
		panic("func: last return value is not error")
	}
	return fnQuery{fn: fn, argType: t.In(0)}
}

var errType = reflect.TypeOf((*error)(nil)).Elem()

type fnQuery struct {
	fn      reflect.Value
	argType reflect.Type
}

func (a fnQuery) eval(v *value) (*value, error) {
	arg := reflect.ValueOf(v.val)
	if !arg.IsValid() {
		arg = reflect.New(a.argType).Elem()
	} else if !arg.Type().AssignableTo(a.argType) {
		return nil, fmt.Errorf("argument %T is not assignable to %v", v.val, a.argType)
	}
	res := a.fn.Call([]reflect.Value{arg})
	if len(res) == 2 {
		if err := res[1].Interface(); err != nil {
			return nil, err.(error)
		}
	}
	return pushValue(v, res[0].Interface()), nil
}

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

// Cat is a Query that accumulates the values of the given queries in a slice
// of type []interface{}. The contents of array or slice values are flattened.
// If no queries are given, or if all values are empty, the result is empty.
type Cat []Query

func (c Cat) eval(v *value) (*value, error) {
	var vs []interface{}
	for _, elt := range c {
		next, err := elt.eval(v)
		if err != nil {
			return nil, err
		}
		rv := reflect.ValueOf(next.val)
		if k := rv.Kind(); k == reflect.Slice || k == reflect.Array {
			for i := 0; i < rv.Len(); i++ {
				vs = append(vs, rv.Index(i).Interface())
			}
		} else {
			vs = append(vs, next.val)
		}
	}
	return pushValue(v, vs), nil
}

func forEach(v interface{}, f func(interface{}) error) error {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < rv.Len(); i++ {
			if err := f(rv.Index(i).Interface()); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range rv.MapKeys() {
			if err := f(Entry{
				Key:   key.Interface(),
				Value: rv.MapIndex(key).Interface(),
			}); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("value of type %T is not an array, map, or slice", v)
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

// IsNil is a Func that reports whether obj is nil, as a bool.
func IsNil(obj interface{}) bool { return obj == nil }

// NotNil is a Func that reports whether obj is non-nil, as a bool.
func NotNil(obj interface{}) bool { return obj != nil }
