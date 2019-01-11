// Date         : 2018/9/30 下午4:11
// Author       : zhangchaoyang
// Description  :
package seelog

import (
	"bytes"
)

type udpwriter struct {
	writeSelfIp bool   //是否在每条日志前把本机的名字给加上
	selfname    []byte //本机名称
	conn        UdpConnection
}

func NewUdpWriter(host string, port int, writeSelfIp bool) (*udpwriter, error) {
	writer := new(udpwriter)
	writer.selfname = []byte("[" + GetHostName() + "] ")
	writer.conn = NewUdpConnection(host, port)
	writer.writeSelfIp = writeSelfIp

	return writer, nil
}

//Close 关闭udp连接
func (writer *udpwriter) Close() error {
	return writer.conn.Close()
}

//Write 每次write时都使用同一个connection。如果每次write都要新建connection则可能会达到文件句柄的上限socket: too many open files
func (writer *udpwriter) Write(body []byte) (n int, err error) {
	//日志前加上本机名称
	var buffer bytes.Buffer
	if writer.writeSelfIp {
		buffer.Write(writer.selfname)
	}
	buffer.Write(body)

	return writer.conn.Write(buffer.Bytes())
}
