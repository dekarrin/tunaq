# converts a .JSON format file into the new TOML-based Tunaquest World (TQW)
# format

import sys
import json
import textwrap


def main():
    if len(sys.argv) < 2:
        print("give name of file to convert as first arg", file=sys.stderr)
        exit(1)

    filename = str(sys.argv[1])
    with open(filename, 'r') as fp:
        try:
            json_data = json.load(fp)
        except json.JSONDecodeError as e:
            print("malormed JSON in file: " + str(e))
            exit(2)

    # print preamble
    print('format = "TUNA"')
    print('type = "DATA"')
    print('')

    # print world (pretty much just the start key)
    print('[world]')
    if 'start' in json_data:
        print('start = "' + str(json_data['start']) + '"')
    else:
        print("WARN: input is missing key 'start'; world.start will be undefined in output", file=sys.stderr)
        print('start = ""')
    print()
    print()

    # print each room
    for r in json_data.get('rooms', []):
        print('# ' + r.get('label', '').lower())
        print('[[rooms]]')
        print('label = "' + r.get('label', '') + '"')
        print('name = "' + r.get('name', '') + '"')
        print("description = '''")
        for l in textwrap.wrap(r.get('description', ''), 120):
            print(l)
        print("'''")
        print()
        
        for eg in r.get('exits', []):
            print('[[rooms.exits]]')
            print('aliases = ' + format_inline_text_list(eg.get('aliases', [])))
            print('destLabel = "' + eg.get('destLabel', '') + '"')
            print('description = "' + eg.get('description', '') + '"')
            print("travelMessage = '''")
            for l in textwrap.wrap(eg.get('travelMessage', ''), 120):
                print(l)
            print("'''")
            print()

        for it in r.get('items', []):
            print('[[rooms.items]]')
            print('label = "' + it.get('label', '') + "'")
            print('aliases = ' + format_inline_text_list(it.get('aliases', [])))
            print('name = "' + it.get('name', '') + "'")
            print("description = '''")
            for l in textwrap.wrap(eg.get('description', ''), 120):
                print(l)
            print("'''")
            print()
        
        print()

    # print each pronoun set
    pronoun_sets = json_data.get('pronouns', {})
    for k in pronoun_sets:
        ps = pronoun_sets[k]
        print('[pronouns."' + str(k) + '"]')
        print('nominative = "' + ps.get('nominative', '') + '"')
        print('objective = "' + ps.get('objective', '') + '"')
        print('possessive = "' + ps.get('possessive', '') + '"')
        print('determiner = "' + ps.get('determiner', '') + '"')
        print('reflexive = "' + ps.get('reflexive', '') + '"')
        print()
    if len(pronoun_sets) > 0:
        print()

    # print each npc
    for npc in json_data.get('npcs', []):
        print('[[npcs]]')
        print('label = "' + npc.get('label', '') + '"')
        print('aliases = ' + format_multiline_text_list(npc.get('aliases', [])))
        print('name = "' + npc.get('name', '') + '"')
        print('start = "' + npc.get('start', '') + '"')

        pn = npc.get('pronouns', "")
        pronouns_after = False

        if isinstance(pn, str):
            print('pronouns = "' + str(pn) + "'")
        else:
            pronouns_after = True

        print("description = '''")
        for l in textwrap.wrap(npc.get('description', ''), 120):
            print(l)
        print("'''")
        print()

        if pronouns_after:
            print("[npcs.pronouns]")
            print('nominative = "' + pn.get('nominative', '') + '"')
            print('objective = "' + pn.get('objective', '') + '"')
            print('possessive = "' + pn.get('possessive', '') + '"')
            print('determiner = "' + pn.get('determiner', '') + '"')
            print('reflexive = "' + pn.get('reflexive', '') + '"')
            print()

        mv = npc.get('movement', {})
        if mv is not None:
            act = mv.get('action', 'STATIC')
            print('[npcs.movement]')
            print('action = "' + act + '"')
            if 'path' in mv:
                print('path = ' + format_multiline_text_list(mv['path']))
            if 'allowedRooms' in mv:
                print('allowedRooms = ' + format_multiline_text_list(mv['allowedRooms']))
            if 'forbiddenRooms' in mv:
                print('forbiddenRooms = ' + format_multiline_text_list(mv['forbiddenRooms']))
            print()

        # TODO: rename this field to 'dialogs' or somefin ending in plural to
        # stay consistent at least with rule of 'ending in s, add more brackets!'
        for dia in npc.get('dialog', []):
            if isinstance(dia, str):
                dia = {
                    'content': dia,
                }
            print('[[npcs.dialog]]')
            if 'label' in dia:
                print('label = "' + dia['label'] + '"')
            if 'action' in dia:
                print('action = "' + dia['action'] + '"')
            if 'content' in dia:
                print('content = "' + dia['content'] + '"')
            if 'response' in dia:
                print('response = "' + dia['response'] + '"')
            if 'choices' in dia:
                print('choices = ' + format_multiline_text_list(dia['choices'], format_inline_text_list))
            print()


def format_multiline_text_list(text_list, op=str):
    out = "["
    if len(text_list) > 0:
        out += "\n"
    for t in text_list:
        out += "\t\""
        out += op(t)
        out += "\",\n"
    out += ']'
    return out


def format_inline_text_list(text_list):
    out = "["
    itemNum = 0
    for t in text_list:
        out += "\""
        out += str(t)
        out += "\","
        if itemNum + 1 < len(text_list):
            out += " "

        itemNum += 1
    out += ']'
    return out
    

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        pass

