// Date         : 2018/10/10 下午1:36
// Author       : zhangchaoyang
// Description  :
package seelog

import (
	"fmt"
	"net"
	"time"
	"github.com/tevino/abool"
	"sync/atomic"
	"math/rand"
)

type UdpConnection struct {
	host            string
	port            int
	conn            net.Conn
	isConnecting    *abool.AtomicBool //正在尝试建立连接
	interval        int64             //连续2次尝试连接必须达到时间间隔interval，单位纳秒
	lastConnectTs   int64             //上次尝试建立连接的时间点，精度纳秒
	writeSuccessCnt int32             //连续写成功的次数
}

func NewUdpConnection(host string, port int) UdpConnection {
	self := UdpConnection{}
	self.host = host
	self.port = port
	self.isConnecting = abool.New()
	self.resetInterval()
	self.lastConnectTs = 0
	self.connect()
	return self
}

func (self *UdpConnection) resetInterval() {
	self.interval = 500 * time.Millisecond.Nanoseconds()
}

func (self *UdpConnection) connect() {
	now := time.Now().UnixNano()
	//与上次连接间隔时间太短，则放弃。以1%的概率绕过此规则
	if now-self.lastConnectTs < self.interval && rand.Float32() < 0.99 {
		return
	}
	//其他routine正在尝试连接，则放弃。
	if self.isConnecting.IsSet() {
		return
	}

	self.isConnecting.Set()
	defer self.isConnecting.UnSet()
	self.lastConnectTs = now
	//连接超时2秒
	self.conn, _ = net.DialTimeout("udp", fmt.Sprintf("%s:%d", self.host, self.port), 2*time.Second)
}

func (self *UdpConnection) Write(data []byte) (int, error) {
	//写超时设置为10毫秒
	self.conn.SetWriteDeadline(time.Now().Add(time.Millisecond * 10))
	n, err := self.conn.Write(data)
	if err != nil || n != len(data) {
		atomic.StoreInt32(&self.writeSuccessCnt, 0)
		//interval的上限为10分钟
		if self.interval < 10*time.Minute.Nanoseconds() {
			//如果写失败，则加大interval
			self.interval = int64(float64(self.interval) * 1.414)
		}
		//如果写失败，则尝试重连
		self.connect()
	} else {
		atomic.AddInt32(&self.writeSuccessCnt, 1)
		//连续写成功达到10次，则把interval置为初始值
		if atomic.LoadInt32(&self.writeSuccessCnt) >= 10 {
			atomic.StoreInt32(&self.writeSuccessCnt, 0)
			self.resetInterval()
		}
	}
	return n, err
}

func (self *UdpConnection) Close() error {
	return self.conn.Close()
}
