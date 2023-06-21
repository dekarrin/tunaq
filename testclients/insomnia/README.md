This JSON can be imported as a new Collection into Insomnia.

It requires having the `insomnia-plugin-save-variables` plugin.

Make sure when saving, you output to a json file (not committed), take the JSON
file and do `jq -r '.' <jsonFile >tunaquest.json` to get a file suitable for
version control.

