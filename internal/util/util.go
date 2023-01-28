package util

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Stack is a stack. It is backed by a slice. The zero-value for the stack is
// ready to use.
type Stack[E any] struct {
	Of []E
}

// Push pushes value onto the stack.
func (s *Stack[E]) Push(v E) {
	if s == nil {
		panic("push to nil stack")
	}
	s.Of = append(s.Of, v)
}

// Pop pops value off of the stack. If the stack is empty, panics.
func (s *Stack[E]) Pop() E {
	if s == nil {
		panic("pop of nil stack")
	}
	if len(s.Of) == 0 {
		panic("pop of empty stack")
	}

	v := s.Of[len(s.Of)-1]
	newVals := make([]E, len(s.Of)-1)
	copy(newVals, s.Of)
	s.Of = newVals
	return v
}

// Peek checks the value at the top of the stack. If the stack is empty, panics.
func (s Stack[E]) Peek() E {
	if len(s.Of) == 0 {
		panic("peek of empty stack")
	}
	return s.Of[len(s.Of)-1]
}

// PeekAt checks the value at the given position of the stack. If the stack is
// not that long, panics.
func (s Stack[E]) PeekAt(p int) E {
	if p >= len(s.Of) {
		panic(fmt.Sprintf("stack index out of range: %d", p))
	}
	return s.Of[p]
}

// Len returns the length of the stack.
func (s Stack[E]) Len() int {
	return len(s.Of)
}

// Empty returns whether the stack is empty.
func (s Stack[E]) Empty() bool {
	return len(s.Of) == 0
}

// Matrix2 is a 2d mapping of coordinates to values. Do not use a Matrix2 by
// itself, use NewMatrix2.
type Matrix2[EX, EY comparable, V any] map[EX]map[EY]V

func NewMatrix2[EX, EY comparable, V any]() Matrix2[EX, EY, V] {
	return map[EX]map[EY]V{}
}

func (m2 Matrix2[EX, EY, V]) Set(x EX, y EY, value V) {
	if m2 == nil {
		panic("assignment to nil Matrix2")
	}

	col, colExists := m2[x]
	if !colExists {
		col = map[EY]V{}
		m2[x] = col
	}
	col[y] = value
	m2[x] = col
}

// Get gets a pointer to the value pointed to by the coordinates. If not at
// those coordinates, nil is returned. Updating the value of the pointer will
// not update the matrix; use Set for that.
func (m2 Matrix2[EX, EY, V]) Get(x EX, y EY) *V {
	if m2 == nil {
		return nil
	}

	col, colExists := m2[x]
	if !colExists {
		return nil
	}

	val, valExists := col[y]
	if !valExists {
		return nil
	}

	return &val
}

type Set[E comparable] map[E]bool

func (s Set[E]) DisjointWith(o Set[E]) bool {
	for k := range s {
		if _, ok := o[k]; ok {
			return false
		}
	}
	return true
}

func (s Set[E]) Has(value E) bool {
	_, has := s[value]
	return has
}

func (s Set[E]) Add(value E) {
	s[value] = true
}

func (s Set[E]) Remove(value E) {
	delete(s, value)
}

// MakeTextList gives a nice list of things based on their display name.
//
// TODO: turn this into a generic function that accepts displayable OR ~string
func MakeTextList(items []string, articles bool) string {
	if len(items) < 1 {
		return ""
	}

	output := ""

	withArts := make([]string, len(items))
	for i := range items {
		art := ""
		item := items[i]
		if articles {
			art = ArticleFor(item, false)

			iRunes := []rune(item)
			leadingUpper := unicode.IsUpper(iRunes[0])
			allCaps := leadingUpper
			if leadingUpper && len(iRunes) > 1 {
				allCaps = unicode.IsUpper(iRunes[1])
			}

			if leadingUpper && !allCaps {
				// make the item lower case
				iRunes[0] = unicode.ToLower(iRunes[0])
				item = string(iRunes)
			}

			art += " "
		}
		withArts[i] = art + " " + item
	}

	if len(withArts) == 1 {
		output += withArts[0]
	} else if len(withArts) == 2 {
		output += withArts[0] + " and " + withArts[1]
	} else {
		// if its more than two, use an oxford comma
		withArts[len(withArts)-1] = "and " + withArts[len(withArts)-1]
		output += strings.Join(withArts, ", ")
	}

	return output
}

// ArticleFor returns the article for the given string. It will be capitalized
// the same as the string. If definite is true, the returned value will be "the"
// capitalized as described; otherwise, it will be "a"/"an" capitalized as
// described.
func ArticleFor(s string, definite bool) string {
	sRunes := []rune(s)

	if len(sRunes) < 1 {
		return ""
	}

	leadingUpper := unicode.IsUpper(sRunes[0])
	allCaps := leadingUpper
	if leadingUpper && len(sRunes) > 1 {
		allCaps = unicode.IsUpper(sRunes[1])
	}

	art := ""
	if definite {
		if allCaps {
			art = "THE"
		} else if leadingUpper {
			art = "The"
		} else {
			art = "the"
		}
	} else {
		if allCaps || leadingUpper {
			art = "A"
		} else {
			art = "a"
		}

		sUpperRunes := []rune(strings.ToUpper(s))
		first := sUpperRunes[0]
		if first == 'A' || first == 'E' || first == 'I' || first == 'O' || first == 'U' {
			if allCaps {
				art += "N"
			} else {
				art += "n"
			}
		}
	}

	return art
}

// InSlice returns whether s is present in the given slice by checking each item
// in order.
func InSlice[V comparable](s V, slice []V) bool {
	for i := range slice {
		if slice[i] == s {
			return true
		}
	}
	return false
}

// OrderedKeys returns the keys of m, ordered a particular way. The order is
// guaranteed to be the same on every run.
//
// As of this writing, the order is alphabetical, but this function does not
// guarantee this will always be the case.
func OrderedKeys[V any](m map[string]V) []string {
	var keys []string
	var idx int

	keys = make([]string, len(m))
	idx = 0

	for k := range m {
		keys[idx] = k
		idx++
	}

	sort.Strings(keys)

	return keys
}

// EqualNilness returns whether the two values are either both nil or both
// non-nil.
func EqualNilness(o1 any, o2 any) bool {
	if o1 == nil {
		return o2 == nil
	} else {
		return o2 != nil
	}
}

// CustomComparable is an interface for items that may be checked against
// arbitrary other objects. In practice most will attempt to typecast to their
// own type and immediately return false if the argument is not the same, but in
// theory this allows for comparison to multiple types of things.
type CustomComparable interface {
	Equal(other any) bool
}

// EqualSlices checks that the two slices contain the same items in the same
// order. Equality of items is checked by items in the slices are equal by
// calling the custom Equal function on each element. In particular, Equal is
// called on elements of sl1 with elements of sl2 passed in as the argument.
func EqualSlices[T CustomComparable](sl1 []T, sl2 []T) bool {
	if len(sl1) != len(sl2) {
		return false
	}

	for i := range sl1 {
		if !sl1[i].Equal(sl2[i]) {
			return false
		}
	}

	return true
}

// LongestCommonPrefix gets the longest prefix that the two slices have in
// common.
func LongestCommonPrefix[T comparable](sl1 []T, sl2 []T) []T {
	var pref []T

	minLen := len(sl1)
	if minLen > len(sl2) {
		minLen = len(sl2)
	}

	for i := 0; i < minLen; i++ {
		if sl1[i] != sl2[i] {
			break
		}
		pref = append(pref, sl1[i])
	}

	return pref
}

// HasPrefix returns whether the given slice has the given prefix. If prefix
// is empty or nil this will always be true regardless of sl's value.
func HasPrefix[T comparable](sl []T, prefix []T) bool {
	if len(prefix) > len(sl) {
		return false
	}

	if len(prefix) == 0 {
		return true
	}

	for i := range prefix {
		if sl[i] != prefix[i] {
			return false
		}
	}

	return true
}
