// Date         : 2018/12/4 3:19 PM
// Author       : zhangchaoyang
// Description  :
package _go

import (
	communicate "ParameterServer/communicate/go"
	"ParameterServer/util"
	"github.com/facebookarchive/inmem"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type PsClient struct {
	node           communicate.Node     //本机标识
	van            communicate.Van      //通信工具
	servers        []*communicate.Node  //所有server
	ParameterTotal int32                //模型参数的总数
	keyRange       *communicate.Range   //本地负责训练的参数key的范围
	parameters     []*communicate.Value //所有参数的值
	parameterLock  sync.RWMutex         //读写parameters前要加锁
	pullWaitCache  inmem.Cache
}

func (self *PsClient) Init(manager string, port int, ParameterTotal int32, KeyRange communicate.Range) {
	if KeyRange.Begin < 0 || KeyRange.End <= KeyRange.Begin || KeyRange.End > ParameterTotal {
		util.Log.Errorf("invalid key range [%d, %d), total parameter %d", KeyRange.Begin, KeyRange.End, ParameterTotal)
		util.Log.Flush()
		syscall.Exit(0)
	}

	self.node = communicate.Node{
		Index: -1,
		Host:  util.GetSelfHost(),
		Ready: true,
		Role:  communicate.Role_WORKER,
	}
	self.van = communicate.GetZMQInstance(port)
	self.ParameterTotal = ParameterTotal
	self.servers = []*communicate.Node{}
	self.keyRange = &communicate.Range{
		Begin: KeyRange.Begin, End: KeyRange.End,
	}
	self.parameters = make([]*communicate.Value, ParameterTotal)
	for i := 0; i < int(ParameterTotal); i++ {
		self.parameters[i] = &communicate.Value{}
	}
	self.parameterLock = sync.RWMutex{}
	self.pullWaitCache = inmem.NewLocked(1000)

	//向ServerManager注册worker，获取Servers
	ManagerNode := communicate.Node{Host: manager, Ready: true, Role: communicate.Role_SERVER_MANAGER}
	msg := &communicate.Message{
		Sender:   &self.node,
		Receiver: &ManagerNode,
		Command:  communicate.Command_ADD_WORKER,
	}
	msgId := self.van.Send(msg)

	//开始接收server集群的变更信息
	go func() {
		self.receiveMsg()
	}()

	//等worker注册成功
	if self.van.WaitAck(msgId, time.Second*10) == nil {
		util.Log.Errorf("regist worker failed")
		util.Log.Flush()
		syscall.Exit(1)
	}
	util.Log.Debugf("regist worker ok")

	//必须等待取到Servers，否则程序就退出
	for retry := 0; retry < 10; retry++ {
		if len(self.servers) == 0 {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	if len(self.servers) == 0 {
		util.Log.Errorf("could not get servers")
		util.Log.Flush()
		syscall.Exit(1)
	}

	ch := make(chan os.Signal, 1)                                                       //创建管道，用于接收信号
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM) //注册信号
	go func() {
		sig := <-ch
		util.Log.Infof("receive signal %v", sig)
		self.ShutDown()
		util.Log.Flush()
		os.Exit(0)
	}()

	util.Log.Infof("init ps client ok")
}

//GetAllParameter 获取模型的所有参数
func (self *PsClient) GetAllParameter() []*communicate.Value {
	self.parameterLock.RLock()
	defer self.parameterLock.RUnlock()
	return self.parameters
}

//UpdateLocalRangedParameter 更新本机负责训练的那部分参数
func (self *PsClient) UpdateLocalRangedParameter(Values []*communicate.Value) bool {
	if len(Values) != int(self.keyRange.End-self.keyRange.Begin) {
		util.Log.Errorf("update value count %d not match self self key range [%d, %d)", len(Values),
			self.keyRange.Begin, self.keyRange.End)
		return false
	}
	self.parameterLock.Lock()
	defer self.parameterLock.Unlock()
	for i, vs := range Values {
		self.parameters[i+int(self.keyRange.Begin)] = vs
	}
	return true
}

func (self *PsClient) UpdateLocalParameter(Values []*communicate.Value) bool {
	if len(Values) != int(self.ParameterTotal) {
		util.Log.Errorf("update value count %d not match total parameter count %d", len(Values),
			self.ParameterTotal)
		return false
	}
	self.parameterLock.Lock()
	defer self.parameterLock.Unlock()
	for i, vs := range Values {
		self.parameters[i] = vs
	}
	return true
}

//Pull 从PS集群上拉取全Range的参数。异步等待数据返回，更新本地的参数值
func (self *PsClient) Pull() int64 {
	if len(self.servers) == 0 {
		util.Log.Errorf("no available server")
		util.Log.Flush()
		syscall.Exit(1)
	}

	//随机获取一台server
	idx := rand.Intn(len(self.servers))
	server := self.servers[idx]
	util.Log.Infof("choose server %s", server.Host)
	//pull数据
	msg := &communicate.Message{
		Sender:   &self.node,
		Receiver: server,
		Command:  communicate.Command_PULL,
		KeyRange: &communicate.Range{Begin: 0, End: self.ParameterTotal},
	}
	msgId := self.van.Send(msg)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	self.pullWaitCache.Add(msgId, wg, time.Now().Add(time.Minute*10))

	return msgId
}

//Push 更新PS上的参数
func (self *PsClient) Push() int64 {
	if len(self.servers) == 0 {
		util.Log.Errorf("no available server")
		return -1
	}

	self.parameterLock.RLock()
	defer self.parameterLock.RUnlock()
	Values := self.parameters[self.keyRange.Begin:self.keyRange.End]
	if len(Values) != int(self.keyRange.End-self.keyRange.Begin) {
		util.Log.Errorf("update value count %d not match self self key range [%d, %d)", len(Values),
			self.keyRange.Begin, self.keyRange.End)
		return -1
	}
	//随机获取一台server
	idx := rand.Intn(len(self.servers))
	server := self.servers[idx]
	util.Log.Infof("choose server %s", server.Host)
	//push数据
	msg := &communicate.Message{
		Sender:   &self.node,
		Receiver: server,
		KeyRange: self.keyRange,
		Values:   Values,
		Command:  communicate.Command_PUSH,
	}
	msgId := self.van.Send(msg)
	return msgId
}

//Inc 把参数增量push给PS集群
func (self *PsClient) Inc(Deltas []*communicate.Value) int64 {
	if len(self.servers) == 0 {
		util.Log.Errorf("no available server")
		return -1
	}

	self.parameterLock.RLock()
	defer self.parameterLock.RUnlock()
	if len(Deltas) != int(self.keyRange.End-self.keyRange.Begin) {
		util.Log.Errorf("add delta value count %d not match self self key range [%d, %d)", len(Deltas),
			self.keyRange.Begin, self.keyRange.End)
		return -1
	}
	//随机获取一台server
	idx := rand.Intn(len(self.servers))
	server := self.servers[idx]
	util.Log.Infof("choose server %s", server.Host)
	//push数据
	msg := &communicate.Message{
		Sender:   &self.node,
		Receiver: server,
		KeyRange: self.keyRange,
		Values:   Deltas,
		Command:  communicate.Command_INC,
	}
	msgId := self.van.Send(msg)
	return msgId
}

//WaitPull 等待某一次的Pull完成。如果超时则返回false。如果不设超时则最多等5分钟
func (self *PsClient) WaitPull(MessageId int64, TimeOut time.Duration) bool {
	if v, exists := self.pullWaitCache.Get(MessageId); exists {
		wg := v.(*sync.WaitGroup)
		if TimeOut == 0 {
			TimeOut = 5 * time.Minute
		}
		if util.WaitTimeout(wg, TimeOut) {
			return true
		} else {
			util.Log.Errorf("wait pull ack for msg %d timeout", MessageId)
			return false
		}
	}
	return true
}

//WaitPush 等待某一次的Push完成。如果超时则返回false
func (self *PsClient) WaitPush(MessageId int64, TimeOut time.Duration) bool {
	msg := self.van.WaitAck(MessageId, TimeOut)
	if msg == nil {
		return false
	}
	return msg.RespondSuccess
}

//WaitInc 等待某一次的Inc完成。如果超时则返回false
func (self *PsClient) WaitInc(MessageId int64, TimeOut time.Duration) bool {
	msg := self.van.WaitAck(MessageId, TimeOut)
	if msg == nil {
		return false
	}
	return msg.RespondSuccess
}

func (self *PsClient) dealMsg(msg *communicate.Message) {
	if msg == nil {
		return
	}
	if msg.Command == communicate.Command_SERVER_CHANGE {
		self.servers = msg.ServerClusterInfo.Servers
		util.Log.Infof("server count change to %d", len(self.servers))
	} else if msg.Command == communicate.Command_PULL_ACK {
		msgId := msg.RespondMsgId
		if msgId > 0 {
			if msg.RespondSuccess {
				self.parameterLock.Lock()
				defer self.parameterLock.Unlock()
				if len(msg.Values) == len(self.parameters) {
					for i, vs := range msg.Values {
						if len(self.parameters[i].Values) == 0 {
							if len(vs.Values) > 0 {
								self.parameters[i].Values = make([]float64, len(vs.Values))
							}
						}
						if len(vs.Values) > 0 {
							for j, v := range vs.Values {
								self.parameters[i].Values[j] = v
							}
						} else {
							util.Log.Errorf("pull 0 values of key %d from ps", i)
						}
					}
					util.Log.Debugf("update %d local parameter from ps", len(msg.Values))
				} else {
					util.Log.Errorf("respond values length %d not match self parameter length %d", len(msg.Values), len(self.parameters))
				}
				if v, exists := self.pullWaitCache.Get(msgId); exists {
					wg := v.(*sync.WaitGroup)
					wg.Done()
				}
			}
		}
	}
}

func (self *PsClient) receiveMsg() {
	for {
		msg := self.van.PeekOneMessage()
		go func(msg *communicate.Message) { //异步处理接收到的消息
			self.dealMsg(msg)
		}(msg)
	}
}

//ShutDown 关闭网络连接
func (self *PsClient) ShutDown() {
	self.van.Close()
}
