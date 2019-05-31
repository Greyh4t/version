# version

版本号解析器，主要是为了适应大部分非标版本号，提供解析，比较等功能

## Example

```go
package main

import (
	"fmt"

	"github.com/Greyh4t/version"
)

func main() {
	version1 := version.Parse("1.11.2.dev")
	version2 := version.Parse("1.11.2.dev-1")
	version3 := version.Parse("1.2.2.release")
	fmt.Println(version1.Lt(version2))
	fmt.Println(version3.Gt(version1))
}

```