package util

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

type Container[E any] interface {
	// Elements returns a slice of the elements. They are not garaunteed to be
	// in any particular order.
	Elements() []E
}

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

type sorter[E any] struct {
	src []E
	lt  func(left, right E) bool
}

func (s sorter[E]) Len() int {
	return len(s.src)
}

func (s sorter[E]) Swap(i, j int) {
	s.src[i], s.src[j] = s.src[j], s.src[i]
}

func (s sorter[E]) Less(i, j int) bool {
	return s.lt(s.src[i], s.src[j])
}

// TruncateWith truncates s to maxLen. If s is longer than maxLen, it is
// truncated and the cont string is placed after it. If it is shorter or equal
// to maxLen, it is returned unchanged.
func TruncateWith(s string, maxLen int, cont string) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen] + cont
}

// SortBy takes the items and uses the provided function to sort the list. The
// function should return true if left is less than (comes before) right.
//
// items will not be modified.
func SortBy[E any](items []E, lt func(left E, right E) bool) []E {
	if len(items) == 0 || lt == nil {
		return items
	}

	s := sorter[E]{
		src: make([]E, len(items)),
		lt:  lt,
	}

	copy(s.src, items)
	sort.Sort(s)
	return s.src
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

// OrdinalSuf returns the suffix for the ordinal version of the given number,
// e.g. "rd" for 3, "st" for 51, etc.
func OrdinalSuf(a int) string {
	// first, if negative, just give the prefix for the positive
	if a < 0 {
		a *= -1
	}

	// special exception for the english language: 11th, 12th, and 13th break
	// the "by 1's place" rule, and need to be allowed for explicitly
	if a == 11 || a == 12 || a == 13 {
		return "th"
	}

	finalDigit := a

	if a > 9 {
		// it all depends on the final digit
		nextTen := int(math.Floor(float64(a)/10) * 10)
		finalDigit = a - nextTen
	}

	if finalDigit == 1 {
		return "st"
	} else if finalDigit == 2 {
		return "nd"
	} else if finalDigit == 3 {
		return "rd"
	} else {
		return "th"
	}
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

type namedSortable[V any] struct {
	val  V
	name string
}

// Alphabetized returns the ordered elements of the given container. The string
// representation '%v' of each element is used for comparison.
func Alphabetized[V any](c Container[V]) []V {
	// convert them all to string and order that.

	toSort := []namedSortable[V]{}

	for _, item := range c.Elements() {
		itemStr := fmt.Sprintf("%v", item)
		toSort = append(toSort, namedSortable[V]{val: item, name: itemStr})
	}

	sortFunc := func(i, j int) bool {
		return toSort[i].name < toSort[j].name
	}

	sort.Slice(toSort, sortFunc)

	sortedVals := make([]V, len(toSort))
	for i := range toSort {
		sortedVals[i] = toSort[i].val
	}

	return sortedVals
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
