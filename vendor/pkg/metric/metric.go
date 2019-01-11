// Date         : 2018/8/23 下午1:20
// Author       : zhangchaoyang
// Description  :
package metric

import (
	"time"
	"github.com/mdaffin/go-telegraf"
)

const INFLUX_SERVER_HOST = "influxdb.int.taou.com"
const INFLUX_SERVER_PORT = "8089" //UDP端口

type Metric interface {
	OccurOnce(string, map[string]string) error
	ReportCount(string, int, map[string]string) error
	UseTime(string, time.Duration, map[string]string) error
	Close() error
}

type BasicMetric struct {
	client telegraf.Client
	prefix string
}

//rmBlankValue 在influxdb中tags是用来作索引的，以string形式存储，所以tags的value不能是空字符串
func rmBlankValue(tags map[string]string) map[string]string {
	rect := make(map[string]string)
	if tags == nil || len(tags) == 0 {
		return rect
	}
	for k, v := range tags {
		if v != "" {
			rect[k] = v
		}
	}
	return rect
}

//NewMetric
func NewMetric(MetricPrefix string) Metric {
	if client, err := telegraf.NewUDP(INFLUX_SERVER_HOST + ":" + INFLUX_SERVER_PORT); err == nil {
		metric := &BasicMetric{client: client, prefix: MetricPrefix}
		return metric
	} else {
		return nil
	}
}

//OccurOnce 发生一次，上报计数1
func (metric *BasicMetric) OccurOnce(name string, tags map[string]string) error {
	measure := telegraf.MeasureInt(metric.prefix+"_"+name+"_occur", "value", 1).AddTags(rmBlankValue(tags))
	return metric.client.Write(measure)
}

//ReportCount 上报一个计数
func (metric *BasicMetric) ReportCount(name string, value int, tags map[string]string) error {
	measure := telegraf.MeasureInt(metric.prefix+"_"+name, "value", value).AddTags(rmBlankValue(tags))
	return metric.client.Write(measure)
}

//UseTime 上报一个耗时
func (metric *BasicMetric) UseTime(name string, time_elapse time.Duration, tags map[string]string) error {
	measure := telegraf.MeasureInt64(metric.prefix+"_"+name+"_usetime", "value", time_elapse.Nanoseconds()).AddTags(rmBlankValue(tags))
	return metric.client.Write(measure)
}

//Close
func (metric *BasicMetric) Close() error {
	return metric.client.Close()
}
