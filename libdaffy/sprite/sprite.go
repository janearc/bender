package sprite

// A sprite is a stencil of roles: a grid of characters from the alphabet
// below, and no colour, because a colourway binds the roles when the
// sprite is drawn. Both a sprite and a canvas store parts rather than
// colours, so a sprite moves onto a canvas intact and back again.
//
// The alphabet is the contract. A sprite that fits it takes any colourway
// there is, including ones written after it, so export refuses a part
// outside the alphabet and names it, and never invents a role character.
//
// Poses are read and preserved on a round trip. They are not edited.

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// RoleChar is one entry of the alphabet.
type RoleChar struct {
	Char byte
	Name string
	What string
}

// Alphabet is the eleven role characters a stencil may contain, in the
// order a person reads them. The order carries nothing on the wire; a
// .sprite carries characters and never names. A name is one token, and the
// transparent character is called "none" rather than after what it does.
var Alphabet = []RoleChar{
	{'.', "none", "transparent"},
	{'K', "dark", "the dark mass"},
	{'W', "light", "the light mass"},
	{
		'V',
		"wing",
		"a tone just off dark, so a large dark area is not a blob",
	},
	{'D', "stripe", "a small dark detail on light"},
	{'B', "beak-base", "the first of three bands of whatever protrudes"},
	{'Y', "beak-band", "the second"},
	{'R', "beak-tip", "the third, usually largest, carrying the accent"},
	{'O', "feet", "whatever touches the ground"},
	{'E', "pupil", "the eye itself"},
	{'X', "eye-ring", "the eye's surround"},
}

// RoleByChar finds an alphabet entry, or false.
func RoleByChar(c byte) (RoleChar, bool) {
	for _, r := range Alphabet {
		if r.Char == c {
			return r, true
		}
	}
	return RoleChar{}, false
}

// RoleByName finds an alphabet entry by its part name, or false.
func RoleByName(name string) (RoleChar, bool) {
	for _, r := range Alphabet {
		if r.Name == name {
			return r, true
		}
	}
	return RoleChar{}, false
}

// Sprite is a .sprite file: its name, its art, and everything else in the
// file kept verbatim so a round trip reproduces the bytes.
type Sprite struct {
	Name string
	Art  []string
	// raw is every line of the file, with the art rows replaced by one
	// ArtMarker line, so the art can be regenerated and the rest emitted
	// as it was.
	Raw []string
}

// ArtMarker stands in for the art rows in raw.
const ArtMarker = "\x00art\x00"

// ParseSprite reads a .sprite. The format: a "sprite <name>" line, an
// "art" line followed by rows of role characters up to a blank line, and
// "pose <name>" blocks, which are kept and not interpreted. Comments start
// with a hash.
func ParseSprite(src string) (*Sprite, error) {
	lines := strings.Split(strings.TrimSuffix(src, "\n"), "\n")
	s := &Sprite{}
	inArt := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case inArt:
			if trimmed == "" || strings.HasPrefix(trimmed, "#") ||
				strings.HasPrefix(trimmed, "pose ") {
				inArt = false
				s.Raw = append(s.Raw, line)
				continue
			}
			for i := 0; i < len(line); i++ {
				if _, ok := RoleByChar(line[i]); !ok {
					return nil, badRole(line, line[i])
				}
			}
			s.Art = append(s.Art, line)
		case strings.HasPrefix(trimmed, "sprite "):
			s.Name = strings.TrimSpace(
				strings.TrimPrefix(trimmed, "sprite "),
			)
			s.Raw = append(s.Raw, line)
		case trimmed == "art":
			if s.Art != nil {
				return nil, fmt.Errorf("sprite: two art blocks")
			}
			inArt = true
			s.Raw = append(s.Raw, line, ArtMarker)
		default:
			s.Raw = append(s.Raw, line)
		}
	}
	if s.Name == "" {
		return nil, fmt.Errorf("sprite: no sprite line")
	}
	if len(s.Art) == 0 {
		return nil, fmt.Errorf("sprite: no art")
	}
	w := len(s.Art[0])
	for _, row := range s.Art {
		if len(row) != w {
			return nil, fmt.Errorf(
				"sprite: art rows are not all %d wide",
				w,
			)
		}
	}
	return s, nil
}

// LoadSprite reads a .sprite from a path.
func LoadSprite(path string) (*Sprite, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSprite(string(b))
}

// String writes the sprite back. With the file's other lines preserved,
// unchanged art gives back the same bytes; a sprite made from scratch gets
// the canonical shape.
func (s *Sprite) String() string {
	var b strings.Builder
	if s.Raw == nil {
		// Made from scratch: the canonical shape.
		b.WriteString("sprite " + s.Name + "\n\nart\n")
		for _, row := range s.Art {
			b.WriteString(row + "\n")
		}
		return b.String()
	}
	for _, line := range s.Raw {
		if line == ArtMarker {
			for _, row := range s.Art {
				b.WriteString(row + "\n")
			}
			continue
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// Write puts the sprite on a writer.
func (s *Sprite) Write(w io.Writer) error {
	_, err := io.WriteString(w, s.String())
	return err
}

// Width and Height in art cells, which are daffy pixels.
func (s *Sprite) Width() int { return len(s.Art[0]) }

// Height in art cells.
func (s *Sprite) Height() int { return len(s.Art) }

// --- into a document ---------------------------------------------------------

// badRole is the error for an art character that names no role.
func badRole(row string, ch byte) error {
	return fmt.Errorf(
		"sprite: art row %q has %q, which is not in the role alphabet",
		row, ch)
}
