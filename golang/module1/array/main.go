package main

import (
	"fmt"
)

func main() {
	myArray := [5]string{"I", "am", "stupid", "and", "weak"}
	fmt.Println(myArray)
	for index, value := range myArray {
		fmt.Println(index, value)
		switch value {
		case "stupid":
			myArray[index] = "smart"
		case "weak":
			myArray[index] = "strong"
		default:
			break
		}
	}
	fmt.Println(myArray)
}
