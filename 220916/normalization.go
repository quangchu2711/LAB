 package main

 import (
         "fmt"
         // "golang.org/x/text/transform"
         "golang.org/x/text/unicode/norm"
         // "strings"
         // "unicode"
 )

 // func isMn(r rune) bool {
 //     return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
 // }


 func main() {

        input1 := []byte("\u00c5")
        outputNFD := norm.NFD.Bytes(input1)
        fmt.Println("NFD: ", input1, "->", outputNFD)
        fmt.Printf("%s -> %s\n\n", string(input1), string(outputNFD))

        input2 := []byte("Å")
        outputNFC := norm.NFC.Bytes(input2)
        fmt.Println("NFC: ", input2, "->", outputNFC)
        fmt.Printf("%s -> %s\n\n", string(input2), string(outputNFC))

        input3 := []byte("ẛ̣")
        outputNFKD := norm.NFKD.Bytes(input3)
        fmt.Println("NFKD: ", input3, "->", outputNFKD)
        fmt.Printf("%s -> %s\n\n", string(input3), string(outputNFKD))

        input4 := []byte("ẛ̣")
        outputNFKC := norm.NFKC.Bytes(input4)
        fmt.Println("NFKC: ", input4, "->", outputNFKC)
        fmt.Printf("%s -> %s\n\n", string(input4), string(outputNFKC))

         // str1 := "ElNi\u212A"
         // t := transform.Chain(norm.NFKD, transform.RemoveFunc(isMn), norm.NFKC)
         // normStr1, _, _ := transform.String(t, str1)
         // fmt.Printf("%s length is %d \n", str1, len(str1))
         // fmt.Println("Normalizing unicode strings....")
         // fmt.Printf("%s length is %d \n", normStr1, len(normStr1))

        // input := []byte(`tschüß; до свидания`)

        // fmt.Println(norm.NFC.Bytes(input))
        // fmt.Println(norm.NFC.Bytes(input))
        // fmt.Println(norm.NFC.Bytes(input))

        // b := make([]byte, len(input))

        // t := transform.RemoveFunc(unicode.IsSpace)
        // n, _, _ := t.Transform(b, input, true)
        // fmt.Println(string(b[:n]))

        // t = transform.RemoveFunc(func(r rune) bool {
        //     return !unicode.Is(unicode.Latin, r)
        // })
        // n, _, _ = t.Transform(b, input, true)
        // fmt.Println(string(b[:n]))

        // n, _, _ := t.Transform(b, norm.NFD.Bytes(input), true)
        // fmt.Println(string(b[:n]))
        // fmt.Println(n)

         // str1 := "ElNi\u00f1o"
         // str2 := "ElNin\u0303o"

         // fmt.Printf("%s length is %d \n", str1, len(str1))
         // fmt.Printf("%s length is %d \n", str2, len(str2))

         // match := strings.EqualFold(str1, str2)
         // fmt.Println(match)

         // fmt.Println("Normalizing unicode strings....")

         // t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
         // normStr1, _, _ := transform.String(t, str1)
         // fmt.Printf("%s length is %d \n", normStr1, len(str1))

         // normStr2, _, _ := transform.String(t, str2)
         // fmt.Printf("%s length is %d \n", normStr2, len(str2))

         // match2 := strings.EqualFold(normStr1, normStr2)
         // fmt.Println(match2)
         //str1 := "ElNi\u00f1o"
         //str1 := "ElNi\u00e9o"
         //str2 := "Bật đèn 1"
         //str2 := "ElNin\u0303o"
         //str1 := "ElNin\u0391"
         // str1 := "ElNin\u0430"

         // fmt.Printf("%s length is %d \n", str1, len(str1))
         // //fmt.Printf("%s length is %d \n", str2, len(str2))

         // // match := strings.EqualFold(str1, str2)
         // // fmt.Println(match)

         // fmt.Println("Normalizing unicode strings....")
         // str1 := "ElNi\u004B"
       
         // normStr2, _, _ := transform.String(t, str2)
         // fmt.Printf("%s length is %d \n", normStr2, len(str2))

         // match2 := strings.EqualFold(normStr1, normStr2)
         // fmt.Println(match2)
         // fmt.Printf("\n[%s, len: %d - %s, len: %d]", normStr1, len(normStr1), normStr2, len(normStr2))
 
        // t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
        // result, _, _ := transform.String(t, "São Paulo, Brazil. Wien, Österreich.")
        // fmt.Println(result)
 }