# converts a .JSON format file into the new TOML-based Tunaquest World (TQW)
# format

import sys
import json
import textwrap


def main():
    if len(sys.argv) < 2:
        print("give name of file to convert as first arg and file to write output to as second arg", file=sys.stderr)
        exit(1)
    if len(sys.argv) < 3:
        print("give name of file to write output to as second arg")

    filename = str(sys.argv[1])
    with open(filename, 'r') as fp:
        try:
            json_data = json.load(fp)
        except json.JSONDecodeError as e:
            print("malormed JSON in file: " + str(e))
            exit(2)

    output_file = str(sys.argv[2])
    with open(output_file, 'w') as fp:
        # print preamble
        print('format = "TUNA"', file=fp)
        print('type = "DATA"', file=fp)
        print('', file=fp)

        # print world (pretty much just the start key)
        print('[world]', file=fp)
        if 'start' in json_data:
            print('start = "' + str(json_data['start']) + '"', file=fp)
        else:
            print("WARN: input is missing key 'start'; world.start will be undefined in output", file=sys.stderr)
            print('start = ""', file=fp)
        print('', file=fp)
        print('', file=fp)

        # print each room
        for r in json_data.get('rooms', []):
            print('# ' + r.get('label', '').lower(), file=fp)
            print('[[rooms]]', file=fp)
            print('label = "' + r.get('label', '') + '"', file=fp)
            print('name = "' + r.get('name', '') + '"', file=fp)
            print("description = '''", file=fp)
            for l in textwrap.wrap(r.get('description', ''), 120):
                print(l, file=fp)
            print("'''", file=fp)
            print('', file=fp)
            
            for eg in r.get('exits', []):
                print('[[rooms.exits]]', file=fp)
                print('aliases = ' + format_inline_text_list(eg.get('aliases', [])), file=fp)
                print('destLabel = "' + eg.get('destLabel', '') + '"', file=fp)
                print('description = "' + eg.get('description', '') + '"', file=fp)
                print("travelMessage = '''", file=fp)
                for l in textwrap.wrap(eg.get('travelMessage', ''), 120):
                    print(l, file=fp)
                print("'''", file=fp)
                print('', file=fp)

            for it in r.get('items', []):
                print('[[rooms.items]]', file=fp)
                print('label = "' + it.get('label', '') + '"', file=fp)
                print('aliases = ' + format_inline_text_list(it.get('aliases', [])), file=fp)
                print('name = "' + it.get('name', '') + '"', file=fp)
                print("description = '''", file=fp)
                for l in textwrap.wrap(eg.get('description', ''), 120):
                    print(l, file=fp)
                print("'''", file=fp)
                print('', file=fp)
            
            print('', file=fp)

        # print each pronoun set
        pronoun_sets = json_data.get('pronouns', {})
        for k in pronoun_sets:
            ps = pronoun_sets[k]
            print('[pronouns."' + str(k) + '"]', file=fp)
            print('nominative = "' + ps.get('nominative', '') + '"', file=fp)
            print('objective = "' + ps.get('objective', '') + '"', file=fp)
            print('possessive = "' + ps.get('possessive', '') + '"', file=fp)
            print('determiner = "' + ps.get('determiner', '') + '"', file=fp)
            print('reflexive = "' + ps.get('reflexive', '') + '"', file=fp)
            print('', file=fp)
        if len(pronoun_sets) > 0:
            print('', file=fp)

        # print each npc
        for npc in json_data.get('npcs', []):
            print('[[npcs]]', file=fp)
            print('label = "' + npc.get('label', '') + '"', file=fp)
            print('aliases = ' + format_multiline_text_list(npc.get('aliases', [])), file=fp)
            print('name = "' + npc.get('name', '') + '"', file=fp)
            print('start = "' + npc.get('start', '') + '"', file=fp)

            pn = npc.get('pronouns', "")
            pronouns_after = False

            if isinstance(pn, str):
                print('pronouns = "' + str(pn) + '"', file=fp)
            else:
                pronouns_after = True

            print("description = '''", file=fp)
            for l in textwrap.wrap(npc.get('description', ''), 120):
                print(l, file=fp)
            print("'''", file=fp)
            print('', file=fp)

            if pronouns_after:
                print("[npcs.pronouns]", file=fp)
                print('nominative = "' + pn.get('nominative', '') + '"', file=fp)
                print('objective = "' + pn.get('objective', '') + '"', file=fp)
                print('possessive = "' + pn.get('possessive', '') + '"', file=fp)
                print('determiner = "' + pn.get('determiner', '') + '"', file=fp)
                print('reflexive = "' + pn.get('reflexive', '') + '"', file=fp)
                print('', file=fp)

            mv = npc.get('movement', {})
            if mv is not None:
                act = mv.get('action', 'STATIC')
                print('[npcs.movement]', file=fp)
                print('action = "' + act + '"', file=fp)
                if 'path' in mv:
                    print('path = ' + format_multiline_text_list(mv['path']), file=fp)
                if 'allowedRooms' in mv:
                    print('allowedRooms = ' + format_multiline_text_list(mv['allowedRooms']), file=fp)
                if 'forbiddenRooms' in mv:
                    print('forbiddenRooms = ' + format_multiline_text_list(mv['forbiddenRooms']), file=fp)
                print('', file=fp)

            # TODO: rename this field to 'dialogs' or somefin ending in plural to
            # stay consistent at least with rule of 'ending in s, add more brackets!'
            for dia in npc.get('dialog', []):
                if isinstance(dia, str):
                    dia = {
                        'content': dia,
                    }
                print('[[npcs.dialog]]', file=fp)
                if 'label' in dia:
                    print('label = "' + dia['label'] + '"', file=fp)
                if 'action' in dia:
                    print('action = "' + dia['action'] + '"', file=fp)
                if 'content' in dia:
                    print('content = "' + dia['content'] + '"', file=fp)
                if 'response' in dia:
                    print('response = "' + dia['response'] + '"', file=fp)
                if 'choices' in dia:
                    print('choices = ' + format_multiline_text_list(dia['choices'], format_inline_text_list), file=fp)
                print('', file=fp)


def format_multiline_text_list(text_list, op=None):
    if op is None:
        op = lambda s: '"' + str(s) + '"'

    out = "["
    if len(text_list) > 0:
        out += "\n"
    for t in text_list:
        out += "\t"
        out += op(t)
        out += ",\n"
    out += ']'
    return out


def format_inline_text_list(text_list):
    out = "["
    itemNum = 0
    for t in text_list:
        out += "\""
        out += str(t)
        out += "\""
        if itemNum + 1 < len(text_list):
            out += ", "

        itemNum += 1
    out += ']'
    return out
    

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        pass

