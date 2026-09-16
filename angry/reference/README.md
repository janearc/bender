# Vendored reference, read-only

`textmap.py` and `hexify.py` are copied verbatim from
`~/mesh/dev/kingfisher`, taken 2026-09-07.

They are here so this project can be read and built without reaching into
kingfisher's tree, which belongs to another session and sits inside the
migration enclave.

Do not edit these files, do not import from that tree, and do not send
changes back through this copy. If kingfisher's originals move on, this copy
goes stale silently, and that is acceptable: it is a design reference, not a
dependency.

`textmap.py` is the one that matters. It is the same pipeline this project
ports to Go, already factored into the three objects worth keeping, and it
already carries the projection interface that makes a globe cheap.

`hexify.py` is included for context on how cells are produced -- cell-driven
rather than pixel-driven -- which this project consumes but does not do.
