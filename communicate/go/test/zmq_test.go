// Date         : 2018/12/4 5:09 PM
// Author       : zhangchaoyang
// Description  :
package test

import (
	communicate "ParameterServer/communicate/go"
	"testing"
	"time"
	"ParameterServer/util"
)

func TestSend(t *testing.T) {
	van := communicate.GetZMQInstance(90010)
	sender := communicate.Node{Host: util.GetSelfHost()}
	receiver := communicate.Node{Host: "127.0.0.1"}
	msg := &communicate.Message{
		Sender:   &sender,
		Receiver: &receiver,
		Command:  communicate.Command_PING,
	}
	msgId := van.Send(msg)

	time.Sleep(1 * time.Second)
	van.WaitAck(msgId, 100*time.Millisecond)
}
