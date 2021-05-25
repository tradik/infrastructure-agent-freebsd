// +build integration

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
)

func main() {
	path := flag.String("path", "", "")
	singleLine := flag.Bool("singleLine", false, "")
	flag.Parse()
	content, err := ioutil.ReadFile(string(*path))
	if err != nil {
		panic(err)
	}
	contentStr := string(content)
	if *singleLine {
		contentStr = strings.ReplaceAll(contentStr, "\n", "")
	}
	fmt.Println(contentStr)
}
