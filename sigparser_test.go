package sigparser

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseSignature(t *testing.T) {
	tests := []struct {
		sig     string
		want    Signature
		wantErr bool
	}{
		// Simple cases
		{
			sig:  "foo",
			want: Signature{Name: "foo"},
		},
		{
			sig:  "foo()",
			want: Signature{Name: "foo"},
		},
		{
			sig:  "_foo(_foo _foo)",
			want: Signature{Name: "_foo", Inputs: []Parameter{{Name: "_foo", Type: "_foo"}}},
		},
		{
			sig:  "$foo($foo $foo)",
			want: Signature{Name: "$foo", Inputs: []Parameter{{Name: "$foo", Type: "$foo"}}},
		},
		{
			sig:  "$0($0 $0)",
			want: Signature{Name: "$0", Inputs: []Parameter{{Name: "$0", Type: "$0"}}},
		},
		{
			sig: "foo(uint256)", // with one argument
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256"}},
			},
		},
		{
			sig: "foo(uint256,bool)", // with two arguments
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256"}, {Type: "bool"}},
			},
		},
		{
			sig: "foo(uint256 a)", // with one named argument
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256", Name: "a"}},
			},
		},
		{
			sig: "foo(uint256 a, bool b)", // with two named arguments
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256", Name: "a"}, {Type: "bool", Name: "b"}},
			},
		},
		{
			sig: "foo()(uint256)", // with one return value
			want: Signature{
				Name:    "foo",
				Outputs: []Parameter{{Type: "uint256"}},
			},
		},
		{
			sig: "foo()(uint256,bool)", // with two return values
			want: Signature{
				Name:    "foo",
				Outputs: []Parameter{{Type: "uint256"}, {Type: "bool"}},
			},
		},
		{
			sig: "foo()(uint256 a)", // with one named return value
			want: Signature{
				Name:    "foo",
				Outputs: []Parameter{{Type: "uint256", Name: "a"}},
			},
		},
		{
			sig: "foo()(uint256 a, bool b)", // with two named return values
			want: Signature{
				Name:    "foo",
				Outputs: []Parameter{{Type: "uint256", Name: "a"}, {Type: "bool", Name: "b"}},
			},
		},
		{
			sig: "foo(uint256)(uint256)", // with one argument and one return value
			want: Signature{
				Name:    "foo",
				Inputs:  []Parameter{{Type: "uint256"}},
				Outputs: []Parameter{{Type: "uint256"}},
			},
		},
		// Tuples
		{
			sig: "foo((uint256,bool))", // with one tuple argument
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "", Tuple: []Parameter{{Type: "uint256"}, {Type: "bool"}}}},
			},
		},
		{
			sig: "foo((uint256 a,bool b) c)", // with one named tuple argument
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "", Tuple: []Parameter{{Type: "uint256", Name: "a"}, {Type: "bool", Name: "b"}}, Name: "c"}},
			},
		},
		{
			sig: "foo((uint256,bool)[])", // with array of tuples argument
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "", Arrays: []int{-1}, Tuple: []Parameter{{Type: "uint256"}, {Type: "bool"}}}},
			},
		},
		{
			sig: "foo()((uint256,bool))", // with one tuple return value
			want: Signature{
				Name:    "foo",
				Outputs: []Parameter{{Type: "", Tuple: []Parameter{{Type: "uint256"}, {Type: "bool"}}}},
			},
		},
		{
			sig: "foo()((uint256 a,bool b) c)", // with one named tuple return value
			want: Signature{
				Name:    "foo",
				Outputs: []Parameter{{Type: "", Tuple: []Parameter{{Type: "uint256", Name: "a"}, {Type: "bool", Name: "b"}}, Name: "c"}},
			},
		},
		// Alternative tuple syntax
		{
			sig: "foo(tuple(uint256,bool))", // with one tuple argument
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "", Tuple: []Parameter{{Type: "uint256"}, {Type: "bool"}}}},
			},
		},
		// Arrays
		{
			sig: "foo(uint256[])", // with one array argument
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256", Arrays: []int{-1}}},
			},
		},
		{
			sig: "foo(uint256[2])", // with one fixed array argument
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256", Arrays: []int{2}}},
			},
		},
		{
			sig: "foo(uint256[2][3])", // with nested arrays
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256", Arrays: []int{2, 3}}},
			},
		},
		{
			sig: "foo(uint256[2][][4])", // with nested arrays, one unbounded
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256", Arrays: []int{2, -1, 4}}},
			},
		},
		{
			sig: "foo(uint256[] a)", // with one named array argument
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256", Arrays: []int{-1}, Name: "a"}},
			},
		},
		// Different types
		{
			sig: "function foo()",
			want: Signature{
				Kind: FunctionKind,
				Name: "foo",
			},
		},
		{
			sig: "constructor()",
			want: Signature{
				Kind: ConstructorKind,
				Name: "",
			},
		},
		{
			sig: "fallback()",
			want: Signature{
				Kind: FallbackKind,
			},
		},
		{
			sig: "receive()",
			want: Signature{
				Kind: ReceiveKind,
			},
		},
		{
			sig: "event foo(uint256)",
			want: Signature{
				Kind:   EventKind,
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256"}},
			},
		},
		{
			sig: "error foo(uint256)",
			want: Signature{
				Kind:   ErrorKind,
				Name:   "foo",
				Inputs: []Parameter{{Type: "uint256"}},
			},
		},
		// Data location
		{
			sig: "foo(int memory a, int storage, int calldata)",
			want: Signature{
				Name:   "foo",
				Inputs: []Parameter{{Type: "int", Name: "a", DataLocation: Memory}, {Type: "int", DataLocation: Storage}, {Type: "int", DataLocation: CallData}},
			},
		},
		{
			sig: "foo()(int memory a, int storage, int calldata)",
			want: Signature{
				Name:    "foo",
				Outputs: []Parameter{{Type: "int", Name: "a", DataLocation: Memory}, {Type: "int", DataLocation: Storage}, {Type: "int", DataLocation: CallData}},
			},
		},
		// Modifiers
		{
			sig: "foo() view pure",
			want: Signature{
				Name:      "foo",
				Modifiers: []string{"view", "pure"},
			},
		},
		{
			sig: "foo() view pure returns (int)",
			want: Signature{
				Name:      "foo",
				Modifiers: []string{"view", "pure"},
				Outputs:   []Parameter{{Type: "int"}},
			},
		},
		//
		// Allowed arguments for fallback function
		{
			sig: "fallback (bytes calldata _input) external returns (bytes memory _output)",
			want: Signature{
				Kind:      FallbackKind,
				Inputs:    []Parameter{{Type: "bytes", Name: "_input", DataLocation: CallData}},
				Outputs:   []Parameter{{Type: "bytes", Name: "_output", DataLocation: Memory}},
				Modifiers: []string{"external"},
			},
		},
		// Different formatting
		{
			sig: "foo(t1 n1,(t2 n2,t3 n3))(t4 n4,(t5 n5,t6 n6))",
			want: Signature{
				Name:    "foo",
				Inputs:  []Parameter{{Type: "t1", Name: "n1"}, {Type: "", Tuple: []Parameter{{Type: "t2", Name: "n2"}, {Type: "t3", Name: "n3"}}}},
				Outputs: []Parameter{{Type: "t4", Name: "n4"}, {Type: "", Tuple: []Parameter{{Type: "t5", Name: "n5"}, {Type: "t6", Name: "n6"}}}},
			},
		},
		{
			sig: " foo ( t1 n1, (t2 n2, t3 n3) ) returns (t4 n4, ( t5 n5, t6 n6 )) ",
			want: Signature{
				Name:    "foo",
				Inputs:  []Parameter{{Type: "t1", Name: "n1"}, {Type: "", Tuple: []Parameter{{Type: "t2", Name: "n2"}, {Type: "t3", Name: "n3"}}}},
				Outputs: []Parameter{{Type: "t4", Name: "n4"}, {Type: "", Tuple: []Parameter{{Type: "t5", Name: "n5"}, {Type: "t6", Name: "n6"}}}},
			},
		},
		{
			sig: "\t\nfunction\t\nfoo\t(t1\nn1,(t2\tn2,t3\nn3))\treturns\n(t4\nn4,(t5\tn5,t6\nn6))\t",
			want: Signature{
				Kind:    FunctionKind,
				Name:    "foo",
				Inputs:  []Parameter{{Type: "t1", Name: "n1"}, {Type: "", Tuple: []Parameter{{Type: "t2", Name: "n2"}, {Type: "t3", Name: "n3"}}}},
				Outputs: []Parameter{{Type: "t4", Name: "n4"}, {Type: "", Tuple: []Parameter{{Type: "t5", Name: "n5"}, {Type: "t6", Name: "n6"}}}},
			},
		},
		// Nested tuples and arrays
		{
			sig: "function foo(((int[][1][2] a,int[][1][2] b)[][1][2],(int[][1][2] a,int[][1][2] b)[][1][2])[][1][2])",
			want: Signature{
				Kind: FunctionKind,
				Name: "foo",
				Inputs: []Parameter{
					{
						Tuple: []Parameter{
							{
								Tuple: []Parameter{
									{Type: "int", Name: "a", Arrays: []int{-1, 1, 2}},
									{Type: "int", Name: "b", Arrays: []int{-1, 1, 2}},
								},
								Arrays: []int{-1, 1, 2},
							},
							{
								Tuple: []Parameter{
									{Type: "int", Name: "a", Arrays: []int{-1, 1, 2}},
									{Type: "int", Name: "b", Arrays: []int{-1, 1, 2}},
								},
								Arrays: []int{-1, 1, 2},
							},
						},
						Arrays: []int{-1, 1, 2},
					},
				},
			},
		},

		// Signatures with a valid syntax but invalid semantics
		{sig: "function foo(int indexed a)", wantErr: true},     // indexed flag not allowed for non-events
		{sig: "foo()(int indexed a)", wantErr: true},            // indexed flag not allowed for output values
		{sig: "foo()[1]", wantErr: true},                        // input tuples cannot be arrays
		{sig: "foo()(int)[1]", wantErr: true},                   // output tuples cannot be arrays
		{sig: "constructor foo()", wantErr: true},               // constructors cannot have a name
		{sig: "constructor() internal", wantErr: true},          // constructors cannot have modifiers
		{sig: "constructor() returns (uint256)", wantErr: true}, // constructors cannot have return values
		{sig: "fallback foo()", wantErr: true},                  // fallbacks cannot have a name
		{sig: "fallback(uint256)", wantErr: true},               // fallbacks cannot have arguments other that bytes
		{sig: "fallback() returns (uint256)", wantErr: true},    // fallbacks cannot have return values other that bytes
		{sig: "fallback(bytes indexed a) ", wantErr: true},      // indexed flag not allowed for non-events
		{sig: "receive foo()", wantErr: true},                   // receives cannot have a name
		{sig: "receive(uint256)", wantErr: true},                // receives cannot have arguments
		{sig: "receive() returns (uint256)", wantErr: true},     // receives cannot have return values
		{sig: "event foo()", wantErr: true},                     // events must have arguments
		{sig: "event foo(int) internal", wantErr: true},         // events cannot have modifiers
		{sig: "event foo(int) returns (int)", wantErr: true},    // events cannot have return values
		{sig: "event foo(int memory a)", wantErr: true},         // event arguments cannot specify data location
		{sig: "error foo()", wantErr: true},                     // errors must have arguments
		{sig: "error foo(int) internal", wantErr: true},         // errors cannot have modifiers
		{sig: "error foo() returns (int)", wantErr: true},       // errors cannot have return values
		{sig: "error foo(int memory a)", wantErr: true},         // error arguments cannot specify data location
		{sig: "error foo(int indexed a)", wantErr: true},        // indexed flag not allowed for non-events
		// Invalid syntax
		{sig: "foo()()a", wantErr: true},
		{sig: "foo()returns[]", wantErr: true},
		{sig: "foo((", wantErr: true},
		{sig: "foo()((", wantErr: true},
		{sig: "foo() a b [", wantErr: true},
		{sig: "foo() a returns b", wantErr: true},
		{sig: "foo(.)", wantErr: true},
		{sig: "foo((int a .))", wantErr: true},
		{sig: "foo((int)[a])", wantErr: true},
		{sig: "foo(int[-1])", wantErr: true},
		{sig: "foo(int[0])", wantErr: true},
		{sig: "foo(int[18446744073709551616])", wantErr: true},
		{sig: "foo(int[0xff])", wantErr: true},
		{sig: "(A", wantErr: true},
		{sig: "A( ", wantErr: true},
		{sig: "(A[", wantErr: true},
		{sig: "A()returns", wantErr: true},
	}
	for n, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
			got, err := ParseSignature(tt.sig)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSignature() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseParameter(t *testing.T) {
	tests := []struct {
		param   string
		want    Parameter
		wantErr bool
	}{
		// Simple types
		{param: "int", want: Parameter{Type: "int"}},
		{param: "int a", want: Parameter{Type: "int", Name: "a"}},
		// Arrays
		{param: "int[]", want: Parameter{Type: "int", Arrays: []int{-1}}},
		{param: "int[1]", want: Parameter{Type: "int", Arrays: []int{1}}},
		{param: "int[][1]", want: Parameter{Type: "int", Arrays: []int{-1, 1}}},
		{param: "int[] a", want: Parameter{Type: "int", Arrays: []int{-1}, Name: "a"}},
		// Names with special characters
		{param: "_a", want: Parameter{Type: "_a"}},
		{param: "$a", want: Parameter{Type: "$a"}},
		{param: "a_", want: Parameter{Type: "a_"}},
		{param: "a$", want: Parameter{Type: "a$"}},
		{param: "a0", want: Parameter{Type: "a0"}},
		{param: "int _a", want: Parameter{Type: "int", Name: "_a"}},
		{param: "int $a", want: Parameter{Type: "int", Name: "$a"}},
		{param: "int a_", want: Parameter{Type: "int", Name: "a_"}},
		{param: "int a$", want: Parameter{Type: "int", Name: "a$"}},
		{param: "int a0", want: Parameter{Type: "int", Name: "a0"}},
		// Types with data location
		{param: "int storage", want: Parameter{Type: "int", DataLocation: Storage}},
		{param: "int storage a", want: Parameter{Type: "int", Name: "a", DataLocation: Storage}},
		{param: "int calldata a", want: Parameter{Type: "int", Name: "a", DataLocation: CallData}},
		{param: "int memory a", want: Parameter{Type: "int", Name: "a", DataLocation: Memory}},
		{param: "int indexed a", want: Parameter{Type: "int", Name: "a", Indexed: true}},
		// Tuples
		{param: "(int,int)", want: Parameter{
			Tuple: []Parameter{
				{Type: "int"}, {Type: "int"},
			}},
		},
		{param: "(int,int)[1]", want: Parameter{
			Tuple: []Parameter{
				{Type: "int"}, {Type: "int"},
			},
			Arrays: []int{1},
		}},
		{param: "(int,int)[][1]", want: Parameter{
			Tuple: []Parameter{
				{Type: "int"}, {Type: "int"},
			},
			Arrays: []int{-1, 1},
		}},
		{param: "(int a, int b) c", want: Parameter{
			Name: "c",
			Tuple: []Parameter{
				{Type: "int", Name: "a"},
				{Type: "int", Name: "b"},
			},
		}},
		{param: "((int a,int b),(int c,int d)) e", want: Parameter{
			Name: "e",
			Tuple: []Parameter{
				{Tuple: []Parameter{{Type: "int", Name: "a"}, {Type: "int", Name: "b"}}},
				{Tuple: []Parameter{{Type: "int", Name: "c"}, {Type: "int", Name: "d"}}},
			}},
		},
		{param: "((int[] a,int[] b)[])", want: Parameter{
			Tuple: []Parameter{
				{
					Tuple: []Parameter{
						{Type: "int", Arrays: []int{-1}, Name: "a"},
						{Type: "int", Arrays: []int{-1}, Name: "b"},
					},
					Arrays: []int{-1},
				},
			}},
		},
		// Whitespaces
		{param: " int", want: Parameter{Type: "int"}},
		{param: "int ", want: Parameter{Type: "int"}},
		{param: " int  memory  a  ", want: Parameter{Type: "int", Name: "a", DataLocation: Memory}},
		{param: "\nint[1]\na", want: Parameter{Type: "int", Arrays: []int{1}, Name: "a"}},
		// Invalid syntax
		{param: "int[", wantErr: true},
		{param: "int[1", wantErr: true},
		{param: "int [1]", wantErr: true},
		{param: "int[ 1]", wantErr: true},
		{param: "int[1 ]", wantErr: true},
		{param: "0[1]", wantErr: true},
		{param: "int 0", wantErr: true},
		{param: "int a[1]", wantErr: true},
		{param: "int[0]", wantErr: true},
		{param: "int[-1]", wantErr: true},
		{param: "int[18446744073709551616]", wantErr: true},
		{param: "int[0xff]", wantErr: true},
		{param: "a^b", wantErr: true},
		{param: "int a^b", wantErr: true},
		{param: "int a a", wantErr: true},
	}
	for n, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
			got, err := ParseParameter(tt.param)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseParameter() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignatureString(t *testing.T) {
	tests := []struct {
		sig  Signature
		want string
	}{
		{sig: mustParseSignature(t, "foo"), want: "foo()"},
		{sig: mustParseSignature(t, "function foo"), want: "function foo()"},
		{sig: mustParseSignature(t, "constructor()"), want: "constructor()"},
		{sig: mustParseSignature(t, "fallback()"), want: "fallback()"},
		{sig: mustParseSignature(t, "receive()"), want: "receive()"},
		{sig: mustParseSignature(t, "event foo(int)"), want: "event foo(int)"},
		{sig: mustParseSignature(t, "error foo(int)"), want: "error foo(int)"},
		{sig: mustParseSignature(t, "foo(int)"), want: "foo(int)"},
		{sig: mustParseSignature(t, "foo(int a, int b)"), want: "foo(int a, int b)"},
		{sig: mustParseSignature(t, "foo(int[][1][2] a)"), want: "foo(int[][1][2] a)"},
		{sig: mustParseSignature(t, "foo()(int)"), want: "foo() returns (int)"},
		{sig: mustParseSignature(t, "foo()(int a)"), want: "foo() returns (int a)"},
		{sig: mustParseSignature(t, "foo()(int a, int b)"), want: "foo() returns (int a, int b)"},
		{sig: mustParseSignature(t, "foo()(int[][1][2] a)"), want: "foo() returns (int[][1][2] a)"},
		{sig: mustParseSignature(t, "foo(int storage)"), want: "foo(int storage)"},
		{sig: mustParseSignature(t, "foo(int memory)"), want: "foo(int memory)"},
		{sig: mustParseSignature(t, "foo(int calldata)"), want: "foo(int calldata)"},
		{sig: mustParseSignature(t, "event foo(int indexed)"), want: "event foo(int indexed)"},
		{sig: mustParseSignature(t, "foo(int storage a)"), want: "foo(int storage a)"},
		{sig: mustParseSignature(t, "foo() internal pure"), want: "foo() internal pure"},
		{sig: mustParseSignature(t, "foo() internal pure (int)"), want: "foo() internal pure returns (int)"},
		{sig: mustParseSignature(t, "foo((int,int))"), want: "foo((int, int))"},
		{sig: mustParseSignature(t, "foo((int,int)[])"), want: "foo((int, int)[])"},
	}
	for n, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
			if got := tt.sig.String(); got != tt.want {
				t.Errorf("Signature.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func FuzzParseSignature(f *testing.F) {
	for _, s := range []string{
		"function",
		"constructor",
		"fallback",
		"receive",
		"event",
		"error",
		"foo",
		"(",
		")",
		"[",
		"]",
		",",
		"tuple",
		"indexed",
		"storage",
		"memory",
		"calldata",
		"returns",
		"_",
		"$",
		" ",
		"\n",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ParseSignature(s)
	})
}

func FuzzParseParameter(f *testing.F) {
	for _, s := range []string{
		"foo",
		"(",
		")",
		"[",
		"]",
		",",
		"tuple",
		"indexed",
		"storage",
		"memory",
		"calldata",
		"_",
		"$",
		" ",
		"\n",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ParseParameter(s)
	})
}

func mustParseSignature(t *testing.T, s string) Signature {
	sig, err := ParseSignature(s)
	if err != nil {
		t.Fatal(err)
	}
	return sig
}
