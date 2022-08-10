package svgpath

import (
	"fmt"
	"strconv"
	"strings"
)

// svg-path:
//     wsp* moveto-drawto-command-groups? wsp*
// moveto-drawto-command-groups:
//     moveto-drawto-command-group
//     | moveto-drawto-command-group wsp* moveto-drawto-command-groups
// moveto-drawto-command-group:
//     moveto wsp* drawto-commands?
// drawto-commands:
//     drawto-command
//     | drawto-command wsp* drawto-commands
// drawto-command:
//     closepath
//     | lineto
//     | horizontal-lineto
//     | vertical-lineto
//     | curveto
//     | smooth-curveto
//     | quadratic-bezier-curveto
//     | smooth-quadratic-bezier-curveto
//     | elliptical-arc
// moveto:
//     ( "M" | "m" ) wsp* moveto-argument-sequence
// moveto-argument-sequence:
//     coordinate-pair
//     | coordinate-pair comma-wsp? lineto-argument-sequence
// closepath:
//     ("Z" | "z")
// lineto:
//     ( "L" | "l" ) wsp* lineto-argument-sequence
// lineto-argument-sequence:
//     coordinate-pair
//     | coordinate-pair comma-wsp? lineto-argument-sequence
// horizontal-lineto:
//     ( "H" | "h" ) wsp* horizontal-lineto-argument-sequence
// horizontal-lineto-argument-sequence:
//     coordinate
//     | coordinate comma-wsp? horizontal-lineto-argument-sequence
// vertical-lineto:
//     ( "V" | "v" ) wsp* vertical-lineto-argument-sequence
// vertical-lineto-argument-sequence:
//     coordinate
//     | coordinate comma-wsp? vertical-lineto-argument-sequence
// curveto:
//     ( "C" | "c" ) wsp* curveto-argument-sequence
// curveto-argument-sequence:
//     curveto-argument
//     | curveto-argument comma-wsp? curveto-argument-sequence
// curveto-argument:
//     coordinate-pair comma-wsp? coordinate-pair comma-wsp? coordinate-pair
// smooth-curveto:
//     ( "S" | "s" ) wsp* smooth-curveto-argument-sequence
// smooth-curveto-argument-sequence:
//     smooth-curveto-argument
//     | smooth-curveto-argument comma-wsp? smooth-curveto-argument-sequence
// smooth-curveto-argument:
//     coordinate-pair comma-wsp? coordinate-pair
// quadratic-bezier-curveto:
//     ( "Q" | "q" ) wsp* quadratic-bezier-curveto-argument-sequence
// quadratic-bezier-curveto-argument-sequence:
//     quadratic-bezier-curveto-argument
//     | quadratic-bezier-curveto-argument comma-wsp?
//         quadratic-bezier-curveto-argument-sequence
// quadratic-bezier-curveto-argument:
//     coordinate-pair comma-wsp? coordinate-pair
// smooth-quadratic-bezier-curveto:
//     ( "T" | "t" ) wsp* smooth-quadratic-bezier-curveto-argument-sequence
// smooth-quadratic-bezier-curveto-argument-sequence:
//     coordinate-pair
//     | coordinate-pair comma-wsp? smooth-quadratic-bezier-curveto-argument-sequence
// elliptical-arc:
//     ( "A" | "a" ) wsp* elliptical-arc-argument-sequence
// elliptical-arc-argument-sequence:
//     elliptical-arc-argument
//     | elliptical-arc-argument comma-wsp? elliptical-arc-argument-sequence
// elliptical-arc-argument:
//     nonnegative-number comma-wsp? nonnegative-number comma-wsp?
//         number comma-wsp flag comma-wsp? flag comma-wsp? coordinate-pair
// coordinate-pair:
//     coordinate comma-wsp? coordinate
// coordinate:
//     number
// nonnegative-number:
//     integer-constant
//     | floating-point-constant
// number:
//     sign? integer-constant
//     | sign? floating-point-constant
// flag:
//     "0" | "1"
// comma-wsp:
//     (wsp+ comma? wsp*) | (comma wsp*)
// comma:
//     ","
// integer-constant:
//     digit-sequence
// floating-point-constant:
//     fractional-constant exponent?
//     | digit-sequence exponent
// fractional-constant:
//     digit-sequence? "." digit-sequence
//     | digit-sequence "."
// exponent:
//     ( "e" | "E" ) sign? digit-sequence
// sign:
//     "+" | "-"
// digit-sequence:
//     digit
//     | digit digit-sequence
// digit:
//     "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
// wsp:
//     (#x20 | #x9 | #xD | #xA)

type state struct {
	data     string
	index    int
	subPaths []*SubPath
	group    *SubPath
	currentX float64
	currentY float64
	relative bool
}

type SubPath struct {
	X, Y   float64
	DrawTo []*DrawTo
}

type Command string

const (
	ClosePath = "Z"
	LineTo    = "L"
	CurveTo   = "C"
)

type DrawTo struct {
	Command Command
	X, Y    float64
	X1, Y1  float64
	X2, Y2  float64
}

func (s *state) parse() error {
	// svg-path:
	//     wsp* moveto-drawto-command-groups? wsp*
	for {
		s.whitespace()

		// moveto-drawto-command-groups:
		//     moveto-drawto-command-group
		//     | moveto-drawto-command-group wsp* moveto-drawto-command-groups
		// moveto-drawto-command-group:
		//     moveto wsp* drawto-commands?

		c := s.peek()
		if c != 'M' && c != 'm' {
			break
		}

		err := s.parseMoveTo()
		if err != nil {
			return err
		}
		s.whitespace()
		err = s.parseDrawToCommands()
		if err != nil {
			return err
		}
	}

	s.whitespace()

	if s.index != len(s.data) {
		return fmt.Errorf("unparsed data: %q", s.data[s.index:])
	}

	return nil
}

// parseMoveTo parses one move to command
func (s *state) parseMoveTo() error {
	// moveto:
	//     ( "M" | "m" ) wsp* moveto-argument-sequence
	// moveto-argument-sequence:
	//     coordinate-pair
	//     | coordinate-pair comma-wsp? lineto-argument-sequence

	command := s.next()
	if command != 'M' && command != 'm' {
		return fmt.Errorf("expected \"M\" or \"m\", got %q", string(command))
	}
	s.relative = command == 'm'
	s.whitespace()

	var err error
	s.currentX, s.currentY, err = s.parseCoordinatePair()
	if err != nil {
		return err
	}

	// The move to command starts a new sub path group
	s.ensureSubPath()

	// The Move To can be followed directly by more coordinate pairs as implicit Line To sequences.
	// lineto-argument-sequence:
	//     coordinate-pair
	//     | coordinate-pair comma-wsp? lineto-argument-sequence
	for {
		savedIndex := s.index
		s.commaWhitespace()
		x, y, err := s.parseCoordinatePair()
		if err != nil {
			// backtrack.
			s.index = savedIndex
			break
		}
		if s.relative {
			x += s.currentX
			y += s.currentY
		}
		s.currentX = x
		s.currentY = y
		s.group.DrawTo = append(s.group.DrawTo,
			&DrawTo{Command: LineTo, X: x, Y: y})
	}

	return nil
}

// ensureSubPath starts a new sub path if there isn't already one.
func (s *state) ensureSubPath() {
	if s.group == nil {
		s.group = &SubPath{X: s.currentX, Y: s.currentY}
		s.subPaths = append(s.subPaths, s.group)
	}
}

// parseCoordinatePair parses "coordinate comma-wsp? coordinate"
func (s *state) parseCoordinatePair() (float64, float64, error) {
	// coordinate-pair:
	//     coordinate comma-wsp? coordinate
	// coordinate:
	//     number

	x, err := s.parseNumber()
	if err != nil {
		return 0, 0, err
	}
	s.commaWhitespace()
	y, err := s.parseNumber()
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

// parseNumber parses a number
func (s *state) parseNumber() (float64, error) {
	// number:
	//     sign? integer-constant
	//     | sign? floating-point-constant
	// sign:
	//     "+" | "-"
	c := s.peek()
	if c == '+' || c == '-' {
		s.next()
		n, err := s.parseNonNegativeNumber()
		if c == '-' {
			n = -n
		}
		return n, err
	}
	return s.parseNonNegativeNumber()
}

func (s *state) parseNonNegativeNumber() (float64, error) {
	// nonnegative-number:
	//     (digit-sequence | fractional-constant) exponent?
	// fractional-constant:
	//     digit-sequence? "." digit-sequence
	//     | digit-sequence "."
	// exponent:
	//     ( "e" | "E" ) sign? digit-sequence

	number := s.digitSequence()
	if number == "" {
		// Possible fractional constant starting with a decimal point
		c := s.next()
		if c != '.' {
			return 0, fmt.Errorf("expected a number, got %q", string(c))
		}
		number = "." + s.digitSequence()
		if number == "." {
			return 0, fmt.Errorf("expected a number, got only a \".\"")
		}
	} else {
		// Check for possible fractional constant
		c := s.peek()
		if c == '.' {
			s.next()
			number += "." + s.digitSequence()
		}
	}

	// Check for possible exponent
	c := s.peek()
	if c == 'E' || c == 'e' {
		s.next()
		sign := ""
		c = s.peek()
		if c == '+' || c == '-' {
			s.next()
			sign = string(c)
		}
		exponent := s.digitSequence()
		if exponent == "" {
			return 0, fmt.Errorf("expected an exponent, got %q", string(c))
		}
		number += "E" + sign + exponent
	}

	n, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *state) digitSequence() string {
	// digit-sequence:
	//     digit
	//     | digit digit-sequence
	// digit:
	//     "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
	var sequence []byte
	for {
		c := s.peek()
		if '0' <= c && c <= '9' {
			sequence = append(sequence, c)
			s.next()
		} else {
			break
		}
	}
	return string(sequence)
}

// parseDrawToCommands parses 0 or more Draw To commands.
func (s *state) parseDrawToCommands() error {
	// drawto-commands:
	//     drawto-command
	//     | drawto-command wsp* drawto-commands
	// drawto-command:
	//     closepath
	//     | lineto
	//     | horizontal-lineto
	//     | vertical-lineto
	//     | curveto
	//     | smooth-curveto
	//     | quadratic-bezier-curveto
	//     | smooth-quadratic-bezier-curveto
	//     | elliptical-arc

	first := true
	for {
		if !first {
			s.whitespace()
		}
		first = false

		var err error

		c := s.peek()
		switch c {
		case 'L', 'l':
			err = s.parseLineTo()
		case 'H', 'h':
			err = s.parseHorizontalLineTo()
		case 'V', 'v':
			err = s.parseVerticalLineTo()
		case 'C', 'c':
			err = s.parseCurveTo()
		case 'Z', 'z':
			err = s.parseClosePath()
		default:
			return nil
		}

		if err != nil {
			return err
		}
	}
}

func (s *state) parseClosePath() error {
	c := s.next()
	if c != 'Z' && c != 'z' {
		return fmt.Errorf("expecting \"Z\" or \"z\", got %q", string(c))
	}
	s.group.DrawTo = append(s.group.DrawTo,
		&DrawTo{Command: ClosePath, X: s.group.X, Y: s.group.Y})
	s.currentX = s.group.X
	s.currentY = s.group.Y
	s.group = nil
	return nil
}

func (s *state) parseLineTo() error {
	// lineto:
	//     ( "L" | "l" ) wsp* lineto-argument-sequence
	c := s.next()
	if c != 'L' && c != 'l' {
		return fmt.Errorf("expecting \"L\" or \"l\", got %q", string(c))
	}
	s.relative = c == 'l'

	s.whitespace()

	s.ensureSubPath()

	// lineto-argument-sequence:
	//     coordinate-pair
	//     | coordinate-pair comma-wsp? lineto-argument-sequence
	first := true
	for {
		oldIndex := s.index
		if !first {
			s.commaWhitespace()
		}

		x, y, err := s.parseCoordinatePair()
		if err != nil {
			if !first {
				s.index = oldIndex
				return nil
			}
			return err
		}

		if s.relative {
			x += s.currentX
			y += s.currentY
		}
		s.group.DrawTo = append(s.group.DrawTo,
			&DrawTo{Command: LineTo, X: x, Y: y})
		s.currentX = x
		s.currentY = y

		first = false
	}
}

func (s *state) parseHorizontalLineTo() error {
	// horizontal-lineto:
	//     ( "H" | "h" ) wsp* horizontal-lineto-argument-sequence
	c := s.next()
	if c != 'H' && c != 'h' {
		return fmt.Errorf("expecting \"H\" or \"h\", got %q", string(c))
	}
	s.relative = c == 'h'

	s.whitespace()

	s.ensureSubPath()

	// horizontal-lineto-argument-sequence:
	//     coordinate
	//     | coordinate comma-wsp? horizontal-lineto-argument-sequence
	first := true
	for {
		oldIndex := s.index
		if !first {
			s.commaWhitespace()
		}

		x, err := s.parseNumber()
		if err != nil {
			if !first {
				s.index = oldIndex
				return nil
			}
			return err
		}

		if s.relative {
			x += s.currentX
		}
		s.group.DrawTo = append(s.group.DrawTo,
			&DrawTo{Command: LineTo, X: x, Y: s.currentY})
		s.currentX = x

		first = false
	}
}

func (s *state) parseVerticalLineTo() error {
	// vertical-lineto:
	//     ( "V" | "v" ) wsp* vertical-lineto-argument-sequence
	c := s.next()
	if c != 'V' && c != 'v' {
		return fmt.Errorf("expecting \"V\" or \"v\", got %q", string(c))
	}
	s.relative = c == 'v'

	s.whitespace()

	s.ensureSubPath()

	// vertical-lineto-argument-sequence:
	//     coordinate
	//     | coordinate comma-wsp? vertical-lineto-argument-sequence
	first := true
	for {
		oldIndex := s.index
		if !first {
			s.commaWhitespace()
		}

		y, err := s.parseNumber()
		if err != nil {
			if !first {
				s.index = oldIndex
				return nil
			}
			return err
		}

		if s.relative {
			y += s.currentY
		}
		s.group.DrawTo = append(s.group.DrawTo,
			&DrawTo{Command: LineTo, X: s.currentX, Y: y})
		s.currentY = y

		first = false
	}
}

func (s *state) parseCurveTo() error {
	// curveto:
	//     ( "C" | "c" ) wsp* curveto-argument-sequence
	c := s.next()
	if c != 'C' && c != 'c' {
		return fmt.Errorf("expecting \"C\" or \"c\", got %q", string(c))
	}
	s.relative = c == 'c'

	s.whitespace()

	s.ensureSubPath()

	// curveto-argument-sequence:
	//     curveto-argument
	//     | curveto-argument comma-wsp? curveto-argument-sequence
	// curveto-argument:
	//     coordinate-pair comma-wsp? coordinate-pair comma-wsp? coordinate-pair
	first := true
	for {
		oldIndex := s.index
		if !first {
			s.commaWhitespace()
		}

		x1, y1, err := s.parseCoordinatePair()
		if err != nil {
			if !first {
				s.index = oldIndex
				return nil
			}
			return err
		}

		s.commaWhitespace()
		x2, y2, err := s.parseCoordinatePair()
		if err != nil {
			return err
		}

		s.commaWhitespace()
		x, y, err := s.parseCoordinatePair()
		if err != nil {
			return err
		}

		if s.relative {
			x1 += s.currentX
			y1 += s.currentY
			x2 += s.currentX
			y2 += s.currentY
			x += s.currentX
			y += s.currentY
		}
		s.group.DrawTo = append(s.group.DrawTo,
			&DrawTo{Command: CurveTo, X: x, Y: y, X1: x1, Y1: y1, X2: x2, Y2: y2})
		s.currentX = x
		s.currentY = y

		first = false
	}
}

// whitespace consumes "wsp*", and returns the number of bytes consumed
func (s *state) whitespace() int {
	count := 0
	for {
		switch s.peek() {
		case ' ', '\t', '\n', '\r':
			s.next()
			count++
		default:
			return count
		}
	}
}

// commaWhitespace consumes an optional "(wsp+ comma? wsp*) | (comma wsp*)",
// and returns true if something was consumed
func (s *state) commaWhitespace() bool {
	if s.peek() == ',' {
		s.next()
		s.whitespace()
		return true
	}

	consumed := s.whitespace()
	if consumed > 0 {
		if s.peek() == ',' {
			s.next()
		}
		s.whitespace()
		return true
	}

	return false
}

// requiredCommaWhitespace consumes "(wsp+ comma? wsp*) | (comma wsp*)"
func (s *state) requiredCommaWhitespace() error {
	consumed := s.commaWhitespace()
	if !consumed {
		return fmt.Errorf("expected comma or whitespace, got %q", string(s.peek()))
	}
	return nil
}

// peek returns the next byte without consuming it, or 0 if at the end of stream
func (s *state) peek() byte {
	if s.index < len(s.data) {
		return s.data[s.index]
	}
	return 0
}

// next consumes and returns the next byte, or 0 if at the end of stream
func (s *state) next() byte {
	if s.index < len(s.data) {
		i := s.index
		s.index++
		return s.data[i]
	}
	return 0
}

// Parse parses a path string
func Parse(path string) ([]*SubPath, error) {
	s := &state{
		data:  path,
		index: 0,
	}
	err := s.parse()
	return s.subPaths, err
}

type Function struct {
	Name string
	Args []float64
}

func (s *state) parseFunctions() ([]*Function, error) {
	var functions []*Function
	// (wsp* identifier wsp* "(" wsp* number (comma-wsp number)* wsp* ")" wsp*)*
	for {
		function := &Function{}
		functions = append(functions, function)

		// identifier
		s.whitespace()
		c := s.next()
		if !(('a' <= c && c <= 'z') || ('A' <= c && c <= 'z')) {
			return functions, fmt.Errorf("identifier must start with a letter, got %q", string(c))
		}
		function.Name += string(c)
		for {
			c := s.peek()
			if ('a' <= c && c <= 'z') || ('A' <= c && c <= 'z') ||
				('0' <= c && c <= '9') || (c == '_') || (c == '-') {
				function.Name += string(s.next())
			} else {
				break
			}
		}

		// Open parenthesis
		s.whitespace()
		c = s.next()
		if c != '(' {
			return functions, fmt.Errorf("expected \"(\", got %q", string(c))
		}

		// First argument (optional)
		s.whitespace()
		oldIndex := s.index
		n, err := s.parseNumber()
		if err != nil {
			s.index = oldIndex
		} else {
			function.Args = append(function.Args, n)
			// Remaining arguments
			for {
				oldIndex = s.index
				s.commaWhitespace()
				n, err = s.parseNumber()
				if err != nil {
					s.index = oldIndex
					break
				}
				function.Args = append(function.Args, n)
			}
		}

		// Close parenthesis
		s.whitespace()
		c = s.next()
		if c != ')' {
			return functions, fmt.Errorf("expected \")\", got %q", string(c))
		}
		s.whitespace()

		if s.peek() == 0 {
			return functions, nil
		}
	}
}

func ParseFunctions(functions string) ([]*Function, error) {
	s := &state{
		data:  functions,
		index: 0,
	}
	return s.parseFunctions()
}

func ToString(groups []*SubPath) string {
	var buf strings.Builder

	// Note: this function runs a simple serialization. It does not try to optimize the path string.

	formatNumber := func(n float64) string {
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	for i, group := range groups {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString("M " + formatNumber(group.X) + " " + formatNumber(group.Y))
		for _, drawTo := range group.DrawTo {
			switch drawTo.Command {
			case LineTo:
				buf.WriteString(" L " + formatNumber(drawTo.X) + " " + formatNumber(drawTo.Y))
			case CurveTo:
				buf.WriteString(" C " +
					formatNumber(drawTo.X1) + " " + formatNumber(drawTo.Y1) + " " +
					formatNumber(drawTo.X2) + " " + formatNumber(drawTo.Y2) + " " +
					formatNumber(drawTo.X) + " " + formatNumber(drawTo.Y))
			case ClosePath:
				buf.WriteString(" Z")
			}
		}
	}

	return buf.String()
}
