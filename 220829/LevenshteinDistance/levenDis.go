/* Refer to the algorithm from the following link: 
 "https://vi.wikipedia.org/wiki/Kho%E1%BA%A3ng_c%C3%A1ch_Levenshtein#:~:text=Kho%E1%BA%A3ng%20c%C3%A1ch%20n%C3%A0y%20%C4%91%C6%B0%E1%BB%A3c%20%C4%91%E1%BA%B7t,ch%C3%ADnh%20t%E1%BA%A3%20c%E1%BB%A7a%20winword%20spellchecker."*/
package main

import (
	"fmt"
	"unicode/utf8"
  	"bufio"
  	"os"
	"strings"
)

func levenDis(str1, str2 string) int {	
	switch {
	case len(str1) == 0:
		return utf8.RuneCountInString(str1)
	case len(str2) == 0:
		return utf8.RuneCountInString(str2)
	case str1 == str2:
		return 0
	default:
		/*Add 1 space character at the beginning of the string 
			and remove duplicate whitespace*/
		str1 = " " + trimAllSpace(str1)
		str2 = " " + trimAllSpace(str2)

		/*Use Rune type to store string*/
		s1 := []rune(str1)
		s2 := []rune(str2)
		
		/*Get the length of the string*/
		lenS1 := len(s1) 
		lenS2 := len(s2)
		
		/*algorithm Levenshtein distance*/
		var disTable [20][20] int

		for i := 0; i < lenS1; i++ {
			disTable[i][0] = i
		}
		for j := 0; j < lenS2; j++ {
			disTable[0][j] = j
		}

		cost := 0
		for i := 1; i < lenS1; i++ {
			for j := 1; j < lenS2; j++ {
				if s1[i] == s2[j] {
					cost = 0
				}else {
					cost = 1
				}
				disTable[i][j] = minimum(disTable[i-1][j] + 1, 
										disTable[i][j-1] + 1, 
										disTable[i-1][j-1] + cost) 
			}
		}
		// Print disTable 
		// for i := 0; i < lenS1; i++ {
		// 	fmt.Println()
		// 	for j := 0; j < lenS2; j++ {
		// 		fmt.Print(disTable[i][j] ," ")
		// 	}
		// }

		return disTable[lenS1-1][lenS2-1]
	}
}

func minimum(a, b, c int) int {
	minVal := a
	if minVal >  b {
		minVal = b
	}
	if minVal >  c {
		minVal = c
	}
	return minVal
}

func trimAllSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func main() {
	// s2 := "áitteng"
	// s1 := "sitting"
	// fmt.Println("\nKhoang cach: ",levenDis(s1, s2))
	strArr := [12]string{"Bật đèn 1", "Tắt đèn 1", "bật đèn 1", "tắt đèn 1", "Led 1 on", "Led 1 off", 
							"Bật đèn 2", "Tắt đèn 2", "bật đèn 2", "tắt đèn 2", "Led 2 on", "Led 2 off"}

	fmt.Println("Enter str: ")

    scanner := bufio.NewScanner(os.Stdin)
    scanner.Scan() // use `for scanner.Scan()` to keep reading
    first := scanner.Text()

	minValue := 10
	resStr := "Khong co"

	for i := 0; i < len(strArr); i++ {
		fmt.Println("KQ: ", levenDis(first, strArr[i]))
		if levenDis(first, strArr[i]) < minValue {
			minValue = levenDis(first, strArr[i])
			resStr = strArr[i]
		}
	}

	fmt.Println("Chuỗi gần giống nhất đó là: ", resStr)

	// fmt.Println(levenDis("bat den 1", "Bật đèn 1"))
	// fmt.Println(levenDis("たいへん", "たいひひ"))
}
