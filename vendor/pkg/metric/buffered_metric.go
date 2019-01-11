// Date         : 2018/8/24 上午10:15
// Author       : zhangchaoyang
// Description  :
package metric

import (
	"github.com/mdaffin/go-telegraf"
	"time"
	"sync"
	"github.com/pkg/errors"
	"runtime"
)

type BufferedMetric struct {
	client        telegraf.Client
	prefix        string
	flushInterval time.Duration //攒够一段时间强制flush
	flushBatch    int           //攒够一定量强制flush

	running       bool
	addbuffer     chan telegraf.Measurement //把要上报的Measurement先缓存起来
	flushbuffer   []telegraf.Measurement
	shutDown      chan bool
	bufferClose   chan bool
	intervalClose chan bool
}

//NewBufferedMetric 创建带缓冲的Metric。flushInterval：攒够一段时间强制flush。flushBatch：攒够一定量强制flush。
func NewBufferedMetric(MetricPrefix string, flushInterval time.Duration, flushBatch int) Metric {
	if flushInterval <= time.Duration(0) {
		flushInterval = 500 * time.Millisecond
	}
	if flushBatch <= 0 {
		flushBatch = 50
	}
	if client, err := telegraf.NewUDP(INFLUX_SERVER_HOST + ":" + INFLUX_SERVER_PORT); err == nil {
		metric := &BufferedMetric{client: client, prefix: MetricPrefix, flushInterval: flushInterval, flushBatch: flushBatch}
		metric.running = false
		metric.addbuffer = make(chan telegraf.Measurement, 10000)
		metric.flushbuffer = make([]telegraf.Measurement, 0, metric.flushBatch)
		metric.shutDown = make(chan bool, 10)
		metric.bufferClose = make(chan bool)
		metric.intervalClose = make(chan bool)
		//run起来
		go metric.run()
		runtime.Gosched()
		return metric
	} else {
		return nil
	}
}

func (metric *BufferedMetric) run() {
	metric.running = true

	lock := sync.Mutex{}

	go func() {
		ticker := time.NewTicker(metric.flushInterval)
		defer ticker.Stop()
		stop := false
		for ; !stop; {
			select {
			case <-ticker.C:
				lock.Lock()
				metric.client.WriteAll(metric.flushbuffer)
				metric.flushbuffer = make([]telegraf.Measurement, 0, metric.flushBatch)
				lock.Unlock()
			case <-metric.shutDown:
				metric.shutDown <- true
				metric.client.WriteAll(metric.flushbuffer)
				stop = true
			}
		}
		metric.intervalClose <- true
	}()

	go func() {
		stop := false
		for ; !stop; {
			select {
			case m := <-metric.addbuffer:
				lock.Lock()
				metric.flushbuffer = append(metric.flushbuffer, m)
				if len(metric.flushbuffer) >= metric.flushBatch {
					metric.client.WriteAll(metric.flushbuffer)
					metric.flushbuffer = make([]telegraf.Measurement, 0, metric.flushBatch)
				}
				lock.Unlock()
			case <-metric.shutDown:
				metric.shutDown <- true
				metric.client.WriteAll(metric.flushbuffer)
				stop = true
			}
		}
		metric.bufferClose <- true
	}()
}

func (metric *BufferedMetric) flush() {
	close(metric.addbuffer)
	//flushbuffer := make([]telegraf.Measurement, 0, metric.flushBatch)
	for ele := range metric.addbuffer {
		if len(metric.flushbuffer) >= metric.flushBatch {
			metric.client.WriteAll(metric.flushbuffer)
			metric.flushbuffer = make([]telegraf.Measurement, 0, metric.flushBatch)
		}
		metric.flushbuffer = append(metric.flushbuffer, ele)
	}
	if len(metric.flushbuffer) > 0 {
		metric.client.WriteAll(metric.flushbuffer)
	}
}

//Close
func (metric *BufferedMetric) Close() error {
	if !metric.running {
		return nil
	}
	metric.running = false
	metric.shutDown <- true
	metric.flush()
	//等run()里面的2个子协和结束
	<-metric.intervalClose
	<-metric.bufferClose
	return metric.client.Close()
}

//OccurOnce 发生一次，上报计数1
func (metric *BufferedMetric) OccurOnce(name string, tags map[string]string) error {
	if metric.running {
		measure := telegraf.MeasureInt(metric.prefix+"_"+name+"_occur", "value", 1).AddTags(tags)
		metric.addbuffer <- measure
		return nil
	} else {
		return errors.New("metric have stoped")
	}
}

//ReportCount 上报一个计数
func (metric *BufferedMetric) ReportCount(name string, value int, tags map[string]string) error {
	if metric.running {
		measure := telegraf.MeasureInt(metric.prefix+"_"+name, "value", value).AddTags(tags)
		metric.addbuffer <- measure
		return nil
	} else {
		return errors.New("metric have stoped")
	}
}

//UseTime 上报一个耗时
func (metric *BufferedMetric) UseTime(name string, time_elapse time.Duration, tags map[string]string) error {
	if metric.running {
		measure := telegraf.MeasureInt64(metric.prefix+"_"+name+"_usetime", "value", time_elapse.Nanoseconds()).AddTags(tags)
		metric.addbuffer <- measure
		return nil
	} else {
		return errors.New("metric have stoped")
	}
}
