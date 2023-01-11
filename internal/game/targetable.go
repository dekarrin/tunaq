package game

// Targetable is something that can be targeted by a player command. All can be
// looked at.
type Targetable interface {
	// GetAliases returns all names that the player may use to refer to the
	/// thing.
	GetAliases() []string

	// GetDescription returns the description to show when the player looks at
	// it.
	GetDescription() string

	/*
		// CanUse returns whether it is possible to use the Targetable on its own.
		CanUse() bool

		// Use attempts to use the targetable by itself. This may be possible in a
		// few cases: the Targetable is an Item, or the Targetable is a Detail in
		// the room.
		//
		// Returns false if the Targetable could not be used.
		Use() error

		// CanUseWith returns whether it is possible to use the Targetable with the
		// given*/
}

// IsItem returns whether the Targetable is an Item and thus can be picked
// up by the player and placed in inventory.
func IsItem(t Targetable) bool {
	_, ok := t.(Item)
	if !ok {
		_, ok = t.(*Item)
		if !ok {
			return false
		}
	}
	return true
}

// IsNPC returns whether the Targetable is an NPC and thus can be talked to
// by the player.
func IsNPC(t Targetable) bool {
	_, ok := t.(NPC)
	if !ok {
		_, ok = t.(*NPC)
		if !ok {
			return false
		}
	}
	return true
}

// IsEgress returns whether the Targetable is an Egress and thus can be
// traversed by the player.
func IsEgress(t Targetable) bool {
	_, ok := t.(Egress)
	if !ok {
		_, ok = t.(*Egress)
		if !ok {
			return false
		}
	}
	return true
}

// IsDetail returns whether the Targetable is a room detail and thus can be
// only be looked at by the player.
func IsDetail(t Targetable) bool {
	_, ok := t.(Detail)
	if !ok {
		_, ok = t.(*Detail)
		if !ok {
			return false
		}
	}
	return true
}
