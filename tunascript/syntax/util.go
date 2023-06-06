package syntax

// TODO: everyfin in this file this was copied from the internal util package so
// that it isn't internal to function with ictcc. Probably this indicates that
// the util package should be upd8ed.

// equalNilness returns whether the two values are either both nil or both
// non-nil.
func equalNilness(o1 any, o2 any) bool {
	if o1 == nil {
		return o2 == nil
	} else {
		return o2 != nil
	}
}

// customComparable is an interface for items that may be checked against
// arbitrary other objects. In practice most will attempt to typecast to their
// own type and immediately return false if the argument is not the same, but in
// theory this allows for comparison to multiple types of things.
type customComparable interface {
	Equal(other any) bool
}

// EqualSlices checks that the two slices contain the same items in the same
// order. Equality of items is checked by items in the slices are equal by
// calling the custom Equal function on each element. In particular, Equal is
// called on elements of sl1 with elements of sl2 passed in as the argument.
func equalSlices[T customComparable](sl1 []T, sl2 []T) bool {
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
