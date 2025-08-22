package foo

import "fmt"

var (
	foobar = "fee"
	feebar = "foo"
)

func FOO() {
	fmt.Printf("The setting of FOOBAR is %s\n", foobar)
	fmt.Printf("The setting of FEEBAR is %s\n", feebar)
}
