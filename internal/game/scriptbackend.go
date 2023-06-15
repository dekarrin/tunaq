package game

import "strings"

type scriptBackend struct {
	game *State
}

func (sb scriptBackend) InInventory(item string) bool {
	_, ok := sb.game.Inventory[strings.ToUpper(item)]
	return ok
}

func (sb scriptBackend) Move(target, dest string) bool {
	target = strings.ToUpper(target)
	dest = strings.ToUpper(dest)

	if _, ok := sb.game.World[dest]; !ok {
		// TODO: don't fail silently
		return false
	}
	if target == TagPlayer {
		if sb.game.CurrentRoom.Label == dest {
			return false
		}
		sb.game.CurrentRoom = sb.game.World[dest]
		return true
	} else {
		// item?
		if roomLabel, ok := sb.game.itemLocations[target]; ok {
			if roomLabel == dest {
				return false
			}

			var item *Item
			if roomLabel == "@INVEN" {
				// it DOES move from backpack
				item = sb.game.Inventory[target]
				delete(sb.game.Inventory, item.Label)
			} else {
				// get the item
				for _, it := range sb.game.World[roomLabel].Items {
					if it.Label == target {
						item = it
						break
					}
				}
				sb.game.World[roomLabel].RemoveItem(target)
			}

			if dest == "@INVEN" {
				sb.game.Inventory[target] = item
			} else {
				sb.game.World[dest].Items = append(sb.game.World[dest].Items, item)
			}
			sb.game.itemLocations[target] = dest

			return true
		}

		// npc?
		roomLabel, ok := sb.game.npcLocations[target]
		if !ok {
			return false
		}
		if roomLabel == dest {
			return false
		}

		npc := sb.game.World[roomLabel].NPCs[target]
		delete(sb.game.World[roomLabel].NPCs, npc.Label)
		sb.game.World[dest].NPCs[npc.Label] = npc
		sb.game.npcLocations[target] = dest
		return true
	}
}

func (sb scriptBackend) Output(s string) bool {
	if sb.game.tsBufferOutput {
		sb.game.tsBuf.WriteString(s)
		return true
	}

	err := sb.game.io.Output(s)
	return err == nil
}
