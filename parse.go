package vql

import (
	"bufio"
	"bytes"
)

/*
Query grammar:

query  = alt
alt    = seq | seq "//" alt
alts   = alt | alt "," alts   -- 1 or more
seq    = term | term "." seq
term   = base | list | cat | map | each | func | select
list   = "[" alts? "]"
cat    = "#[" alts? "]"
map    = "{" kvals? "}"
kvals  = kval "," kvals?
kval   = key ":" alt
each   = "each" term
select = "select" term
func   = "@" name
base   = atom | atom "[" int "]" | atom op atom
atom   = const | name | quoted | hole | "(" alt ")"
const  = string | int | float | bool
quoted = "'" name
key    = string | name
op     = "==" | "<" | "<=" | ">" | ">="
string = "\"" schars "\""
hole   = "$" name

type selfQuery struct{}
type Seq []Query
type Map map[string]Query
type Or []Query
type List []Query
type Cat []Query
func Const(obj interface{}) Query { return constQuery{newValue(obj)} }
func Key(keys ...interface{}) Query {
func Each(q Query) Query { return mapQuery{q} }
func Select(q ...Query) Query { return selectQuery{Seq(q)} }
func Func(v interface{}) Query {
func Index(i int) Query { return indexQuery(i) }
func Eq(needle interface{}) Query {
func Lt(needle interface{}) Query {
func Le(needle interface{}) Query {
func Gt(needle interface{}) Query {
func Ge(needle interface{}) Query {
func IsNil(obj interface{}) bool { return obj == nil }
func NotNil(obj interface{}) bool { return obj != nil }
*/

const (
	tInvalid = iota
	tName
	tQName // quoted, func, or hole
	tInt
	tFloat
	tTrue
	tFalse
	tLeftSQ
	tRightSQ
	tLeftHashSQ
	tLeftCurly
	tRightCurly
	tDot
	tColon
)

type parser struct {
	buf *bufio.Reader
	tok bytes.Buffer
}
