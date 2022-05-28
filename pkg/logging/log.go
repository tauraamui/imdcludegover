package logging

import "fmt"

var OUTPUT = false

func Println(l string) {
	if OUTPUT {
		fmt.Println(l)
	}
}

func Printfln(l string, a ...interface{}) {
	if OUTPUT {
		fmt.Printf(l+"\n", a...)
	}
}
