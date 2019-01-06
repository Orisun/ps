// Date         : 2018/8/17 下午5:39
// Author       : zhangchaoyang
// Description  :
package util

import (
	metric2 "pkg/metric"
	"time"
)

var (
	Metric metric2.Metric
)

func InitMetric(project string) {
	prefix := project

	Metric = metric2.NewBufferedMetric(prefix, time.Second/2, 50)
	if Metric == nil {
		panic("create radic metric failed")
	}
}
