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
// Tuples are represented as a list of types enclosed in parentheses, optionally
// prefixed with the "tuple" keyword.
//
// Signature may be prepended with the keyword describing the signature kind.
// The following kinds are supported:
//
//   - function
//   - constructor
//   - fallback
//   - receive
//   - event
//   - error
//
// The following examples are valid signatures:
//
//   - function foo(uint256 memory a, tuple(uint256 b1, uint256 b2) memory b) internal returns (uint256)
//   - function foo(uint256 a, (uint256 b1, uint256 b2) b) (uint256)
//   - foo(uint256,(uint256,uint256))(uint256)
//   - constructor(uint256 a, uint256 b)
//   - fallback(bytes memory a) returns (bytes memory)
//   - receive()
//   - event Foo(uint256 a, uint256 b)
//   - error Foo(uint256 a, uint256 b)
//
// Signatures that are syntactically correct, but semantically invalid are
// rejected by the parser.
func ParseSignature(signature string) (Signature, error) {
	return ParseSignatureAs(UnknownKind, signature)
}

// ParseSignatureAs works like ParseSignature, but it allows to specify the
// signature kind.
//
// The kind can be UnknownKind, in which case the kind is inferred from the
// signature.
func ParseSignatureAs(kind SignatureKind, signature string) (Signature, error) {
	p := &parser{in: []byte(signature)}
	p.parseWhitespace()
	sig, err := p.parseSignature(kind)
	if err != nil {
		return Signature{}, err
	}
	if !p.onlyWhitespaceOrDelimiterLeft() {
		return Signature{}, fmt.Errorf(`unexpected character %q at the end of the signature`, p.peek())
	}
	return sig, nil
}

// ParseParameter parses the single parameter. The syntax is same as for
// parameters in the ParseSignature function.
func ParseParameter(signature string) (Parameter, error) {
	p := &parser{in: []byte(signature)}
	p.parseWhitespace()
	typ, err := p.parseParameter()
	if err != nil {
		return Parameter{}, err
	}
	if !p.onlyWhitespaceOrDelimiterLeft() {
		return Parameter{}, fmt.Errorf(`unexpected character %q at the end of the parameter`, p.peek())
	}
	return typ, nil
}

// ParseStruct parses the struct definition.
//
// It returns a structure as a tuple type where the tuple name is the struct
// name and the tuple elements are the struct fields.
func ParseStruct(definition string) (Parameter, error) {
	p := &parser{in: []byte(definition)}
	p.parseWhitespace()
	str, err := p.parseStruct()
	if err != nil {
		return Parameter{}, err
	}
	if !p.onlyWhitespaceOrDelimiterLeft() {
		return Parameter{}, fmt.Errorf(`unexpected character %q at the end of the struct`, p.peek())
	}
	return str, nil
}

// Kind returns the kind of the input string.
//
// This function helps determine which parser should be used to parse the
// input.
//
// Note that some inputs are ambiguous. They could be interpreted either
// as a type or a function signature. For example, "foo" could be a type or a
// function name. Similarly, "function foo" could be interpreted as a function
// signature or a parameter "foo" with the type "function".
//
// To avoid ambiguity, always add an empty parameter list to function
// signatures.
func Kind(input string) (k InputKind) {
	p := &parser{in: []byte(input)}
	p.parseWhitespace()
	pos := p.pos
	if param, err := p.parseParameter(); err == nil && p.onlyWhitespaceOrDelimiterLeft() {
		if len(param.Arrays) > 0 {
			return ArrayInput
		}
		if len(param.Tuple) > 0 {
			return TupleInput
		}
		return TypeInput
	}
	p.pos = pos
	if sig, err := p.parseSignature(UnknownKind); err == nil && p.onlyWhitespaceOrDelimiterLeft() {
		switch sig.Kind {
		case FunctionKind, UnknownKind:
			return FunctionSignatureInput
		case ConstructorKind:
			return ConstructorSignatureInput
		case FallbackKind:
			return FallbackSignatureInput
		case ReceiveKind:
			return ReceiveSignatureInput
		case EventKind:
			return EventSignatureInput
		case ErrorKind:
			return ErrorSignatureInput
		}
	}
	p.pos = pos
	if _, err := p.parseStruct(); err == nil && p.onlyWhitespaceOrDelimiterLeft() {
		return StructDefinitionInput
	}
	return InvalidInput
}

// InputKind is the kind of the input string returned by the Kind function.
type InputKind int8

const (
	InvalidInput InputKind = iota
	TypeInput
	ArrayInput
	TupleInput
	StructDefinitionInput
	FunctionSignatureInput
	ConstructorSignatureInput
	FallbackSignatureInput
	ReceiveSignatureInput
	EventSignatureInput
	ErrorSignatureInput
)

func (k InputKind) String() string {
	switch k {
	case InvalidInput:
		return "invalid"
	case TypeInput:
		return "type"
	case ArrayInput:
		return "array"
	case TupleInput:
		return "tuple"
	case StructDefinitionInput:
		return "struct"
	case FunctionSignatureInput:
		return "function"
	case ConstructorSignatureInput:
		return "constructor"
	case FallbackSignatureInput:
		return "fallback"
	case ReceiveSignatureInput:
		return "receive"
	case EventSignatureInput:
		return "event"
	case ErrorSignatureInput:
		return "error"
	default:
		return "unknown"
	}
}

// IsSignature returns true if the input is a signature for any type of function.
//
// It can be parsed using ParseSignature function.
func (k InputKind) IsSignature() bool {
	switch k {
	case FunctionSignatureInput, ConstructorSignatureInput, FallbackSignatureInput, ReceiveSignatureInput, EventSignatureInput, ErrorSignatureInput:
		return true
	default:
		return false
	}
}

// IsParameter returns true if the input is a parameter.
//
// It can be parsed using ParseParameter function.
func (k InputKind) IsParameter() bool {
	switch k {
	case TypeInput, ArrayInput, TupleInput:
		return true
	default:
		return false
	}
}

// IsStruct returns true if the input is a struct definition.
//
// It can be parsed using ParseStruct function.
func (k InputKind) IsStruct() bool {
	return k == StructDefinitionInput
}

// SignatureKind is the kind of the signature, like function, constructor,
// fallback, etc.
type SignatureKind int8

const (
	UnknownKind SignatureKind = iota
	FunctionKind
	ConstructorKind
	FallbackKind
	ReceiveKind
	EventKind
	ErrorKind
)

func (s SignatureKind) String() string {
	switch s {
	case FunctionKind:
		return "function"
	case ConstructorKind:
		return "constructor"
	case FallbackKind:
		return "fallback"
	case ReceiveKind:
		return "receive"
	case EventKind:
		return "event"
	case ErrorKind:
		return "error"
	default:
		return "unknown"
	}
}

// DataLocation is the data location of the parameter, like storage, memory
// or calldata.
type DataLocation int8

const (
	UnspecifiedLocation DataLocation = iota
	Storage
	CallData
	Memory
)

func (d DataLocation) String() string {
	switch d {
	case Storage:
		return "storage"
	case CallData:
		return "calldata"
	case Memory:
		return "memory"
	default:
		return ""
	}
}

// Signature represents a signature of a function, constructor, fallback,
// receive, event or error.
type Signature struct {
	// Kind is the kind of the signature.
	Kind SignatureKind

	// Name is the name of the function, event or error. It should be empty for
	// fallback, receive and constructor kinds.
	Name string

	// Inputs is the list of input parameters.
	Inputs []Parameter

	// Outputs is the list of output parameters.
	Outputs []Parameter

	// Modifiers is the list of function modifiers.
	Modifiers []string
}

// Parameter represents an argument or return value.
type Parameter struct {
	// Name is an optional name of the argument or return value.
	Name string

	// Type is the parameter type, like uint256, bytes32, etc. It must
	// be empty for tuples.
	Type string

	// Tuple is a list tuple elements. It must be empty for non-tuple types.
	Tuple []Parameter

	// Arrays is the list of array dimensions, where each dimension is the
	// maximum length of the array. If the length is -1, the array is
	// unbounded. If the Arrays is empty, the argument is not an array.
	Arrays []int

	// Indexed indicates whether the argument is indexed. It must be false
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
	if len(s.Modifiers) > 0 {
		buf.WriteString(" ")
		for i, m := range s.Modifiers {
			buf.WriteString(m)
			if i < len(s.Modifiers)-1 {
				buf.WriteString(" ")
			}
		}
	}
	if len(s.Outputs) > 0 {
		buf.WriteString(" returns (")
		for i, c := range s.Outputs {
			buf.WriteString(c.String())
			if i < len(s.Outputs)-1 {
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

type parser struct {
	in  []byte
	pos int
}

func (p *parser) parseSignature(kind SignatureKind) (Signature, error) {
	var (
		err error
		sig Signature
	)
	// Parse signature type.
	sig.Kind = p.parseSignatureKind()
	if sig.Kind == UnknownKind {
		sig.Kind = kind
	}
	if kind != UnknownKind && sig.Kind != kind {
		return sig, fmt.Errorf("invalid signature kind: %s", sig.Kind)
	}
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
		validInOut := len(sig.Inputs) == 1 && sig.Inputs[0].Type == "bytes" && len(sig.Outputs) == 1 && sig.Outputs[0].Type == "bytes"
		if !validInOut && len(sig.Inputs) > 0 {
			return Signature{}, fmt.Errorf(`unexpected fallback inputs`)
		}
		if !validInOut && len(sig.Outputs) > 0 {
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
		if !(len(sig.Modifiers) == 0 || (len(sig.Modifiers) == 1 && sig.Modifiers[0] == "anonymous")) {
			return Signature{}, fmt.Errorf(`unexpected event modifiers`)
		}
		for _, input := range sig.Inputs {
			if input.DataLocation != UnspecifiedLocation {
				return Signature{}, fmt.Errorf(`unexpected event input data location`)
			}
		}
	case ErrorKind:
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
	if sig.Kind != UnknownKind && sig.Kind != EventKind {
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
	return UnknownKind
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
		if !p.hasNext() {
			return nil, fmt.Errorf(`unexpected end of input, expected '(' after 'returns' keyword`)
		}
		return nil, fmt.Errorf(`unexpected character %q, expected '(' after 'returns' keyword`, p.peek())
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

func (p *parser) parseStruct() (Parameter, error) {
	s := Parameter{}
	// Parse struct keyword.
	if !p.readBytes([]byte("struct")) {
		if !p.hasNext() {
			return Parameter{}, fmt.Errorf(`unexpected end of input, 'struct' keyword expected`)
		}
		return Parameter{}, fmt.Errorf(`unexpected character %q, 'struct' keyword expected`, p.peek())
	}
	p.parseWhitespace()
	// Parse struct name.
	s.Name = string(p.parseName())
	p.parseWhitespace()
	// Parse struct fields.
	if !p.readByte('{') {
		if !p.hasNext() {
			return Parameter{}, fmt.Errorf(`unexpected end of input, '{' expected`)
		}
		return Parameter{}, fmt.Errorf(`unexpected character %q, '{' expected`, p.peek())
	}
	for {
		p.parseWhitespace()
		if p.readByte('}') {
			break
		}
		// Parse field type.
		field, err := p.parseElementaryType()
		if err != nil {
			return Parameter{}, err
		}
		p.parseWhitespace()
		// Parse field name.
		field.Name = string(p.parseName())
		if len(field.Name) == 0 {
			return Parameter{}, fmt.Errorf(`unexpected end of input, field name expected`)
		}
		s.Tuple = append(s.Tuple, field)
		p.parseWhitespace()
		// Parse field separator.
		if !p.readByte(';') {
			if !p.hasNext() {
				return Parameter{}, fmt.Errorf(`unexpected end of input, ';' expected`)
			}
			return Parameter{}, fmt.Errorf(`unexpected character %q, ';' expected`, p.peek())
		}
	}
	return s, nil
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
	// Parameter can be either a composite type or an elementary type.
	// The composite types start with a parenthesis, or a "tuple" keyword
	// followed by a parenthesis. All elementary types start with a letter.
	// We can use this fact to distinguish between the two.
	switch {
	case !p.hasNext():
		return Parameter{}, fmt.Errorf(`unexpected end of input, type expected`)
	case p.peekByte('(') || p.peekBytes([]byte("tuple(")):
		arg, err = p.parseCompositeType()
		if err != nil {
			return Parameter{}, err
		}
	case isAlpha(p.peek()) || isIdentifierSymbol(p.peek()):
		arg, err = p.parseElementaryType()
		if err != nil {
			return Parameter{}, err
		}
	default:
		return Parameter{}, fmt.Errorf(`unexpected character %q, type expected`, p.peek())
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
	if !p.readByte('(') && !p.readBytes([]byte("tuple(")) {
		if !p.hasNext() {
			return Parameter{}, fmt.Errorf(`unexpected end of input, 'tuple(' or '(' expected`)
		}
		return Parameter{}, fmt.Errorf(`unexpected character %q, 'tuple(' or '(' expected`, p.peek())
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
			if !p.hasNext() {
				return Parameter{}, fmt.Errorf(`unexpected end of input, ',' or ')' expected`)
			}
			return Parameter{}, fmt.Errorf(`unexpected character %q, ',' or ')' expected`, p.peek())
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
func (p *parser) parseNumber() (int, bool, error) {
	pos := p.pos
	for p.hasNext() {
		if !isDigit(p.peek()) {
			break
		}
		p.read()
	}
	if pos == p.pos {
		return 0, false, nil
	}
	n, err := strconv.ParseInt(string(p.in[pos:p.pos]), 10, 0)
	if err != nil {
		return 0, false, err
	}
	return int(n), true, nil
}

// parseArray parses array part of the type declaration. It returns a slice
// with array dimensions. The -1 value represents an unspecified array size.
func (p *parser) parseArray() ([]int, error) {
	var arr []int
	for p.hasNext() {
		if p.readByte('[') {
			n, ok, err := p.parseNumber()
			if err != nil {
				return nil, fmt.Errorf(`invalid array size: %v`, err)
			}
			if ok && n <= 0 {
				return nil, fmt.Errorf(`invalid array size: %d`, n)
			}
			if ok {
				arr = append(arr, n)
			} else {
				arr = append(arr, -1)
			}
			if !p.hasNext() {
				return nil, fmt.Errorf(`unexpected end of input, ']' expected`)
			}
			if !p.readByte(']') {
				return nil, fmt.Errorf(`unexpected character %q, ']' expected`, p.peek())
			}
			continue
		}
		break
	}
	return arr, nil
}

// onlyWhitespaceOrDelimiterLeft returns true if there are only whitespaces left in the
// input or if the remaining input is empty.
func (p *parser) onlyWhitespaceOrDelimiterLeft() bool {
	for pos := p.pos; pos < len(p.in); pos++ {
		if !isWhitespace(p.in[pos]) && p.in[pos] != ';' {
			return false
		}
	}
	return true
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
