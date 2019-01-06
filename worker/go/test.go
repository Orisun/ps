// Date         : 2018/12/7 10:58 AM
// Author       : zhangchaoyang
// Description  :
package main

import (
	communicate "ParameterServer/communicate/go"
	psclient "ParameterServer/client/go"
	"math/rand"
	"time"
)

func main0() {
	client := new(psclient.PsClient)
	defer client.ShutDown()
	
	const ParameterTotal = 10000 //多少个key
	SelfRange := communicate.Range{Begin: 1000, End: 9000}
	const ParameterCountOfKey = 10 //一个key对应多个参数
	client.Init("jobflume", 40010, int32(ParameterTotal), SelfRange)
	
	var WaitedPushMsgId int64 = 0
	var WaitedPullMsgId int64 = 0
	
	for i := 0; i < 1000000; i++ {
		//等待上上次的Push完成
		if WaitedPushMsgId > 0 {
			client.WaitPush(WaitedPushMsgId, time.Second)
		}
		
		//等待上上次的Pull完成
		if WaitedPullMsgId > 0 {
			client.WaitPull(WaitedPullMsgId, time.Second)
		}
		
		//Pull
		msgId := client.Pull()
		if i%3 == 0 {
			WaitedPullMsgId = msgId
		}
		
		//Push
		Values := []*communicate.Value{}
		for key := SelfRange.Begin; key < SelfRange.End; key++ {
			Value := make([]float64, ParameterCountOfKey)
			for i := 0; i < ParameterCountOfKey; i++ {
				Value[i] = rand.Float64()
			}
			Values = append(Values, &communicate.Value{Values: Value})
		}
		client.UpdateLocalRangedParameter(Values)
		msgId = client.Push()
		if i%3 == 0 {
			WaitedPushMsgId = msgId
		}
		
		time.Sleep(100 * time.Millisecond)
	}
}
