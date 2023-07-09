# go-sigparser

The `go-sigparser` is a Go package that helps parse Ethereum function signatures. This parser uses the Solidity grammar
rules.

However, it doesn't need argument names or the `returns` and `function` keywords to parse signatures. So, you can use it
for both complete Solidity signatures and shorter versions like `bar(uint256,bytes32)`.

Tuples are represented as a list of parameters, e.g. `(uint256,bytes32)`. The list can be optionally prefixed with
`tuple` keyword, e.g. `tuple(uint256,bytes32)`.

The `go-sigparser` supports many different signature formats. Here are a few examples:

- `getPrice(string)`
- `getPrice(string)((uint256,unit256))`
- `function getPrice(string calldata symbol) external view returns (tuple(uint256 price, uint256 timestamp) result)`
- `constructor(string symbol, string name)`
- `receive() external payable`
- `fallback (bytes calldata input) external payable returns (bytes memory output)`
- `event PriceUpdated(string indexed symbol, uint256 price)`
- `error PriceExpired(string symbol, uint256 timestamp)`

## Installation

You can install the `go-sigparser` package with this command:

```bash
go get github.com/ethereum/go-sigparser
```

## Usage

### Parsing Signature

```go
package main

import (
	"fmt"

	"github.com/defiweb/go-sigparser"
)

func main() {
	sig, err := sigparser.ParseSignature("function getPrices(string[] calldata symbols) external view returns ((uint256 price, uint256 timestamp)[] result)")
	if err != nil {
		panic(err)
	}

	fmt.Println(sig.Kind)                     // function
	fmt.Println(sig.Name)                     // getPrice
	fmt.Println(sig.Inputs[0].Name)           // symbol
	fmt.Println(sig.Inputs[0].Type)           // string
	fmt.Println(sig.Inputs[0].Arrays)         // [-1] (-1 means that the array is unbounded)
	fmt.Println(sig.Inputs[0].DataLocation)   // calldata
	fmt.Println(sig.Modifiers)                // [external, view]
	fmt.Println(sig.Outputs[0].Name)          // result
	fmt.Println(sig.Outputs[0].Arrays)        // [-1] 
	fmt.Println(sig.Outputs[0].Tuple[0].Name) // price
	fmt.Println(sig.Outputs[0].Tuple[0].Type) // uint256
	fmt.Println(sig.Outputs[0].Tuple[1].Name) // timestamp
	fmt.Println(sig.Outputs[0].Tuple[1].Type) // uint256
}
```

### Parsing Parameter

```go
package main

import (
	"fmt"

	"github.com/defiweb/go-sigparser"
)

func main() {
	param, err := sigparser.ParseParameter("(uint256 price, uint256 timestamp)")
	if err != nil {
		panic(err)
	}

	fmt.Println(param.Tuple[0].Name) // price
	fmt.Println(param.Tuple[0].Type) // uint256
	fmt.Println(param.Tuple[1].Name) // timestamp
	fmt.Println(param.Tuple[1].Type) // uint256
}
```

### Parsing Struct

You can also parse structs with `go-sigparser`. When you parse a struct, you get a tuple where the struct name
is the parameter name, and the struct fields are the tuple elements. Here's an example:

```go
package main

import (
	"fmt"

	"github.com/defiweb/go-sigparser"
)

func main() {
	param, err := sigparser.ParseStruct("struct name { uint256 price; uint256 timestamp; }")
	if err != nil {
		panic(err)
	}

	fmt.Println(param.Name)          // name
	fmt.Println(param.Tuple[0].Name) // price
	fmt.Println(param.Tuple[0].Type) // uint256
	fmt.Println(param.Tuple[1].Name) // timestamp
	fmt.Println(param.Tuple[1].Type) // uint256
}
```

## Documentation

For more information about the `go-sigparser` package,
visit [https://pkg.go.dev/github.com/defiweb/go-sigparser](https://pkg.go.dev/github.com/defiweb/go-sigparser).
