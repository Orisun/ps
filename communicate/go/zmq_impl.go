// Date         : 2018/11/29 2:49 PM
// Author       : zhangchaoyang
// Description  :
package _go

import (
	"ParameterServer/util"
	"fmt"
	"github.com/facebookarchive/inmem"
	"github.com/golang/protobuf/proto"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"zmq4"
)

var (
	initZmqOnce sync.Once
	zmq         *ZMQ
)

//ZMQ 实现Van接口
type ZMQ struct {
	port          int
	receiveSocket *zmq4.Socket

	sendHistory    inmem.Cache //发出去的消息存在sendHistory
	waitedMsgCache inmem.Cache
	msgChannel     chan Message
	receiveWait    inmem.Cache //等待ack时，每一个等待对应一个WaitGroup

	closed bool
}

//GetZMQInstance 单例
func GetZMQInstance(port int) *ZMQ {
	initZmqOnce.Do(func() {
		if zmq == nil {
			//REQ--REP是同步的，即必须收到reply后才能send下一条消息；DEALER--ROUTER是完全异步的
			receiveSocket, _ := zmq4.NewSocket(zmq4.ROUTER)
			if err := receiveSocket.Bind(fmt.Sprintf("tcp://*:%d", port)); err != nil {
				util.Log.Criticalf("bind to port %d failed: %v", port, err)
				util.Log.Flush()
				syscall.Exit(1)
			}
			time.Sleep(100 * time.Millisecond) //等待receiveSocket监听成功
			sendHistory := inmem.NewLocked(sendMsgKeepCount)
			receiveWait := inmem.NewLocked(receiveWaitKeepCount)
			ackCache := inmem.NewLocked(ackKeepCount)
			zmq = &ZMQ{
				port:           port,
				receiveSocket:  receiveSocket,
				sendHistory:    sendHistory,
				receiveWait:    receiveWait,
				waitedMsgCache: ackCache,
				msgChannel:     make(chan Message, msgKeepCount), //通道已满时再add就会阻塞
				closed:         false,
			}

			zmq.receive() //另起一个协程，开始接收消息
		}
	})
	return zmq
}

//Send
func (self *ZMQ) Send(msg *Message) int64 {
	msg.Id = int64(util.GetSnowNode().Generate()) //临发送的时候才给MessageID赋值

	sendSocket, _ := zmq4.NewSocket(zmq4.DEALER)
	defer sendSocket.Close()
	sendSocket.Connect(fmt.Sprintf("tcp://%s:%d", msg.Receiver.Host, self.port))

	if b, err := proto.Marshal(msg); err == nil {
		wg := sync.WaitGroup{}
		wg.Add(1)
		self.receiveWait.Add(msg.Id, &wg, time.Now().Add(receiveWaitKeepTime))
		sendSocket.SendBytes(b, 0)
		self.sendHistory.Add(msg.Id, msg, time.Now().Add(sendMsgKeepTime))
		if msg.Command != Command_PING_ACK && msg.Command != Command_PING {
			util.Log.Debugf("send command %s to %s:%d, message id %d, resp message id %d", msg.Command.String(), msg.Receiver.Host, self.port, msg.Id, msg.RespondMsgId)
		}
	} else {
		util.Log.Errorf("marshal message %d failed", msg.Id)
	}
	return msg.Id
}

//ReSend 重发。如果还能从缓存里找到msgId对应的消息体，则返回msgId；否则返回0
func (self *ZMQ) ReSend(msgId int64) int64 {
	if v, exists := self.sendHistory.Get(msgId); exists {
		msg := v.(*Message)
		self.Send(msg)
		return msgId
	} else {
		util.Log.Errorf("ask resend message %d, but not cache it", msgId)
		return 0
	}
}

//receive 开启接收消息的协程
func (self *ZMQ) receive() {
	go func() {
		chSignal := make(chan os.Signal, 100)
		signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
		for {
			select {
			case <-chSignal:
				self.Close()
				break
			default:
				if !self.closed {
					if request, err := self.receiveSocket.RecvMessageBytes(0); err == nil {
						var msg Message
						if err := proto.Unmarshal(request[1], &msg); err == nil {
							if msg.Command != Command_PING_ACK && msg.Command != Command_PING {
								util.Log.Debugf("receive command %s from %s", msg.Command.String(), msg.Sender.Host)
							}
							if msg.RespondMsgId > 0 {
								//util.Log.Debugf("msg %d got ack", msg.RespondMsgId)
								self.waitedMsgCache.Add(msg.RespondMsgId, &msg, time.Now().Add(ackKeepTime))
								if v, exists := self.receiveWait.Get(msg.RespondMsgId); exists {
									wg := v.(*sync.WaitGroup)
									wg.Done()
									//util.Log.Debugf("msg %d wait through", msg.RespondMsgId)
								} else {
									//有2种情况程序会走到这里：
									//1. msg从waitedMsgCache中溢出
									//2. server manager广播的一些消息，在本server中找不到对应的原消息
								}
							}
							self.msgChannel <- msg
						} else {
							util.Log.Errorf("unmarshal message failed")
						}
					} else {
						util.Log.Errorf("socket receive error:%v", err)
						if zmq4.AsErrno(err) == zmq4.Errno(syscall.EINTR) {
							break
						}
					}
				}
			}
		}
		util.Log.Criticalf("stop receive message")
	}()
}

//WaitAck 等待响应。TimeOut=0表示不设置超时（如果不设超时则最多等5分钟）。只有PULL、PUSH、UPDATE这3类消息可以被wait
func (self *ZMQ) WaitAck(MsgId int64, TimeOut time.Duration) *Message {
	if v, exists := self.receiveWait.Get(MsgId); exists {
		wg := v.(*sync.WaitGroup)

		if TimeOut == 0 {
			TimeOut = ackKeepTime
		}

		//util.Log.Debugf("begin wait for msg %d", MsgId)
		if util.WaitTimeout(wg, TimeOut) {
			//util.Log.Debugf("msg %d wait through", MsgId)
			if data, ok := self.waitedMsgCache.Get(MsgId); ok {
				//util.Log.Debugf("got response for msg %d", MsgId)
				msg := data.(*Message)
				return msg
			}
		} else {
			util.Log.Errorf("wait ack for msg %d timeout", MsgId)
		}
	}
	util.Log.Errorf("no ack for message %d", MsgId)
	return nil
}

//PeekOneMessage 取出一条消息，如果没有请求则会一直阻塞
func (self *ZMQ) PeekOneMessage() *Message {
	msg := <-self.msgChannel
	return &msg
}

func (self *ZMQ) Close() {
	self.closed = true
	if self.receiveSocket != nil {
		self.receiveSocket.Close()
	}
}
