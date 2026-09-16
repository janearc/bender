package sprite

import (
	"testing"
)

// a sprite with no name, no art or ragged rows is refused, by line.
func TestParseSpriteErrors(t *testing.T) {
	for _, src := range []string{
		"",
		"art\nKK\n",
		"sprite x\n",
		"sprite x\nart\nKZ\n",
		"sprite x\nart\nKK\nK\n",
		"sprite x\nart\nK\n\nart\nK\n",
	} {
		if _, err := ParseSprite(src); err == nil {
			t.Errorf("accepted %q", src)
		}
	}
	// Art ends at a comment as well as a blank line.
	s, err := ParseSprite("sprite x\nart\nKW\n# done\npose p\n")
	if err != nil || len(s.Art) != 1 {
		t.Errorf("comment ends art: %v %v", s, err)
	}
	if s.String() != "sprite x\nart\nKW\n# done\npose p\n" {
		t.Errorf("round trip %q", s.String())
	}
}
