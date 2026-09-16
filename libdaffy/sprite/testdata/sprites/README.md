# sprite fixtures

`stranger.sprite` uses all eleven roles. The round-trip test reads it,
writes it back and expects the same bytes, which is how the format is
proven rather than assumed. Read, never sent back.
