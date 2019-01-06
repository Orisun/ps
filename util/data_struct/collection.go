// Date         : 2018/1/11
// Author       : zhangchaoyang
// Description  :
package data_struct

import (
	"math/rand"
	"reflect"
	"time"
)

//Contain 判断obj是否在target中，target支持的类型arrary,slice,map
func Contain(obj interface{}, target interface{}) bool {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true
		}
	}

	return false
}

//Sample 从target中随机抽取一个元素，target支持的类型arrary,slice
func Sample(target interface{}) interface{} {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		idx := rand.Intn(targetValue.Len())
		return targetValue.Index(idx).Interface()
	}
	return nil
}

//Merge 合并两个列表
func Merge(target interface{}, other interface{}) interface{} {
	if reflect.TypeOf(target).Kind() != reflect.TypeOf(other).Kind() {
		return nil
	}
	targetValue := reflect.ValueOf(target)
	otherValue := reflect.ValueOf(other)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		rect := make([]interface{}, 0, targetValue.Len()+otherValue.Len())
		for i := 0; i < targetValue.Len(); i++ {
			rect = append(rect, targetValue.Index(i))

		}
		for i := 0; i < otherValue.Len(); i++ {
			rect = append(rect, otherValue.Index(i))

		}
		return rect
	}
	return nil
}

func Shuffle(arr []interface{}) []interface{} {
	if arr == nil || len(arr) == 0 {
		return nil
	}
	rect := make([]interface{}, len(arr))
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i, j := range r.Perm(len(arr)) {
		rect[i] = arr[j]
	}
	return rect
}

//BinarySearchBigger 从有序数组中寻找第一个大于等于Target的那个元素的下标。如果Target比最大的那个元素还大，则返回len(List)
func BinarySearchBigger(List []int32, Target int32) int {
	if len(List) == 0 {
		return -1
	}
	low := 0
	high := len(List) - 1
	mid := (low + high) / 2
	for ; low < mid; {
		if List[mid] >= Target {
			high = mid - 1
		} else {
			low = mid + 1
		}
		mid = (low + high) / 2
	}
	for i := low; i < len(List); i++ {
		if List[i] >= Target {
			return i
		}
	}
	return len(List)
}

//BinarySearchSmaller 从有序数组中寻找最后一个小于等于Target的那个元素的下标。如果Target比最小的那个元素还小，则返回-1
func BinarySearchSmaller(List []int32, Target int32) int {
	if len(List) == 0 {
		return -1
	}
	low := 0
	high := len(List) - 1
	mid := (low + high) / 2
	for ; low < mid; {
		if List[mid] <= Target {
			low = mid + 1
		} else {
			high = mid - 1
		}
		mid = (low + high) / 2
	}
	for i := high; i >= 0; i-- {
		if List[i] <= Target {
			return i
		}
	}
	return -1
}
