// +build integration

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

func main() {
	path := flag.String("path", "", "")
	singleLine := flag.Bool("singleLine", false, "")
	times := flag.Int("times", 1, "")
	sleepTime := flag.Duration("sleepTime", 0 * time.Second, "")
	flag.Parse()
	content, err := ioutil.ReadFile(string(*path))
	if err != nil {
		panic(err)
	}
	contentStr := string(content)
	if *singleLine {
		contentStr = strings.ReplaceAll(contentStr, "\n", "")
	}
	for i:=0;i<*times;i++{
		fmt.Println(contentStr)
		time.Sleep(*sleepTime)
	}
}
