基于influxdb+grafana搭建的数据上报、监控报警系统。<br>
# 数据上报
## python
需要先 pip install pytelegraf
```python
import random
import time

from metric import Metric

metric = Metric("test")


def Search():
    begin = time.time()
    metric.OccurOnce("Search")  # 函数每调用一次就上报一个计数

    count = random.randint(0, 9)
    result = [0] * count

    metric.ReportCount("RetriveCount", len(result))  # 上报搜索结果数
    if len(result) == 0:
        metric.OccurOnce("RetriveZero")  # 无搜索结果，发生一次业务异常就上报一个计数

    metric.UseTime("Search", time.time() - begin)  # 上报函数耗时
    return result


if __name__ == "__main__":
    for i in xrange(100):
        Search()
        time.sleep(0.1)

```
## go
```gotemplate
import (
	metric2 "pkg/metric"
	"time"
	"math/rand"
)

var metric metric2.Metric

func init() {
	//metric = metric2.NewMetric("test")
	metric = metric2.NewBufferedMetric("test", time.Second/2, 50)
	if metric == nil {
		panic("create metric failed")
	}
}

func Search() []int {
	begin := time.Now()
	metric.OccurOnce("Search", nil) //函数每调用一次就上报一个计数

	count := rand.Intn(10)
	result := make([]int, count)

	metric.ReportCount("RetriveCount", len(result), nil) //上报搜索结果数
	if len(result) == 0 {
		metric.OccurOnce("RetriveZero", nil) //无搜索结果，发生一次业务异常就上报一个计数
	}

	metric.UseTime("Search", time.Since(begin), nil) //上报函数耗时
	return result
}

func main() {
	defer metric.Close()
	for i := 0; i < 10000; i++ {
		Search()
		time.Sleep(100 * time.Millisecond)
	}
}

```
# 数据展现
[数据展现后台](http://jobflume:3000/)<br>
需要先找zhangchaoyang开通账号。
## 计数类展现
![avatar](img/计数展现.png)
## 耗时类展现
![avatar](img/耗时展现.png)<br>
注意把Y轴的时间单位设为纳秒。<br>
![avatar](img/axis.png)
# 监控报警
## 设置邮件组
![avatar](img/channel.png)
## 报警阈值设置
![avatar](img/计数alert.png)<br>
设置时间阈值时注意单位是纳秒。<br>
![avatar](img/耗时类alert.png)
## 收件人设置
![avatar](img/报警接收人.png)
## 报警邮件形式
![avatar](img/mail.png)