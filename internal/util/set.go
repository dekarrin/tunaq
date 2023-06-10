package util

import (
	"fmt"
	"sort"
	"strings"
)

type ISet[E any] interface {
	Container[E]

	// Add adds the given element to the Set. If the element is already in the
	// set, no effect occurs.
	Add(element E)

	// AddAll adds all elements in s2 to the Set.
	AddAll(s2 ISet[E])

	// Remove removes the given element from the Set. If the element is already
	// not in the set, no effect occurs.
	Remove(element E)

	// Has returns whether the given set has the specified element.
	Has(element E) bool

	// Len returns the number of elements in the set.
	Len() int

	// Copy returns a copy of the Set.
	Copy() ISet[E]

	// Equal returns whether a Set equals another value. It should check if the
	// value implements Set and if so, does a comparison of the elements and
	// not of their ordering. For those sets which implement value mapping to
	// elements, this does NOT compare the data values.
	Equal(o any) bool

	// String is a string with the contents of the set, not gauranateed to be in
	// any particular order.
	String() string

	// StringOrdered is a string with the contents of the set, ordered
	// alphabetically.
	StringOrdered() string

	// Union returns a new Set that is the union of s and o.
	Union(s2 ISet[E]) ISet[E]

	// Intersection returns a new Set that contains the elements that are in both
	// s and o.
	Intersection(s2 ISet[E]) ISet[E]

	// Difference returns a new Set that contains the elements that are in the
	// set but not in s2.
	Difference(s2 ISet[E]) ISet[E]

	// DisjointWith returns whether the set is disjoint (contains no elements
	// of) s2.
	DisjointWith(s2 ISet[E]) bool

	// Empty returns whether the set is empty.
	Empty() bool

	// Any returns whether any element in the set meets some condition.
	Any(predicate func(v E) bool) bool
}

// VSet is a set that contains values mapped to items.
type VSet[E any, V any] interface {
	ISet[E]

	// Set assigns the value of the element. The element is added if it isn'
	// already in the set, and that element is assigned the given data value.
	Set(element E, data V)

	// Get retrieves the value of an element. The value of the element is
	// returned if it exists, otherwise the zero-value for V is returned.
	Get(element E) V
}

// Set that uses strings as its item type and some other type as its stored
// data type.
type SVSet[V any] map[string]V

func NewSVSet[V any](of ...map[string]V) SVSet[V] {
	bs := SVSet[V](map[string]V{})
	for _, m := range of {
		for k := range m {
			bs.Set(k, m[k])
		}
	}
	return bs
}

func (s SVSet[V]) Copy() ISet[string] {
	return NewSVSet(s)
}

// Add adds an index. Has no effect if it's already there.
func (s SVSet[V]) Add(idx string) {
	newRef := new(V)
	s[idx] = *newRef
}

func (s SVSet[V]) Set(idx string, val V) {
	s[idx] = val
}

func (s SVSet[V]) Get(idx string) V {
	return s[idx]
}

func (s SVSet[V]) Has(idx string) bool {
	_, ok := s[idx]
	return ok
}

func (s SVSet[V]) Remove(idx string) {
	delete(s, idx)
}

func (s SVSet[V]) Len() int {
	return len(s)
}

func (s SVSet[V]) Elements() []string {
	elems := []string{}
	for k := range s {
		elems = append(elems, k)
	}
	return elems
}

func (s SVSet[V]) AddAll(s2 ISet[string]) {
	// if this is also a VSet[string, E], then we go by value
	valuedSet, isValued := s2.(VSet[string, V])
	if isValued {
		for _, k := range valuedSet.Elements() {
			s.Add(k)
			s.Set(k, valuedSet.Get(k))
		}
	} else {
		for _, k := range s2.Elements() {
			s.Add(k)
		}
	}
}

func (s SVSet[V]) Union(s2 ISet[string]) ISet[string] {
	newSet := s.Copy()

	newSet.AddAll(s)
	newSet.AddAll(s2)

	return newSet
}

// Intersection returns a new Set that contains the elements that are in both
// s and o.
func (s SVSet[V]) Intersection(s2 ISet[string]) ISet[string] {
	newSet := NewSVSet[V]()

	for k := range s {
		if s2.Has(k) {
			newSet.Add(k)
			newSet.Set(k, s.Get(k))
		}
	}

	return newSet
}

// Difference returns a new Set that contains the elements that are in s but not
// in o.
func (s SVSet[V]) Difference(o ISet[string]) ISet[string] {
	newSet := NewSVSet(s)

	for _, k := range o.Elements() {
		newSet.Remove(k)
	}

	return newSet
}

func (s SVSet[V]) DisjointWith(o ISet[string]) bool {
	for k := range s {
		if o.Has(k) {
			return false
		}
	}
	return true
}

func (s SVSet[V]) Empty() bool {
	return s.Len() == 0
}

func (s SVSet[V]) Any(predicate func(v string) bool) bool {
	for k := range s {
		if predicate(k) {
			return true
		}
	}
	return false
}

// StringOrdered shows the contents of the set. Items are guaranteed to be
// alphabetized.
func (s SVSet[V]) StringOrdered() string {
	convs := []string{}

	for k := range s {
		convs = append(convs, fmt.Sprintf("%v", k))
	}

	sort.Strings(convs)

	var sb strings.Builder

	sb.WriteRune('{')
	for i := range convs {
		sb.WriteString(convs[i])
		if i+1 < len(convs) {
			sb.WriteRune(',')
			sb.WriteRune(' ')
		}
	}
	sb.WriteRune('}')
	return sb.String()
}

// String shows the contents of the set. Items are not guaranteed to be in any
// particular order.
func (s SVSet[V]) String() string {
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
// Set[E], *Set[E], they will not be considered equal.
func (s SVSet[V]) Equal(o any) bool {
	other, ok := o.(ISet[string])
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*ISet[string])
		if !ok {
			return false
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

// StringSet is a map[string]bool with methods added to fulfill ISet[string]
type StringSet map[string]bool

func NewStringSet(of ...map[string]bool) StringSet {
	s := StringSet{}
	for _, m := range of {
		for k := range m {
			s.Add(k)
		}
	}
	return s
}

func (s StringSet) Copy() ISet[string] {
	newS := NewStringSet()

	for k := range s {
		newS[k] = true
	}

	return newS
}

// Union returns a new Set that is the union of s and o.
func (s StringSet) Union(o ISet[string]) ISet[string] {
	newSet := NewStringSet()
	newSet.AddAll(s)
	newSet.AddAll(o)

	return newSet
}

// Intersection returns a new Set that contains the elements that are in both
// s and o.
func (s StringSet) Intersection(o ISet[string]) ISet[string] {
	newSet := NewStringSet()

	for k := range s {
		if o.Has(k) {
			newSet.Add(k)
		}
	}

	return newSet
}

// Difference returns a new Set that contains the elements that are in s but not
// in o.
func (s StringSet) Difference(o ISet[string]) ISet[string] {
	newSet := NewStringSet()
	newSet.AddAll(s)

	for _, k := range o.Elements() {
		newSet.Remove(k)
	}

	return newSet
}

func (s StringSet) DisjointWith(o ISet[string]) bool {
	for k := range s {
		if o.Has(k) {
			return false
		}
	}
	return true
}

func (s StringSet) Empty() bool {
	return s.Len() == 0
}

func (s StringSet) Any(predicate func(v string) bool) bool {
	for k := range s {
		if predicate(k) {
			return true
		}
	}
	return false
}

func (s StringSet) Has(value string) bool {
	_, has := s[value]
	return has
}

func (s StringSet) Add(value string) {
	s[value] = true
}

func (s StringSet) Remove(value string) {
	delete(s, value)
}

func (s StringSet) Len() int {
	return len(s)
}

func (s StringSet) AddAll(s2 ISet[string]) {
	for _, element := range s2.Elements() {
		s.Add(element)
	}
}

// StringOrdered shows the contents of the set. Items are guaranteed to be
// alphabetized.
func (s StringSet) StringOrdered() string {
	convs := []string{}

	for k := range s {
		convs = append(convs, fmt.Sprintf("%v", k))
	}

	sort.Strings(convs)

	var sb strings.Builder

	sb.WriteRune('{')
	for i := range convs {
		sb.WriteString(convs[i])
		if i+1 < len(convs) {
			sb.WriteRune(',')
			sb.WriteRune(' ')
		}
	}
	sb.WriteRune('}')
	return sb.String()
}

// String shows the contents of the set. Items are not guaranteed to be in any
// particular order.
func (s StringSet) String() string {
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
func (s StringSet) Equal(o any) bool {
	other, ok := o.(ISet[string])
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*ISet[string])
		if !ok {
			return false
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

// Slice returns the elements of s as a slice. No particular order is
// guaranteed nor should it be relied on.
func (s StringSet) Elements() []string {
	if s == nil {
		return nil
	}

	sl := make([]string, 0)

	for item := range s {
		sl = append(sl, item)
	}

	return sl
}

func StringSetOf(sl []string) StringSet {
	if sl == nil {
		return nil
	}

	s := StringSet{}

	for i := range sl {
		s.Add(sl[i])
	}

	return s
}

// KeySet is a map[E comparable]bool with methods added to fulfill ISet[E]
type KeySet[E comparable] map[E]bool

func NewKeySet[E comparable](of ...map[E]bool) KeySet[E] {
	s := KeySet[E]{}
	for _, m := range of {
		for k := range m {
			s.Add(k)
		}
	}
	return s
}

func (s KeySet[E]) Copy() ISet[E] {
	newS := NewKeySet[E]()

	for k := range s {
		newS[k] = true
	}

	return newS
}

// Union returns a new Set that is the union of s and o.
func (s KeySet[E]) Union(o ISet[E]) ISet[E] {
	newSet := NewKeySet[E]()
	newSet.AddAll(s)
	newSet.AddAll(o)

	return newSet
}

// Intersection returns a new Set that contains the elements that are in both
// s and o.
func (s KeySet[E]) Intersection(o ISet[E]) ISet[E] {
	newSet := NewKeySet[E]()

	for k := range s {
		if o.Has(k) {
			newSet.Add(k)
		}
	}

	return newSet
}

// Difference returns a new Set that contains the elements that are in s but not
// in o.
func (s KeySet[E]) Difference(o ISet[E]) ISet[E] {
	newSet := NewKeySet[E]()
	newSet.AddAll(s)

	for _, k := range o.Elements() {
		newSet.Remove(k)
	}

	return newSet
}

func (s KeySet[E]) DisjointWith(o ISet[E]) bool {
	for k := range s {
		if o.Has(k) {
			return false
		}
	}
	return true
}

func (s KeySet[E]) Empty() bool {
	return s.Len() == 0
}

func (s KeySet[E]) Any(predicate func(v E) bool) bool {
	for k := range s {
		if predicate(k) {
			return true
		}
	}
	return false
}

func (s KeySet[E]) Has(value E) bool {
	_, has := s[value]
	return has
}

func (s KeySet[E]) Add(value E) {
	s[value] = true
}

func (s KeySet[E]) Remove(value E) {
	delete(s, value)
}

func (s KeySet[E]) Len() int {
	return len(s)
}

func (s KeySet[E]) AddAll(s2 ISet[E]) {
	for _, element := range s2.Elements() {
		s.Add(element)
	}
}

// StringOrdered shows the contents of the set. Items are guaranteed to be
// alphabetized.
func (s KeySet[E]) StringOrdered() string {
	convs := []string{}

	for k := range s {
		convs = append(convs, fmt.Sprintf("%v", k))
	}

	sort.Strings(convs)

	var sb strings.Builder

	sb.WriteRune('{')
	for i := range convs {
		sb.WriteString(convs[i])
		if i+1 < len(convs) {
			sb.WriteRune(',')
			sb.WriteRune(' ')
		}
	}
	sb.WriteRune('}')
	return sb.String()
}

// String shows the contents of the set. Items are not guaranteed to be in any
// particular order.
func (s KeySet[E]) String() string {
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
func (s KeySet[E]) Equal(o any) bool {
	other, ok := o.(ISet[E])
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*ISet[E])
		if !ok {
			return false
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

// Slice returns the elements of s as a slice. No particular order is
// guaranteed nor should it be relied on.
func (s KeySet[E]) Elements() []E {
	if s == nil {
		return nil
	}

	sl := make([]E, 0)

	for item := range s {
		sl = append(sl, item)
	}

	return sl
}

func KeySetOf[E comparable](sl []E) KeySet[E] {
	if sl == nil {
		return nil
	}

	s := NewKeySet[E]()

	for i := range sl {
		s.Add(sl[i])
	}

	return s
}
