// Date         : 2018/11/27 5:30 PM
// Author       : zhangchaoyang
// Description  :
package _go

import "time"

const (
	sendMsgKeepTime      = time.Minute * 5 //发出去的消息要保留5分钟，因为对方可能会要求重发
	sendMsgKeepCount     = 100             //发出去的消息最多保留100条
	receiveWaitKeepTime  = time.Minute * 5
	receiveWaitKeepCount = 100
	ackKeepTime          = time.Minute * 5
	ackKeepCount         = 100
	msgKeepCount         = 1000
)

type Van interface {
	//发送消息，返回消息ID
	Send(msg *Message) int64
	//重发消息
	ReSend(msgId int64) int64
	//等待响应。TimeOut=0表示不设置超时
	WaitAck(MsgId int64, TimeOut time.Duration) *Message
	//PeekOneMessage 取出一条消息，如果没有请求则会一直阻塞
	PeekOneMessage() *Message
	//关闭网络连接
	Close()
}
