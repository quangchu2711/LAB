package main

import (
	"fmt"

	"github.com/abadojack/whatlanggo"
)

func main() {
	info := whatlanggo.Detect("/led on 1")
	fmt.Println("Language:", info.Lang.String(), " Script:", whatlanggo.Scripts[info.Script], " Confidence: ", info.Confidence)
}