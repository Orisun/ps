// Date         : 2018/11/21 8:13 PM
// Author       : zhangchaoyang
// Description  :
package util

import (
	"sync"
	"time"
)

//WaitTimeout 等一个WaitGroup结束，并限定超时时间。如果超时则返回false
func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return true //正常结束
	case <-time.After(timeout):
		return false // 超时
	}
}
