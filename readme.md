# go-sigparser

The `go-sigparser` package provides a parser of Ethereum function, event and error signatures.

The parser is based on the Solidity grammar, but allows to omit argument names and the `returns` and `function`
keywords, so it can parse full Solidity signatures as well as short signatures like: `bar(uint256,bytes32)`.
Tuples are represented as a list of parameters, e.g. `(uint256,bytes32)`. The list can be optionally prefixed with
`tuple` keyword, e.g. `tuple(uint256,bytes32)`.

Examples of signatures that are supported by the parser:

- `getPrice(string)`
- `getPrice(string)((uint256,unit256))`
- `function getPrice(string calldata symbol) external view returns (tuple(uint256 price, uint256 timestamp) result)`
- `constructor(string symbol, string name)`
- `receive() external payable`
- `fallback (bytes calldata input) external payable returns (bytes memory output)`
- `event PriceUpated(string indexed symbol, uint256 price)`
- `error PriceExpired(string symbol, uint256 timestamp)`

## Installation

```bash
go get github.com/ethereum/go-sigparser
```

## Usage

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

	fmt.Println(sig.Kind)                     // FunctionKind
	fmt.Println(sig.Name)                     // getPrice
	fmt.Println(sig.Inputs[0].Name)           // symbol
	fmt.Println(sig.Inputs[0].Type)           // string
	fmt.Println(sig.Inputs[0].Arrays)         // [-1] (-1 means that the array is unbounded)
	fmt.Println(sig.Inputs[0].DataLocation)   // CallData
	fmt.Println(sig.Modifiers)                // [external, view]
	fmt.Println(sig.Outputs[0].Name)          // result
	fmt.Println(sig.Outputs[0].Arrays)        // [-1] 
	fmt.Println(sig.Outputs[0].Tuple[0].Name) // price
	fmt.Println(sig.Outputs[0].Tuple[0].Type) // uint256
	fmt.Println(sig.Outputs[0].Tuple[1].Name) // timestamp
	fmt.Println(sig.Outputs[0].Tuple[1].Type) // uint256
}
```

## Documentation

[https://pkg.go.dev/github.com/defiweb/go-sigparser](https://pkg.go.dev/github.com/defiweb/go-sigparser)
