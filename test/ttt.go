package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func main() {

	name := "asdasd.png"

	fmt.Println(strings.Replace(name, filepath.Ext(name), "", 1))

}
