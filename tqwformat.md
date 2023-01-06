This is a quick and dirty description of the data files that TunaQuest uses to
define worlds that it runs.

World definitions are in a format of files called "TunaQuest Worlds Format", or
"TQW". TQW is a TOML-based text file format, with additional semantic
restrictions. A TQW file may be referred to by this library as a "Resource" file
as all of its world resources will be in that format.

All TQW files have some common keys at the top level, and from there the
contents vary based on the specific type of the file.

### Case Sensitivity
In TQW files, the keys themselves are always case sensitive. The values are
mostly case-sensitive; however, `label` values, `aliases` values, and other
values that reference labels and aliases are explicitly case-insensitive. They
will be marked as such in their section.

The values for the common keys `format` and `type` are always case-insensitive.

### Common Keys
The TQW format requires that there be two keys defined in every TQW file, in the
top-level TOML table (so, before the first "[]" or "[[]]" header). By convention
these are the first two keys in the first two lines of the file, but this is not
strictly required.

The keys are:
    * `format` - This must always be set to "tuna" (case-insensitive)
    * `type` - This must be set to one of "data" or "manifest" (case-insensitive)

# Types

### Manifest

Keys:
    * `format` - Always set to "tuna" (case-insens)
    * `type` - Always set to "manifest" (case-insens)
    * `files` - A list of files to include, relative to the manifest file. This
    can be other manifest files.

### World Data

Keys:
    * 