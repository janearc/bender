package canvas

// the internals the external tests exercise, reachable by name.
var Wrap = wrap

// NextID is the id the regions will hand out next.
func NextID(r *Regions) int { return r.nextID }
