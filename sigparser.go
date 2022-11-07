package sigparser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// ParseSignature parses the function, constructor, fallback, receive, event or
// error signature. The syntax is similar to that of Solidity, but it is less
// strict. The argument names are always optional, and the return keyword can
// be omitted.
//
// Signature can be prepended with the keyword describing the signature kind.
// If the kind is not specified, it is assumed to be a function.
// The following kinds are supported:
//
//  - function
//  - constructor
//  - fallback
//  - receive
//  - event
//  - error
//
// The following examples are valid signatures:
//
//  - function foo(uint256 memory a, uint256 memory b) internal returns (uint256)
//  - function foo(uint256 a, uint256 b) (uint256)
//  - foo(uint256,uint256)(uint256)
//  - constructor(uint256 a, uint256 b)
//  - fallback(bytes memory a) returns (bytes memory)
//  - receive()
//  - event Foo(uint256 a, uint256 b)
//  - error Foo(uint256 a, uint256 b)
func ParseSignature(signature string) (Signature, error) {
	p := &parser{in: []byte(signature)}
	p.parseWhitespace()
	sig, err := p.parseSignature()
	if err != nil {
		return Signature{}, err
	}
	p.parseWhitespace()
	if p.hasNext() {
		return Signature{}, fmt.Errorf(`unexpected token %q at the end of the signature`, p.peek())
	}
	return sig, nil
}

// ParseParameter parses the type and returns its definition. The syntax is the same
// as in Solidity.
func ParseParameter(signature string) (Parameter, error) {
	p := &parser{in: []byte(signature)}
	p.parseWhitespace()
	typ, err := p.parseParameter()
	if err != nil {
		return Parameter{}, err
	}
	p.parseWhitespace()
	if p.hasNext() {
		return Parameter{}, fmt.Errorf(`unexpected token %q at the end of the type definition`, p.peek())
	}
	return typ, nil
}

type SignatureKind int8

const (
	FunctionKind SignatureKind = iota
	ConstructorKind
	FallbackKind
	ReceiveKind
	EventKind
	ErrorKind
)

type DataLocation int8

const (
	UnspecifiedLocation DataLocation = iota
	Storage
	CallData
	Memory
)

// Signature represents a signature of a function, constructor, fallback,
// receive, event or error.
type Signature struct {
	// Kind is the kind of the signature.
	Kind SignatureKind
	// Name is the name of the function, event or error. It should be empty for
	// fallback, receive and constructor kinds.
	Name string
	// Inputs is the list of input argument types.
	Inputs []Parameter
	// Outputs is the list of output value types.
	Outputs []Parameter
	// Modifiers is the list of function modifiers.
	Modifiers []string
}

// Parameter represents an argument or return value.
type Parameter struct {
	// Name is an optional name of the argument or return value.
	Name string
	// Type is the name of the type, e.g. uint256, address, etc.
	Type string
	// Tuple is a list tuple elements. It must be empty for non-tuple types.
	Tuple []Parameter
	// Arrays is the list of array dimensions, where each dimension is the
	// maximum length of the array. If the length is -1, the array is
	// unbounded. If the Arrays is empty, the argument is not an array.
	Arrays []int
	// Indexed indicates whether the argument is indexed. It should be false
	// for types other than event.
	Indexed bool
	// DataLocation indicates the data location of the argument. It should be
	// UnspecifiedLocation for types other than function and constructor.
	DataLocation DataLocation
}

// String returns the string representation of the signature.
func (s Signature) String() string {
	var buf strings.Builder
	switch s.Kind {
	case FunctionKind:
		buf.WriteString("function ")
		buf.WriteString(s.Name)
	case ConstructorKind:
		buf.WriteString("constructor")
	case FallbackKind:
		buf.WriteString("fallback")
	case ReceiveKind:
		buf.WriteString("receive")
	case EventKind:
		buf.WriteString("event ")
		buf.WriteString(s.Name)
	case ErrorKind:
		buf.WriteString("error ")
		buf.WriteString(s.Name)
	default:
		buf.WriteString(s.Name)
	}
	buf.WriteByte('(')
	for i, c := range s.Inputs {
		buf.WriteString(c.String())
		if i < len(s.Inputs)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteByte(')')
	if len(s.Outputs) > 0 {
		buf.WriteByte('(')
		for i, c := range s.Outputs {
			buf.WriteString(c.String())
			if i < len(s.Inputs)-1 {
				buf.WriteString(", ")
			}
		}
		buf.WriteByte(')')
	}
	return buf.String()
}

// String returns the string representation of the type.
func (p Parameter) String() string {
	var buf strings.Builder
	if len(p.Type) > 0 {
		buf.WriteString(p.Type)
	} else {
		buf.WriteByte('(')
		for i, c := range p.Tuple {
			buf.WriteString(c.String())
			if i < len(p.Tuple)-1 {
				buf.WriteString(", ")
			}
		}
		buf.WriteByte(')')
	}
	for _, n := range p.Arrays {
		if n == -1 {
			buf.WriteString("[]")
		} else {
			buf.WriteByte('[')
			buf.WriteString(strconv.Itoa(n))
			buf.WriteByte(']')
		}
	}
	if p.Indexed {
		buf.WriteByte(' ')
		buf.WriteString("indexed")
	}
	switch p.DataLocation {
	case Storage:
		buf.WriteByte(' ')
		buf.WriteString("storage")
	case CallData:
		buf.WriteByte(' ')
		buf.WriteString("calldata")
	case Memory:
		buf.WriteByte(' ')
		buf.WriteString("memory")
	}
	if len(p.Name) > 0 {
		buf.WriteByte(' ')
		buf.WriteString(p.Name)
	}
	return buf.String()
}

func (p Parameter) copy() Parameter {
	cpy := Parameter{
		Name:         p.Name,
		Type:         p.Type,
		Indexed:      p.Indexed,
		DataLocation: p.DataLocation,
	}
	cpy.Arrays = make([]int, len(p.Arrays))
	cpy.Tuple = make([]Parameter, len(p.Tuple))
	copy(cpy.Arrays, p.Arrays)
	for i := range p.Tuple {
		cpy.Tuple[i] = p.Tuple[i].copy()
	}
	return cpy
}

type parser struct {
	in  []byte
	pos int
}

func (p *parser) parseSignature() (Signature, error) {
	var (
		err error
		sig Signature
	)
	// Parse signature type.
	sig.Kind = p.parseSignatureKind()
	// Parse name.
	p.parseWhitespace()
	sig.Name = string(p.parseName())
	// Parse inputs.
	p.parseWhitespace()
	if sig.Inputs, err = p.parseInputs(); err != nil {
		return Signature{}, err
	}
	// Parse modifiers.
	p.parseWhitespace()
	sig.Modifiers = p.parseModifiers()
	// Parse outputs.
	p.parseWhitespace()
	if sig.Outputs, err = p.parseOutputs(); err != nil {
		return Signature{}, err
	}
	// Validate signature based on its kind.
	switch sig.Kind {
	case ConstructorKind:
		if len(sig.Name) > 0 {
			return Signature{}, fmt.Errorf(`unexpected constructor name %q`, sig.Name)
		}
		if len(sig.Modifiers) > 0 {
			return Signature{}, fmt.Errorf(`unexpected constructor modifiers`)
		}
		if len(sig.Outputs) > 0 {
			return Signature{}, fmt.Errorf(`unexpected constructor outputs`)
		}
	case FallbackKind:
		if len(sig.Name) > 0 {
			return Signature{}, fmt.Errorf(`unexpected fallback name %q`, sig.Name)
		}
		if len(sig.Inputs) == 1 && sig.Inputs[0].String() != "bytes" {
			return Signature{}, fmt.Errorf(`unexpected fallback input type %q`, sig.Inputs[0].String())
		}
		if len(sig.Inputs) > 1 {
			return Signature{}, fmt.Errorf(`unexpected fallback inputs`)
		}
		if len(sig.Outputs) == 1 && sig.Outputs[0].String() != "bytes" {
			return Signature{}, fmt.Errorf(`unexpected fallback output type %q`, sig.Outputs[0].String())
		}
		if len(sig.Outputs) > 1 {
			return Signature{}, fmt.Errorf(`unexpected fallback outputs`)
		}
	case ReceiveKind:
		if len(sig.Name) > 0 {
			return Signature{}, fmt.Errorf(`unexpected receive name %q`, sig.Name)
		}
		if len(sig.Inputs) > 0 {
			return Signature{}, fmt.Errorf(`unexpected receive inputs`)
		}
		if len(sig.Outputs) > 0 {
			return Signature{}, fmt.Errorf(`unexpected receive outputs`)
		}
	case EventKind:
		if len(sig.Inputs) == 0 {
			return Signature{}, fmt.Errorf(`event must have inputs`)
		}
		if len(sig.Outputs) > 0 {
			return Signature{}, fmt.Errorf(`unexpected event outputs`)
		}
		if len(sig.Modifiers) > 0 {
			return Signature{}, fmt.Errorf(`unexpected event modifiers`)
		}
		for _, input := range sig.Inputs {
			if input.DataLocation != UnspecifiedLocation {
				return Signature{}, fmt.Errorf(`unexpected event input data location`)
			}
		}
	case ErrorKind:
		if len(sig.Inputs) == 0 {
			return Signature{}, fmt.Errorf(`error must have inputs`)
		}
		if len(sig.Outputs) > 0 {
			return Signature{}, fmt.Errorf(`unexpected error outputs`)
		}
		if len(sig.Modifiers) > 0 {
			return Signature{}, fmt.Errorf(`unexpected error modifiers`)
		}
		for _, input := range sig.Inputs {
			if input.DataLocation != UnspecifiedLocation {
				return Signature{}, fmt.Errorf(`unexpected error input data location`)
			}
		}
	}
	if sig.Kind != EventKind {
		for _, input := range sig.Inputs {
			if input.Indexed {
				return Signature{}, fmt.Errorf(`unexpected indexed flag`)
			}
		}
	}
	for _, output := range sig.Outputs {
		if output.Indexed {
			return Signature{}, fmt.Errorf(`unexpected indexed flag`)
		}
	}
	return sig, nil
}

// parseSignatureKind parses signature kind.
func (p *parser) parseSignatureKind() SignatureKind {
	switch {
	case p.readBytes([]byte("function")):
		return FunctionKind
	case p.readBytes([]byte("constructor")):
		return ConstructorKind
	case p.readBytes([]byte("fallback")):
		return FallbackKind
	case p.readBytes([]byte("receive")):
		return ReceiveKind
	case p.readBytes([]byte("event")):
		return EventKind
	case p.readBytes([]byte("error")):
		return ErrorKind
	}
	return FunctionKind
}

func (p *parser) parseInputs() ([]Parameter, error) {
	if p.peekByte('(') {
		// Parameter list have exactly the same syntax as composite type, except
		// that it cannot have arrays.
		args, err := p.parseCompositeType()
		if err != nil {
			return nil, err
		}
		if len(args.Arrays) > 0 {
			return nil, fmt.Errorf(`unexpected array declaration`)
		}
		return args.Tuple, nil
	}
	return nil, nil
}

func (p *parser) parseOutputs() ([]Parameter, error) {
	returnsKeyword := false
	p.parseWhitespace()
	if p.readBytes([]byte("returns")) { // optional "returns" keyword
		returnsKeyword = true
		p.parseWhitespace()
	}
	if returnsKeyword && !p.peekByte('(') {
		return nil, fmt.Errorf(`unexpected token %q, expected '(' after 'returns' keyword`, p.peek())
	}
	if p.peekByte('(') {
		// Return types list have exactly the same syntax as composite type,
		// except that it cannot have arrays.
		args, err := p.parseCompositeType()
		if err != nil {
			return nil, err
		}
		if len(args.Arrays) > 0 {
			return nil, fmt.Errorf(`unexpected array declaration`)
		}
		return args.Tuple, nil
	}
	return nil, nil
}

// parseModifiers parses method modifiers.
func (p *parser) parseModifiers() []string {
	var mods []string
	for {
		if !p.hasNext() || p.peekByte('(') || p.peekBytes([]byte("returns")) {
			break
		}
		mod := string(p.parseName())
		if len(mod) == 0 {
			break
		}
		mods = append(mods, mod)
		if !p.hasNext() || !isWhitespace(p.peek()) {
			break
		}
		p.parseWhitespace()
	}
	return mods
}

// parseParameter parses a single argument or return value.
func (p *parser) parseParameter() (Parameter, error) {
	var (
		err error
		arg Parameter
	)
	// Parameter can be either a composite type or an elementary type. All
	// elementary types start with a letter and composite types start with
	// a parenthesis. We can use this fact to distinguish between the two.
	switch {
	case isAlpha(p.peek()) || isIdentifierSymbol(p.peek()):
		arg, err = p.parseElementaryType()
		if err != nil {
			return Parameter{}, err
		}
	case p.peekByte('('):
		arg, err = p.parseCompositeType()
		if err != nil {
			return Parameter{}, err
		}
	default:
		return Parameter{}, fmt.Errorf(`unexpected token %q, type expected`, p.peek())
	}
	// Parse data location, indexed flag and name.
	if p.hasNext() && isWhitespace(p.peek()) {
		p.parseWhitespace()
		has := false
		switch {
		case p.readBytes([]byte("indexed")):
			arg.Indexed = true
			has = true
		case p.readBytes([]byte("storage")):
			arg.DataLocation = Storage
			has = true
		case p.readBytes([]byte("memory")):
			arg.DataLocation = Memory
			has = true
		case p.readBytes([]byte("calldata")):
			arg.DataLocation = CallData
			has = true
		}
		if has {
			if p.hasNext() && isWhitespace(p.peek()) {
				p.parseWhitespace()
				arg.Name = string(p.parseName())
			}
		} else {
			arg.Name = string(p.parseName())
		}
	}
	return arg, err
}

// parseCompositeType parses composite type argument along with optional array
// declarations.
func (p *parser) parseCompositeType() (Parameter, error) {
	if !p.readByte('(') {
		return Parameter{}, fmt.Errorf(`unexpected token %q, '(' expected`, p.peek())
	}
	if !p.hasNext() {
		return Parameter{}, fmt.Errorf(`unexpected end of input, composite type expected`)
	}
	var arg Parameter
	p.parseWhitespace()
	// Parse components, but only if composite type is not empty.
	if !p.readByte(')') {
		for {
			p.parseWhitespace()
			comp, err := p.parseParameter()
			if err != nil {
				return Parameter{}, err
			}
			arg.Tuple = append(arg.Tuple, comp)
			p.parseWhitespace()
			if p.readByte(',') {
				continue
			}
			if p.readByte(')') {
				break
			}
			return Parameter{}, fmt.Errorf(`unexpected token %q, ',' or ')' expected`, p.peek())
		}
	}
	// Parse array declarations, if any.
	if p.peekByte('[') {
		arr, err := p.parseArray()
		if err != nil {
			return Parameter{}, err
		}
		arg.Arrays = arr
	}
	return arg, nil
}

// parseElementaryType parses elementary type along with optional array
// declaration.
func (p *parser) parseElementaryType() (Parameter, error) {
	var arg Parameter
	// Parse type name.
	pos := p.pos
	for p.hasNext() {
		b := p.peek()
		if pos == p.pos && (isAlpha(b) || isIdentifierSymbol(b)) {
			p.read()
			continue
		}
		if pos != p.pos && (isAlpha(b) || isDigit(b) || isIdentifierSymbol(b)) {
			p.read()
			continue
		}
		break
	}
	arg.Type = string(p.in[pos:p.pos])
	// Parse array declaration, if any.
	if p.peekByte('[') {
		arr, err := p.parseArray()
		if err != nil {
			return Parameter{}, err
		}
		arg.Arrays = arr
	}
	return arg, nil
}

// parseWhitespace parses whitespaces.
func (p *parser) parseWhitespace() {
	for p.hasNext() {
		if !isWhitespace(p.peek()) {
			break
		}
		p.read()
	}
	return
}

// parseName parses name of the argument or method and returns it.
func (p *parser) parseName() []byte {
	pos := p.pos
	for p.hasNext() {
		b := p.peek()
		if pos == p.pos && (isAlpha(b) || isIdentifierSymbol(b)) {
			p.read()
			continue
		}
		if pos != p.pos && (isAlpha(b) || isDigit(b) || isIdentifierSymbol(b)) {
			p.read()
			continue
		}
		break
	}
	return p.in[pos:p.pos]
}

// parseNumber parses decimal number from the input. The parsed number is
// returned as integer. If there was no number to parse, the false is returned
// as second value.
func (p *parser) parseNumber() (int, bool) {
	pos := p.pos
	for p.hasNext() {
		if isDigit(p.peek()) {
			p.read()
		}
		break
	}
	if pos == p.pos {
		return 0, false
	}
	n, err := strconv.ParseUint(string(p.in[pos:p.pos]), 10, 0)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

// parseArray parses array part of the type declaration. It returns a slice
// with array dimensions. The -1 value represents an unspecified array size.
func (p *parser) parseArray() ([]int, error) {
	var arr []int
	for p.hasNext() {
		if p.readByte('[') {
			n, ok := p.parseNumber()
			if ok {
				arr = append(arr, n)
			} else {
				arr = append(arr, -1)
			}
			if !p.readByte(']') {
				return nil, fmt.Errorf(`unexpected token %q, ']' expected`, p.peek())
			}
			continue
		}
		break
	}
	return arr, nil
}

// hasNext returns true if there are more bytes to read.
func (p *parser) hasNext() bool {
	return p.pos < len(p.in)
}

// peek returns the next byte to read.
func (p *parser) peek() byte {
	return p.in[p.pos]
}

// nextByte returns the next byte and advances the position.
func (p *parser) read() byte {
	p.pos++
	return p.in[p.pos-1]
}

// peekByte returns true if the next byte is equal to b.
func (p *parser) peekByte(b byte) bool {
	if p.pos >= len(p.in) {
		return false
	}
	if p.in[p.pos] == b {
		return true
	}
	return false
}

// peekBytes returns true if the next bytes are equal to b.
func (p *parser) peekBytes(b []byte) bool {
	if p.pos+len(b) > len(p.in) {
		return false
	}
	if bytes.HasPrefix(p.in[p.pos:], b) {
		return true
	}
	return false
}

// readByte returns true if the next byte is equal to b and advances the
// position.
func (p *parser) readByte(b byte) bool {
	if p.peekByte(b) {
		p.pos++
		return true
	}
	return false
}

// readBytes returns true if the next bytes are equal to b and advances the
// position.
func (p *parser) readBytes(b []byte) bool {
	if p.peekBytes(b) {
		p.pos += len(b)
		return true
	}
	return false
}

// isDigit returns true if b is a digit.
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isAlpha returns true if b is an alphabetic character
func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isWhitespace returns true if b is a whitespace character.
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n'
}

// isIdentifierSymbol returns true if b is a valid identifier symbol.
func isIdentifierSymbol(c byte) bool {
	return c == '$' || c == '_'
}
