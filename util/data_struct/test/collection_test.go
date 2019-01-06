// Date         : 2018/12/4 4:34 PM
// Author       : zhangchaoyang
// Description  :
package test

import (
	"testing"
	"ParameterServer/util/data_struct"
	"fmt"
)

func TestBinarySearch(t *testing.T) {
	List := []int32{2, 4, 5, 7, 7, 8, 9, 14}
	var Target int32
	for Target = 15; Target >= 0; Target-- {
		idx := data_struct.BinarySearchBigger(List, Target)
		fmt.Printf("%d %d\n", Target, idx)
	}

	fmt.Println("===========================")

	for Target = 15; Target >= 0; Target-- {
		idx := data_struct.BinarySearchSmaller(List, Target)
		fmt.Printf("%d %d\n", Target, idx)
	}
}
