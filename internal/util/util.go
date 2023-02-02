package util

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

// Stringer

// Stack is a stack. It is backed by a slice where the left-most position is the
// top of the stack. The zero-value for the stack is ready to use.
type Stack[E any] struct {
	Of []E
}

// Push pushes value onto the stack.
func (s *Stack[E]) Push(v E) {
	if s == nil {
		panic("push to nil stack")
	}
	s.Of = append([]E{v}, s.Of...)
}

// Pop pops value off of the stack. If the stack is empty, panics.
func (s *Stack[E]) Pop() E {
	if s == nil {
		panic("pop of nil stack")
	}
	if len(s.Of) == 0 {
		panic("pop of empty stack")
	}

	v := s.Of[0]
	newVals := make([]E, len(s.Of)-1)
	copy(newVals, s.Of[1:])
	s.Of = newVals
	return v
}

// Peek checks the value at the top of the stack. If the stack is empty, panics.
func (s Stack[E]) Peek() E {
	if len(s.Of) == 0 {
		panic("peek of empty stack")
	}
	return s.Of[0]
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

// String shows the contents of the stack as a simple slice.
func (s Stack[E]) String() string {
	var sb strings.Builder

	sb.WriteString("Stack[")
	for i := range s.Of {
		sb.WriteString(fmt.Sprintf("%v", s.Of[i]))
		if i+1 < len(s.Of) {
			sb.WriteRune(',')
			sb.WriteRune(' ')
		}
	}
	sb.WriteRune(']')
	return sb.String()
}

// Equal returns whether two stacks have exactly the same contents in the same
// order. If anything other than a Stack[E], *Stack[E], []E, or *[]E is passed
// in, they will not be considered equal.
//
// A Stack[E] *is* considered equal to a *[]E or []E that has the same contents
// as the Stack in the same order.
//
// This does NOT do Equal on the individual items, but rather a simple equality
// check. To do full Equal on everything, use EqualSlices on the Ofs of the
// stacks.
func (s Stack[E]) Equal(o any) bool {
	other, ok := o.(Stack[E])
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*Stack[E])
		if !ok {
			// also okay if it's a slice
			otherSlice, ok := o.([]E)

			if !ok {
				// also okay if it's a ptr to slice
				otherSlicePtr, ok := o.(*[]E)
				if !ok {
					return false
				} else if otherSlicePtr == nil {
					return false
				} else {
					other = Stack[E]{Of: *otherSlicePtr}
				}
			} else {
				other = Stack[E]{Of: otherSlice}
			}
		} else if otherPtr == nil {
			return false
		} else {
			other = *otherPtr
		}
	}

	return reflect.DeepEqual(s.Of, other.Of)
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

func (s Set[E]) Len() int {
	return len(s)
}

// String shows the contents of the set. Items are not guaranteed to be in any
// particular order.
func (s Set[E]) String() string {
	var sb strings.Builder

	totalLen := s.Len()
	itemsWritten := 0

	sb.WriteRune('{')
	for k := range s {
		sb.WriteString(fmt.Sprintf("%v", k))
		itemsWritten++
		if itemsWritten < totalLen {
			sb.WriteRune(',')
			sb.WriteRune(' ')
		}
	}
	sb.WriteRune('}')
	return sb.String()
}

// Equal returns whether two sets have the same items. If anything other than a
// Set[E], *Set[E], []map[E]bool, or *[]map[E]bool is passed
// in, they will not be considered equal.
//
// A Stack[E] *is* considered equal to a *[]E or []E that has the same contents
// as the Stack in the same order.
//
// This does NOT do Equal on the individual items, but rather a simple equality
// check. To do full Equal on everything, use EqualSlices on the Ofs of the
// stacks.
func (s Set[E]) Equal(o any) bool {
	other, ok := o.(Set[E])
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*Set[E])
		if !ok {
			// also okay if it's a map
			otherMap, ok := o.(map[E]bool)

			if !ok {
				// also okay if it's a ptr to map
				otherMapPtr, ok := o.(*map[E]bool)
				if !ok {
					return false
				} else if otherMapPtr == nil {
					return false
				} else {
					other = Set[E](*otherMapPtr)
				}
			} else {
				other = Set[E](otherMap)
			}
		} else if otherPtr == nil {
			return false
		} else {
			other = *otherPtr
		}
	}

	if s.Len() != other.Len() {
		return false
	}

	for k := range s {
		if !other.Has(k) {
			return false
		}
	}

	return true
}

func SetFromSlice[E comparable](sl []E) Set[E] {
	if sl == nil {
		return nil
	}

	s := Set[E]{}

	for i := range sl {
		s.Add(sl[i])
	}

	return s
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
