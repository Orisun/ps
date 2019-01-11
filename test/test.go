// Date         : 2018/12/22
// Author       : zhangchaoyang
// Description  : 
package main

import (
	"fmt"
	"math/rand"
)

func main() {
	for i:=0;i<10;i++{
		slotIdx := -rand.Int63()
		fmt.Println(slotIdx)
		fmt.Println(int32(slotIdx & 0x7fffffff))
		fmt.Println(int32(slotIdx))
		fmt.Println("======")
	}
}
