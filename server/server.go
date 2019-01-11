// Date         : 2018/11/30 10:56 AM
// Author       : zhangchaoyang
// Description  :
package server

import (
	communicate "ParameterServer/communicate/go"
	"ParameterServer/util"
	"ParameterServer/util/data_struct"
	"ParameterServer/util/distribution"
	"github.com/fanliao/go-concurrentMap"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	index         int32
	node          communicate.Node
	manager       string
	keyRange      communicate.Range
	clusterInfo   communicate.Cluster
	van           communicate.Van
	parameterMap  *concurrent.ConcurrentMap
	updateTimes   int
	paramInitFunc distribution.Distribution
}

//getFromLocal 从本地获取参数值
func (self *Server) getFromLocal(KeyRange *communicate.Range) (map[int32]*communicate.Value, bool) {
	success := true
	data := make(map[int32]*communicate.Value)
	if KeyRange == nil {
		util.Log.Errorf("KeyRange is nil")
		return data, success
	}

	for key := KeyRange.Begin; key < KeyRange.End; key++ {
		if v, err := self.parameterMap.Get(key); err == nil && v != nil {
			values := v.(*communicate.Value)
			data[key] = values
		}
	}
	if len(data) < int(KeyRange.End-KeyRange.Begin) {
		util.Log.Warnf("request %d keys, but only response %d keys", int(KeyRange.End-KeyRange.Begin), len(data))
	}
	//util.Log.Debugf("get data from local")
	return data, success
}

//updateToLocal 更新本地参数值。参数KeyRange可能不是self.keyRange的子集，因self.keyRange没有包含self.Masrters上的KeyRange
func (self *Server) updateToLocal(KeyRange *communicate.Range, Values []*communicate.Value) bool {
	if KeyRange == nil {
		util.Log.Errorf("KeyRange is nil")
		return true
	}
	if int(KeyRange.End-KeyRange.Begin) != len(Values) {
		util.Log.Errorf("key range [%d, %d) not match value length %d, will not update parameter", KeyRange.Begin, KeyRange.End, len(Values))
		return false
	}

	for key := KeyRange.Begin; key < KeyRange.End; key++ {
		value := Values[key-KeyRange.Begin]
		self.parameterMap.Put(key, value)
	}

	//KeyRange是self.keyRange的子集，说明在更新本地的主参数（非备份参数）
	if KeyRange.Begin >= self.keyRange.Begin && KeyRange.End <= self.keyRange.End {
		self.updateTimes += 1
		//updae执行若干次后才push给slave
		if self.updateTimes%5 == 0 {
			if !self.pushToSlaves() {
				util.Log.Errorf("push data to slaves failed")
			}
		}
	}
	//util.Log.Debugf("updae local data")
	return true
}

//addToLocal 让本地的参数加一个增量。参数KeyRange可能不是self.keyRange的子集，因self.keyRange没有包含self.Masrters上的KeyRange
func (self *Server) addToLocal(KeyRange *communicate.Range, Deltas []*communicate.Value) bool {
	if KeyRange == nil {
		util.Log.Errorf("KeyRange is nil")
		return true
	}
	if int(KeyRange.End-KeyRange.Begin) != len(Deltas) {
		util.Log.Errorf("key range [%d, %d) not match value length %d, will not update parameter", KeyRange.Begin, KeyRange.End, len(Deltas))
		return false
	}

	for key := KeyRange.Begin; key < KeyRange.End; key++ {
		delta := Deltas[key-KeyRange.Begin]
		if v, err := self.parameterMap.Get(key); err == nil && v != nil {
			oldV := v.(*communicate.Value)
			values := make([]float64, len(oldV.Values))
			for i, ov := range oldV.Values {
				nv := ov + delta.Values[i]
				values[i] = nv
			}
			self.parameterMap.Put(key, &communicate.Value{Values: values})
		} else {
			util.Log.Errorf("no value for key %d", key)
		}
	}

	//KeyRange是self.keyRange的子集，说明在更新本地的主参数（非备份参数）
	if KeyRange.Begin >= self.keyRange.Begin && KeyRange.End <= self.keyRange.End {
		self.updateTimes += 1
		//updae执行若干次后才push给slave
		if self.updateTimes%5 == 0 {
			if !self.pushToSlaves() {
				util.Log.Errorf("push data to slaves failed")
			}
		}
	}
	//util.Log.Debugf("updae local data")
	return true
}

func (self *Server) pullFromMasters() bool {
	success := true
	ServerCount := len(self.clusterInfo.Servers)
	if len(self.clusterInfo.Masters) > 0 {
		masters := self.clusterInfo.Masters[self.index]
		if masters != nil {
			for _, master := range masters.Nodes {
				masterIndex := master.Index
				end := self.clusterInfo.RangeEnd[masterIndex]
				var begin int32
				if masterIndex == 0 {
					begin = 0
				} else {
					begin = self.clusterInfo.RangeEnd[(ServerCount+int(masterIndex)-1)%ServerCount]
				}
				r := &communicate.Range{Begin: begin, End: end}
				values, ok := self.pull(r, *master)
				success = success && ok
				if ok {
					for key := begin; key < end; key++ {
						self.parameterMap.Put(key, values[key-begin])
					}
				}
			}
		}
	}
	//util.Log.Debugf("pull data from masters")
	return success
}

func (self *Server) pushToSlaves() bool {
	success := true
	if len(self.clusterInfo.Slaves) > 0 {
		slaves := self.clusterInfo.Slaves[self.index]
		values, ok := self.get(&self.keyRange)
		success = success && ok
		if ok {
			if slaves != nil {
				for _, slave := range slaves.Nodes {
					ok := self.push(&self.keyRange, values, *slave)
					success = success && ok
				}
			}
		}
	}
	//util.Log.Debugf("push data to slaves")
	return success
}

//rangeOwnedByMe 命中自己的范围，或命中自己的masters的范围
func (self *Server) rangeOwnedByMe(KeyRange *communicate.Range) bool {
	if KeyRange.Begin >= self.keyRange.Begin && KeyRange.End <= self.keyRange.End {
		return true
	}
	if self.clusterInfo.Masters != nil && len(self.clusterInfo.Masters) > 0 && self.clusterInfo.RangeEnd != nil && len(self.clusterInfo.RangeEnd) > 0 {
		for _, master := range self.clusterInfo.Masters[self.index].Nodes {
			var beginRange int32
			if master.Index > 0 {
				beginRange = self.clusterInfo.RangeEnd[master.Index-1]
			}
			endRange := self.clusterInfo.RangeEnd[master.Index]
			if KeyRange.Begin >= beginRange && KeyRange.End <= endRange {
				return true
			}
		}
	}
	return false
}

//get 从集群中获取参数
func (self *Server) get(KeyRange *communicate.Range) ([]*communicate.Value, bool) {
	if KeyRange == nil {
		util.Log.Errorf("KeyRange is nil")
		return []*communicate.Value{}, true
	}
	util.Log.Debugf("will get key range [%d, %d)", KeyRange.Begin, KeyRange.End)
	if KeyRange.End <= KeyRange.Begin {
		util.Log.Errorf("invalid key range [%d, %d)", KeyRange.Begin, KeyRange.End)
		return []*communicate.Value{}, false
	}

	success := true
	rect := concurrent.NewConcurrentMap()
	//if KeyRange.Begin >= self.keyRange.Begin && KeyRange.End <= self.keyRange.End {
	if self.rangeOwnedByMe(KeyRange) {
		dict, ok := self.getFromLocal(KeyRange)
		success = success && ok
		for k, v := range dict {
			rect.Put(k, v)
		}
	} else {
		target := make(map[int]*communicate.Range)
		BeginIdx := data_struct.BinarySearchSmaller(self.clusterInfo.RangeEnd, KeyRange.Begin) + 1 //从第几段开始(包含)
		EndIdx := data_struct.BinarySearchBigger(self.clusterInfo.RangeEnd, KeyRange.End)          //结束于第几段(包含)
		util.Log.Debugf("will pull data from [%d, %d] seg", BeginIdx, EndIdx)
		if BeginIdx > EndIdx {
			if len(self.clusterInfo.RangeEnd) > 0 {
				util.Log.Errorf("BeginIdx %d more than EndIdx %d", BeginIdx, EndIdx)
			}
		} else if BeginIdx < EndIdx {
			target[BeginIdx] = &communicate.Range{
				Begin: KeyRange.Begin,
				End:   self.clusterInfo.RangeEnd[BeginIdx],
			}
			//util.Log.Debugf("begin %d %d %d", BeginIdx, keyRange.Begin, self.clusterInfo.RangeEnd[BeginIdx])
			target[EndIdx] = &communicate.Range{
				Begin: self.clusterInfo.RangeEnd[EndIdx-1],
				End:   KeyRange.End,
			}
			//util.Log.Debugf("end %d %d %d", EndIdx, self.clusterInfo.RangeEnd[EndIdx-1], self.keyRange.End)
			for i := BeginIdx + 1; i < EndIdx; i++ {
				target[i] = &communicate.Range{
					Begin: self.clusterInfo.RangeEnd[i-1],
					End:   self.clusterInfo.RangeEnd[i],
				}
				//util.Log.Debugf("i %d %d %d", i, self.clusterInfo.RangeEnd[i-1], self.clusterInfo.RangeEnd[i])
			}
		} else {
			target[BeginIdx] = KeyRange
		}

		if len(target) > 0 {
			wg := sync.WaitGroup{}
			wg.Add(len(target))
			for idx, scope := range target {
				go func(idx int, scope *communicate.Range) {
					defer wg.Done()
					if scope.Begin >= self.keyRange.Begin && scope.End <= self.keyRange.End {
						dict, ok := self.getFromLocal(scope)
						success = success && ok
						for k, v := range dict {
							rect.Put(k, v)
						}
					} else {
						if idx < len(self.clusterInfo.Servers) {
							//util.Log.Debugf("will pull [%d, %d) from server %s", scope.Begin, scope.End, self.clusterInfo.servers[idx].Host)
							datas, ok := self.pull(scope, *self.clusterInfo.Servers[idx])
							success = success && ok
							for i, data := range datas {
								rect.Put(scope.Begin+int32(i), data)
							}
						}
					}
				}(idx, scope)
			}
			wg.Wait()
		}
	}

	Values := make([]*communicate.Value, 0, int(KeyRange.End-KeyRange.Begin))
	for key := KeyRange.Begin; key < KeyRange.End; key++ {
		if v, err := rect.Get(key); err == nil && v != nil {
			Values = append(Values, v.(*communicate.Value))
		} else {
			Values = append(Values, &communicate.Value{})
		}
	}
	//util.Log.Debugf("get %d values [%d, %d) from cluster", len(Values), KeyRange.Begin, KeyRange.End)
	return Values, success
}

//update 更新集群上的参数
func (self *Server) update(KeyRange *communicate.Range, Values []*communicate.Value) bool {
	if KeyRange == nil {
		util.Log.Errorf("KeyRange is nil")
		return true
	}
	if KeyRange.End <= KeyRange.Begin {
		util.Log.Errorf("invalid key range [%d, %d)", KeyRange.Begin, KeyRange.End)
		return false
	}
	if int(KeyRange.End-KeyRange.Begin) != len(Values) {
		util.Log.Errorf("key range [%d, %d) not match values count %d", KeyRange.Begin, KeyRange.End, len(Values))
		return false
	}

	success := true
	if KeyRange.Begin >= self.keyRange.Begin && KeyRange.End <= self.keyRange.End {
		ok := self.updateToLocal(KeyRange, Values)
		success = success && ok
	} else {
		target := make(map[int]communicate.Range)
		RangeBegin := KeyRange.Begin
		RangeEnd := KeyRange.End
		BeginIdx := data_struct.BinarySearchSmaller(self.clusterInfo.RangeEnd, RangeBegin) + 1 //从第几段开始
		EndIdx := data_struct.BinarySearchBigger(self.clusterInfo.RangeEnd, RangeEnd)          //结束于第几段
		if BeginIdx > EndIdx {
			if len(self.clusterInfo.RangeEnd) > 0 {
				util.Log.Errorf("BeginIdx %d more than EndIdx %d", BeginIdx, EndIdx)
			}
		} else if BeginIdx < EndIdx {
			target[BeginIdx] = communicate.Range{
				Begin: KeyRange.Begin,
				End:   self.clusterInfo.RangeEnd[BeginIdx],
			}
			//util.Log.Debugf("push range [%d, %d) to %d server %s", KeyRange.Begin, self.clusterInfo.RangeEnd[BeginIdx], BeginIdx, self.clusterInfo.Servers[BeginIdx].Host)
			target[EndIdx] = communicate.Range{
				Begin: self.clusterInfo.RangeEnd[EndIdx-1],
				End:   KeyRange.End,
			}
			//util.Log.Debugf("push range [%d, %d) to %d server %s", self.clusterInfo.RangeEnd[EndIdx-1], KeyRange.End, EndIdx, self.clusterInfo.Servers[EndIdx].Host)
			for i := BeginIdx + 1; i < EndIdx; i++ {
				target[i] = communicate.Range{
					Begin: self.clusterInfo.RangeEnd[i-1],
					End:   self.clusterInfo.RangeEnd[i],
				}
				//util.Log.Debugf("push range [%d, %d) to %d server %s", self.clusterInfo.RangeEnd[i-1], self.clusterInfo.RangeEnd[i], i, self.clusterInfo.Servers[i].Host)
			}
		} else {
			target[BeginIdx] = *KeyRange
		}

		if len(target) > 0 {
			wg := sync.WaitGroup{}
			wg.Add(len(target))
			for idx, scope := range target {
				go func(idx int, scope communicate.Range) {
					defer wg.Done()
					if idx < len(self.clusterInfo.Servers) {
						//util.Log.Debugf("push idx %d scope [%d, %d) key range [%d, %d) values range [%d, %d) values count %d\n", idx, scope.Begin, scope.End, KeyRange.Begin, KeyRange.End, scope.Begin-KeyRange.Begin, scope.End-KeyRange.Begin, len(Values))
						values := Values[scope.Begin-KeyRange.Begin : scope.End-KeyRange.Begin]
						receiver := *self.clusterInfo.Servers[idx]
						if scope.Begin >= self.keyRange.Begin && scope.End <= self.keyRange.End {
							ok := self.updateToLocal(&scope, values)
							success = success && ok
						} else {
							ok := self.push(&scope, values, receiver)
							success = success && ok
						}
					}
				}(idx, scope)
			}
			wg.Wait()
		}
	}
	//util.Log.Debugf("update [%d, %d) to cluster", KeyRange.Begin, KeyRange.End)
	return success
}

//add 让集群上的参数加一个增量
func (self *Server) add(KeyRange *communicate.Range, Deltas []*communicate.Value) bool {
	if KeyRange == nil {
		util.Log.Errorf("KeyRange is nil")
		return true
	}
	if KeyRange.End <= KeyRange.Begin {
		util.Log.Errorf("invalid key range [%d, %d)", KeyRange.Begin, KeyRange.End)
		return false
	}
	if int(KeyRange.End-KeyRange.Begin) != len(Deltas) {
		util.Log.Errorf("key range [%d, %d) not match values count %d", KeyRange.Begin, KeyRange.End, len(Deltas))
		return false
	}

	success := true
	if KeyRange.Begin >= self.keyRange.Begin && KeyRange.End <= self.keyRange.End {
		ok := self.addToLocal(KeyRange, Deltas)
		success = success && ok
	} else {
		target := make(map[int]communicate.Range)
		RangeBegin := KeyRange.Begin
		RangeEnd := KeyRange.End
		BeginIdx := data_struct.BinarySearchSmaller(self.clusterInfo.RangeEnd, RangeBegin) + 1 //从第几段开始
		EndIdx := data_struct.BinarySearchBigger(self.clusterInfo.RangeEnd, RangeEnd)          //结束于第几段
		if BeginIdx > EndIdx {
			if len(self.clusterInfo.RangeEnd) > 0 {
				util.Log.Errorf("BeginIdx %d more than EndIdx %d", BeginIdx, EndIdx)
			}
		} else if BeginIdx < EndIdx {
			target[BeginIdx] = communicate.Range{
				Begin: KeyRange.Begin,
				End:   self.clusterInfo.RangeEnd[BeginIdx],
			}
			//util.Log.Debugf("push range [%d, %d) to %d server %s", KeyRange.Begin, self.clusterInfo.RangeEnd[BeginIdx], BeginIdx, self.clusterInfo.Servers[BeginIdx].Host)
			target[EndIdx] = communicate.Range{
				Begin: self.clusterInfo.RangeEnd[EndIdx-1],
				End:   KeyRange.End,
			}
			//util.Log.Debugf("push range [%d, %d) to %d server %s", self.clusterInfo.RangeEnd[EndIdx-1], KeyRange.End, EndIdx, self.clusterInfo.Servers[EndIdx].Host)
			for i := BeginIdx + 1; i < EndIdx; i++ {
				target[i] = communicate.Range{
					Begin: self.clusterInfo.RangeEnd[i-1],
					End:   self.clusterInfo.RangeEnd[i],
				}
				//util.Log.Debugf("push range [%d, %d) to %d server %s", self.clusterInfo.RangeEnd[i-1], self.clusterInfo.RangeEnd[i], i, self.clusterInfo.Servers[i].Host)
			}
		} else {
			target[BeginIdx] = *KeyRange
		}

		if len(target) > 0 {
			wg := sync.WaitGroup{}
			wg.Add(len(target))
			for idx, scope := range target {
				go func(idx int, scope communicate.Range) {
					defer wg.Done()
					if idx < len(self.clusterInfo.Servers) {
						//util.Log.Debugf("push idx %d scope [%d, %d) key range [%d, %d) deltas range [%d, %d) deltas count %d\n", idx, scope.Begin, scope.End, KeyRange.Begin, KeyRange.End, scope.Begin-KeyRange.Begin, scope.End-KeyRange.Begin, len(Values))
						deltas := Deltas[scope.Begin-KeyRange.Begin : scope.End-KeyRange.Begin]
						receiver := *self.clusterInfo.Servers[idx]
						if scope.Begin >= self.keyRange.Begin && scope.End <= self.keyRange.End {
							ok := self.addToLocal(&scope, deltas)
							success = success && ok
						} else {
							ok := self.pushDelta(&scope, deltas, receiver)
							success = success && ok
						}
					}
				}(idx, scope)
			}
			wg.Wait()
		}
	}
	//util.Log.Debugf("update [%d, %d) to cluster", KeyRange.Begin, KeyRange.End)
	return success
}

//pull 从另一台Server上获取参数
func (self *Server) pull(KeyRange *communicate.Range, Server communicate.Node) ([]*communicate.Value, bool) {
	if KeyRange == nil {
		util.Log.Errorf("KeyRange is nil")
		return []*communicate.Value{}, true
	}
	var storer *communicate.Node
	if Server.Ready {
		storer = &Server
		//util.Log.Debugf("will pull data [%d, %d] from master %s", keyRange.Begin, keyRange.End, Server.Host)
	} else {
		//如果Server不ready，就从它的Slave上拉取数据。Slave可能就是自己
		if len(self.clusterInfo.Slaves) > 0 { //Server机器数太少时len(self.clusterInfo.Slaves)=len(self.clusterInfo.Masters)=0
			//util.Log.Debugf("slaves of %s are %v", Server.Host, self.clusterInfo.Slaves[Server.Index].Nodes)
			for _, ele := range self.clusterInfo.Slaves[Server.Index].Nodes {
				if ele.Ready {
					storer = ele
					break
				}
			}
			util.Log.Debugf("will pull data [%d, %d] from %s's slave %s", KeyRange.Begin, KeyRange.End, Server.Host, storer.Host)
		}
	}
	if storer == nil {
		util.Log.Criticalf("server %s and its slaves are all dead, could not pull data", Server.Host)
		return []*communicate.Value{}, false
	}

	if storer.Host == self.node.Host {
		//从自己上pull数据
		if valueMap, ok := self.getFromLocal(KeyRange); ok {
			rect := make([]*communicate.Value, KeyRange.End-KeyRange.Begin)
			for i := KeyRange.Begin; i < KeyRange.End; i++ {
				if v, exists := valueMap[i]; exists {
					rect[i-KeyRange.Begin] = v
				}
			}
			return rect, true
		} else {
			return []*communicate.Value{}, false
		}
	} else {
		msg := communicate.Message{
			Sender:   &self.node,
			Receiver: storer,
			KeyRange: KeyRange,
			Command:  communicate.Command_PULL,
		}
		msgId := self.van.Send(&msg)
		ack := self.van.WaitAck(msgId, 2*time.Second)
		if ack == nil || !ack.RespondSuccess {
			if self.van.ReSend(msgId) > 0 { //只重试一次
				ack = self.van.WaitAck(msgId, 2*time.Second)
			}
		}
		if ack != nil && ack.RespondSuccess {
			rect := make([]*communicate.Value, len(ack.Values))
			for i, ele := range ack.Values {
				rect[i] = ele
			}
			//util.Log.Debugf("pull [%d, %d) from server %s", keyRange.Begin, keyRange.End, Server.Host)
			return rect, true
		} else {
			return []*communicate.Value{}, false
		}
	}
}

//push 把参数push给另一台Server
func (self *Server) push(KeyRange *communicate.Range, Values []*communicate.Value, Server communicate.Node) bool {
	if KeyRange == nil {
		util.Log.Errorf("KeyRange is nil")
		return true
	}
	if Server.Ready == false {
		util.Log.Errorf("server %s is dead, will not push data to it", Server.Host)
		return true
	}
	msg := communicate.Message{
		Sender:   &self.node,
		Receiver: &Server,
		KeyRange: KeyRange,
		Values:   Values,
		Command:  communicate.Command_UPDATE,
	}
	//util.Log.Debugf("push [%d, %d) to server %s", KeyRange.Begin, KeyRange.End, Server.Host)
	msgId := self.van.Send(&msg)
	ack := self.van.WaitAck(msgId, 2*time.Second)
	if ack == nil || ack.RespondSuccess == false {
		if self.van.ReSend(msgId) > 0 { //只重试一次
			ack = self.van.WaitAck(msgId, 2*time.Second)
		}
		//util.Log.Debugf("push [%d, %d) to server %s", keyRange.Begin, keyRange.End, Server.Host)
		if ack != nil {
			return ack.RespondSuccess
		} else {
			util.Log.Errorf("wanna push data to %s, but receive no ack", Server.Host)
			return false
		}
	} else {
		return true
	}
}

//pushDelta 把参数增量push给另一台Server
func (self *Server) pushDelta(KeyRange *communicate.Range, Deltas []*communicate.Value, Server communicate.Node) bool {
	if KeyRange == nil {
		util.Log.Errorf("KeyRange is nil")
		return true
	}
	if Server.Ready == false {
		util.Log.Errorf("server %s is dead, will not push data to it", Server.Host)
		return true
	}
	msg := communicate.Message{
		Sender:   &self.node,
		Receiver: &Server,
		KeyRange: KeyRange,
		Values:   Deltas,
		Command:  communicate.Command_ADD,
	}
	//util.Log.Debugf("push delta [%d, %d) to server %s", KeyRange.Begin, KeyRange.End, Server.Host)
	msgId := self.van.Send(&msg)
	ack := self.van.WaitAck(msgId, 2*time.Second)
	if ack == nil || ack.RespondSuccess == false {
		if self.van.ReSend(msgId) > 0 { //只重试一次
			ack = self.van.WaitAck(msgId, 2*time.Second)
		}
		//util.Log.Debugf("push delta [%d, %d) to server %s", keyRange.Begin, keyRange.End, Server.Host)
		if ack != nil {
			return ack.RespondSuccess
		} else {
			util.Log.Errorf("wanna push delta to %s, but receive no ack", Server.Host)
			return false
		}
	} else {
		return true
	}
}

func (self *Server) nodeChange(msg *communicate.Message) bool {
	success := true
	var oldKeyRange = self.keyRange
	var newKeyRange communicate.Range

	for i, server := range msg.ServerClusterInfo.Servers {
		if server.Host == self.node.Host {
			self.index = server.Index
			end := msg.ServerClusterInfo.RangeEnd[i]
			var begin int32
			if i > 0 {
				begin = msg.ServerClusterInfo.RangeEnd[i-1]
			} else {
				begin = 0
			}
			newKeyRange = communicate.Range{Begin: begin, End: end}
		}
	}
	if newKeyRange.End == 0 {
		util.Log.Critical("could not find myself %s from cluster info", self.node.Host)
		success = false
	} else {
		//KeyRange有变化。此时只关注Range扩大的情况，从别的server上copy数据
		if newKeyRange.Begin < oldKeyRange.Begin {
			copyRange := &communicate.Range{
				Begin: newKeyRange.Begin,
				End:   int32(math.Min(float64(newKeyRange.End), float64(oldKeyRange.Begin))),
			}
			values, ok := self.get(copyRange)
			success = success && ok
			if ok {
				for i := copyRange.Begin; i < copyRange.End; i++ {
					self.parameterMap.Put(i, values[i-copyRange.Begin])
				}
			}
		}
		if newKeyRange.End > oldKeyRange.End {
			copyRange := &communicate.Range{
				Begin: int32(math.Max(float64(newKeyRange.Begin), float64(oldKeyRange.End))),
				End:   newKeyRange.End,
			}
			values, ok := self.get(copyRange)
			success = success && ok
			if ok {
				for i := copyRange.Begin; i < copyRange.End; i++ {
					self.parameterMap.Put(i, values[i-copyRange.Begin])
				}
			}
		}
		if success {
			self.clusterInfo.Servers = msg.ServerClusterInfo.Servers
			self.clusterInfo.RangeEnd = msg.ServerClusterInfo.RangeEnd
			self.keyRange = newKeyRange
			util.Log.Infof("self key range change to [%d, %d)", self.keyRange.Begin, self.keyRange.End)
		}
	}
	return success
}

func (self *Server) nodeDead(server *communicate.Node) bool {
	success := true
	if int(server.Index) >= len(self.clusterInfo.Servers) {
		success = false
	} else {
		self.clusterInfo.Servers[server.Index].Ready = false
		util.Log.Debugf("set server %s to dead", server.Host)
	}
	return success
}

//changeSlaveData 拿到最新的主从关系后，把不属于自己的数据删除掉，释放内存；然后把自己的masters的数据pull过来
func (self *Server) changeSlaveData(msg *communicate.Message) bool {
	success := true

	self.clusterInfo.Servers = msg.ServerClusterInfo.Servers
	self.clusterInfo.Masters = msg.ServerClusterInfo.Masters
	self.clusterInfo.Slaves = msg.ServerClusterInfo.Slaves
	self.clusterInfo.RangeEnd = msg.ServerClusterInfo.RangeEnd

	//删除不属于自身KeyRange的数据
	iter := self.parameterMap.Iterator()
	for ; iter.HasNext(); {
		if k, _, ok := iter.Next(); ok {
			key := k.(int32)
			if key < self.keyRange.Begin || key >= self.keyRange.End {
				iter.Remove()
			}
		}
	}
	//util.Log.Debugf("delete data not owned by myself")

	//从master那儿把数据pull过来
	ok := self.pullFromMasters()
	success = success && ok

	//util.Log.Debugf("changeSlaveData %t", success)
	return success
}

//Init 初始化Van，向manager申请加入Server集群
func (self *Server) Init(manager string, port int) {
	self.node = communicate.Node{Host: util.GetSelfHost(), Ready: false, Role: communicate.Role_SERVER}
	self.van = communicate.GetZMQInstance(port)
	self.parameterMap = concurrent.NewConcurrentMap()
	self.manager = manager

	ManagerNode := communicate.Node{Host: manager, Role: communicate.Role_SERVER_MANAGER}
	msg := &communicate.Message{
		Sender:   &self.node,
		Receiver: &ManagerNode,
		Command:  communicate.Command_ADD_SERVER,
	}
	self.van.Send(msg)

	ch := make(chan os.Signal, 1)                                                       //创建管道，用于接收信号
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM) //注册信号
	go func() {
		sig := <-ch
		util.Log.Infof("receive signal %v", sig)
		self.ShutDown()
		util.Log.Flush()
		os.Exit(0)
	}()

	util.Log.Infof("server %s start, communicate on port %d", self.node.Host, port)
}

func (self *Server) dealMsg(msg *communicate.Message) {
	if msg == nil {
		return
	}

	//Ready之前，不处理来自worker的消息
	if !self.node.Ready && msg.Sender.Role == communicate.Role_WORKER {
		util.Log.Warnf("I am not ready, could not execute %s command from worker %s", msg.Command.String(), msg.Sender.Host)
		return
	}

	if msg.Command == communicate.Command_PULL {
		//对方想从集群上拉取数据
		values, success := self.get(msg.KeyRange)
		respMsg := &communicate.Message{
			Sender:         &self.node,
			Receiver:       msg.Sender,
			KeyRange:       msg.KeyRange,
			Command:        communicate.Command_PULL_ACK,
			Values:         values,
			RespondMsgId:   msg.Id,
			RespondSuccess: success,
		}
		self.van.Send(respMsg)
	} else if msg.Command == communicate.Command_PUSH {
		//对方想更新集群上的数据
		success := self.update(msg.KeyRange, msg.Values)
		respMsg := &communicate.Message{
			Sender:         &self.node,
			Receiver:       msg.Sender,
			Command:        communicate.Command_PUSH_ACK,
			RespondMsgId:   msg.Id,
			RespondSuccess: success,
		}
		self.van.Send(respMsg)
	} else if msg.Command == communicate.Command_UPDATE {
		//对方想更新我本地的数据
		success := self.updateToLocal(msg.KeyRange, msg.Values)
		respMsg := &communicate.Message{
			Sender:         &self.node,
			Receiver:       msg.Sender,
			Command:        communicate.Command_UPDATE_ACK,
			RespondMsgId:   msg.Id,
			RespondSuccess: success,
		}
		self.van.Send(respMsg)
	} else if msg.Command == communicate.Command_INC {
		//对方想让集群上的数据加一个增量
		success := self.add(msg.KeyRange, msg.Values)
		respMsg := &communicate.Message{
			Sender:         &self.node,
			Receiver:       msg.Sender,
			Command:        communicate.Command_INC_ACK,
			RespondMsgId:   msg.Id,
			RespondSuccess: success,
		}
		self.van.Send(respMsg)
	} else if msg.Command == communicate.Command_ADD {
		//对方想让我本地的数据加一个增量
		success := self.addToLocal(msg.KeyRange, msg.Values)
		respMsg := &communicate.Message{
			Sender:         &self.node,
			Receiver:       msg.Sender,
			Command:        communicate.Command_ADD_ACK,
			RespondMsgId:   msg.Id,
			RespondSuccess: success,
		}
		self.van.Send(respMsg)
	} else if msg.Command == communicate.Command_PING {
		//对方ping我是否还存活
		respMsg := &communicate.Message{
			Sender:         &self.node,
			Receiver:       msg.Sender,
			Command:        communicate.Command_PING_ACK,
			RespondMsgId:   msg.Id,
			RespondSuccess: true,
		}
		self.van.Send(respMsg)
	} else if msg.Command == communicate.Command_ADD_SERVER_ACK {
		if msg.RespondSuccess {
			self.clusterInfo = *msg.ServerClusterInfo //老的集群信息
			values, success := self.get(msg.KeyRange) //拿到数据
			if success {
				self.keyRange = *msg.KeyRange //拿到数据后更新自己的KeyRange
				util.Log.Infof("self key range change to [%d, %d)", self.keyRange.Begin, self.keyRange.End)
				self.updateToLocal(msg.KeyRange, values) //拿到数据后更新本地的Parameter
			} else {
				util.Log.Criticalf("start server failed, could not pull data")
			}
			respMsg := &communicate.Message{
				Sender:         &self.node,
				Receiver:       msg.Sender,
				Command:        communicate.Command_KEY_RANGE_CHANGE,
				KeyRange:       msg.KeyRange,
				RespondMsgId:   msg.Id,
				RespondSuccess: success,
			}
			self.van.Send(respMsg)
		} else {
			util.Log.Criticalf("start server failed, could not allocate key range")
		}
	} else if msg.Command == communicate.Command_KEY_RANGE_CHANGE_ACK {
		if msg.RespondSuccess {
			success := self.nodeChange(msg)
			respMsg := &communicate.Message{
				Sender:         &self.node,
				Receiver:       msg.Sender,
				Command:        communicate.Command_MASTER_SLAVE_CHANGE,
				RespondMsgId:   msg.Id,
				RespondSuccess: success,
			}
			if !success {
				util.Log.Criticalf("pll data of my own range failed")
			}
			self.van.Send(respMsg)
		} else {
			util.Log.Critical("manager recalculate key range failed")
		}
	} else if msg.Command == communicate.Command_MASTER_SLAVE_CHANGE_ACK {
		if msg.RespondSuccess {
			//util.Log.Debugf("will change master slave")
			success := self.changeSlaveData(msg)
			if success {
				self.node.Ready = true //把自己置为Ready
				util.Log.Infof("I am ready to work")
			}
			respMsg := &communicate.Message{
				Sender:         &self.node,
				Receiver:       msg.Sender,
				Command:        communicate.Command_CHANGE_SERVER_FINISH,
				RespondMsgId:   msg.Id,
				RespondSuccess: success,
			}
			self.van.Send(respMsg)
		} else {
			util.Log.Critical("recalculate master slave relation failed")
		}
	} else if msg.Command == communicate.Command_DELETE_SERVER {
		success := self.nodeDead(msg.DeleteNode)
		respMsg := &communicate.Message{
			Sender:         &self.node,
			Receiver:       msg.Sender,
			Command:        communicate.Command_DELETE_SERVER_ACK,
			RespondMsgId:   msg.Id,
			RespondSuccess: success,
		}
		self.van.Send(respMsg)
	}
}

//Work 开始接收并处理各种消息
func (self *Server) Work() {
	for {
		msg := self.van.PeekOneMessage()
		go func() { //异步处理接收到的消息
			self.dealMsg(msg)
		}()
	}
}

func (self *Server) ShutDown() {
	self.van.Close()
}
