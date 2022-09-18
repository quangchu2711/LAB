 package main

 import (
         "fmt"
         "golang.org/x/text/runes"
         "golang.org/x/text/transform"
         "golang.org/x/text/unicode/norm"
         "unicode"
         "strings"
 )

 // func isMn(r rune) bool {
 //     return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
 // }


      // - 'Bật đèn 1'
      // - 'Bat den 1'
      // - 'bật đèn 1'
      // - 'bat den 1'
func getNormStr(inputStr string) string {

        lowerStr := strings.ToLower(inputStr)

        t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
        // t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
        normStr, _, _ := transform.String(t, lowerStr)

        return normStr      
}

 func main() {
        strArr := [...]string{"Bật đèn 1", "Bat den 1", "bật đèn 1", "bat den 1", "BẬT ĐÈN 1"}

        for _, str := range strArr {
                fmt.Println(getNormStr(str))
        }

        // str1 := "ElNi\u00f1o"
        // getNormStr("Bật đèn 1")



        // str1 := "Bật đèn 1"
        // fmt.Printf("%s length is %d \n", str1, len(str1))

        // fmt.Println("Normalizing unicode strings....")
        // fmt.Printf("[%T]\n", norm.NFD)
        // t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
        // // t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
        // normStr1, _, _ := transform.String(t, str1)
        // fmt.Printf("%s length is %d \n", normStr1, len(normStr1))
}