package main

import (
	"fmt"

	"github.com/abadojack/whatlanggo"
)

func main() {
	info := whatlanggo.Detect("Foje funkcias kaj foje ne funkcias")
	fmt.Println("Language:", info.Lang.String(), " Script:", whatlanggo.Scripts[info.Script], " Confidence: ", info.Confidence)
}