package vql

// This file defines some useful transformation functions for use with vql.As.

// A Transform maps one object to another, for use with vql.As.
type Transform = func(interface{}) interface{}

// IsNil is a Transform that reports whether obj is nil, as a bool.
func IsNil(obj interface{}) interface{} { return obj == nil }

// NotNil is a Transform that reports whether obj is non-nil, as a bool.
func NotNil(obj interface{}) interface{} { return obj != nil }
