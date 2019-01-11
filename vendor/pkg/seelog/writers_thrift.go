// Date         : 2018/7/25 下午8:35
// Author       : zhangchaoyang
// Description  :
package seelog

import (
	"pkg/seelog/thrift"
	"pkg/seelog/flume_thrift"
	"net"
	"errors"
	"os"
	"bytes"
	"time"
)

type Status int32

const (
	STATUS_INIT  Status = 0
	STATUS_READY Status = 1
	STATUS_DEAD  Status = 2
)

// thriftWriter
type thriftWriter struct {
	selfname     []byte
	host         string
	port         string
	transport    thrift.TTransport
	thriftclient *flume_thrift.ThriftSourceProtocolClient
	status       Status //连接状态
	writeSelfIp  bool   //是否在每条日志前把本机的名字给加上
}

//NewThriftWriter
func NewThriftWriter(host string, port string, selfip bool) *thriftWriter {
	newWriter := new(thriftWriter)

	newWriter.selfname = []byte("[" + GetHostName() + "] ")
	newWriter.status = STATUS_INIT
	//创建一个物理连接
	tsocket, err := thrift.NewTSocket(net.JoinHostPort(host, port))
	if err == nil {
		tsocket.SetTimeout(time.Millisecond * 50)
		transportFactory := thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())
		//TLV 方式传输
		protocolFactory := thrift.NewTCompactProtocolFactory()
		//使用非阻塞io来传输
		newWriter.transport = transportFactory.GetTransport(tsocket)
		newWriter.thriftclient = flume_thrift.NewThriftSourceProtocolClientFactory(newWriter.transport, protocolFactory)
		if err := newWriter.transport.Open(); nil == err {
			newWriter.status = STATUS_READY
		}
		newWriter.writeSelfIp = selfip
	}

	return newWriter
}

func (thriftWriter *thriftWriter) Close() error {
	thriftWriter.status = STATUS_DEAD
	return thriftWriter.transport.Close()
}

func (thriftWriter *thriftWriter) Write(body []byte) (n int, err error) {
	n = 0
	if thriftWriter.status != STATUS_READY {
		err = errors.New("flume thrift connection closed")
		return
	}

	//日志前上本机ip
	var buffer bytes.Buffer
	if thriftWriter.writeSelfIp {
		buffer.Write(thriftWriter.selfname)
	}
	buffer.Write(body)

	event := flume_thrift.NewThriftFlumeEvent()
	event.Headers = make(map[string]string, 0)
	event.Body = buffer.Bytes()
	n = len(event.Body)

	thriftWriter.thriftclient.Append(event)

	return
}

//GetHostName 获取本机的hostname，如果获取不到就获取内网IP
func GetHostName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			os.Stderr.WriteString("Oops:" + err.Error())
			return ""
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
		os.Stderr.WriteString("can not get self ip")
		return ""
	} else {
		return host
	}
}
