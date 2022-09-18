 package main

 import (
         "fmt"
         "golang.org/x/text/runes"
         "golang.org/x/text/transform"
         "golang.org/x/text/unicode/norm"
         "unicode"
         "strings"
 )

func getNormStr(inputStr string) string {

        lowerStr := strings.ToLower(inputStr)

        t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
        normStr, _, _ := transform.String(t, lowerStr)

        return normStr      
}

 func main() {
        strArr := [...]string{"Bật đèn 1", "Bat den 1", "bật đèn 1", "bat den 1", "BẬT ĐÈN 1"}

        for _, str := range strArr {
                fmt.Printf("%s -> %s\n", str, getNormStr(str))
        }
}