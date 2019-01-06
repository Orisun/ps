// Date         : 2018/12/2
// Author       : zhangchaoyang
// Description  : 
package server

import (
	communicate "ParameterServer/communicate/go"
	"ParameterServer/util"
	"ParameterServer/util/data_struct"
	"github.com/fanliao/go-concurrentMap"
	"hash"
	"hash/fnv"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type ChangeNodeType int

func (self ChangeNodeType) String() string {
	if self == ADD {
		return "ADD"
	} else if self == DELETE {
		return "DELETE"
	} else {
		return "UNKNOWN"
	}
}

const (
	UNKNOWN = iota
	ADD
	DELETE
)

type ServerManager struct {
	groupName             string
	node                  communicate.Node
	clusterInfo           communicate.Cluster
	van                   communicate.Van
	parameterTotal        int32
	changeNodeTransaction *ChangeNodeTransaction
	pingWait              *concurrent.ConcurrentMap
	workers               []*communicate.Node
	hasher                hash.Hash32
}

type ChangeNodeTransaction struct {
	changeClsterLock         sync.RWMutex
	keyRangeChangeBarrier    int
	masterSlaveChangeBarrier int
	changeNodeType           ChangeNodeType
	changingNodeBegin        time.Time
	changingNode             *communicate.Node
}

func (self *ChangeNodeTransaction) Begin(host *communicate.Node, ct ChangeNodeType) {
	self.changeClsterLock.Lock()
	self.changeNodeType = ct
	self.changingNodeBegin = time.Now()
	self.changingNode = host
	self.keyRangeChangeBarrier = 0
	self.masterSlaveChangeBarrier = 0
}

func (self *ChangeNodeTransaction) End() {
	self.changeClsterLock.Unlock()
	self.changeNodeType = UNKNOWN
	self.changingNode = nil
	self.keyRangeChangeBarrier = 0
	self.masterSlaveChangeBarrier = 0
}

func (self *ChangeNodeTransaction) Init() {
	self.changeNodeType = UNKNOWN
	self.changingNode = nil
	self.keyRangeChangeBarrier = 0
	self.masterSlaveChangeBarrier = 0
}

func (self *ServerManager) Init(groupName string, TotalParameterCount int32, port int) {
	self.groupName = groupName
	self.parameterTotal = TotalParameterCount
	self.node = communicate.Node{Host: util.GetSelfHost(), Ready: true, Role: communicate.Role_SERVER_MANAGER}
	self.van = communicate.GetZMQInstance(port)
	self.changeNodeTransaction = new(ChangeNodeTransaction)
	self.changeNodeTransaction.Init()
	self.pingWait = concurrent.NewConcurrentMap()
	self.hasher = fnv.New32()
	
	ch := make(chan os.Signal, 1)                                                       //创建管道，用于接收信号
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM) //注册信号
	go func() {
		sig := <-ch
		util.Log.Infof("receive signal %v", sig)
		self.ShutDown()
		util.Log.Flush()
		os.Exit(0)
	}()
	util.Log.Infof("server manager %s start, communicate on port %d", self.node.Host, port)
}

//calNewNodeRange 计算新节点的range
func (self *ServerManager) calNewNodeRange(host string) (communicate.Range, bool) {
	success := true
	var r communicate.Range
	if len(self.clusterInfo.Servers) == 0 {
		r.Begin = 0
		r.End = int32(self.parameterTotal)
	} else {
		//self.hasher.Write([]byte(host))
		//slotIdx := int32(self.hasher.Sum32() % uint32(self.parameterTotal))
		slotIdx := int32(util.GetSnowNode().Generate().Int64()&0x7fffffff) % self.parameterTotal
		prev := data_struct.BinarySearchSmaller(self.clusterInfo.RangeEnd, slotIdx)
		if prev == -1 { //成为第一个节点
			r.Begin = 0
			r.End = slotIdx
		} else if self.clusterInfo.RangeEnd[prev] == slotIdx {
			util.Log.Warnf("hash %s to same slot of the ring", host)
			success = false
		} else {
			r.Begin = self.clusterInfo.RangeEnd[prev]
			r.End = slotIdx
		}
	}
	util.Log.Debugf("key range for new server %s is [%d, %d)", host, r.Begin, r.End)
	return r, success
}

//addNode 更新ClusterInfo.Servers和ClusterInfo.RangeEnd
func (self *ServerManager) addNode(msg *communicate.Message) bool {
	success := true
	newNode := &communicate.Node{Host: msg.Sender.Host, Ready: true, Role: communicate.Role_SERVER}
	newRange := msg.KeyRange
	ServerCount := len(self.clusterInfo.Servers)
	if ServerCount == 0 {
		//第1个Server加入，放入Hash环的最后一个位置上
		if newRange.Begin == 0 && newRange.End == int32(self.parameterTotal) {
			newNode.Index = 0
			self.clusterInfo.Servers = []*communicate.Node{newNode}
			self.clusterInfo.RangeEnd = []int32{newRange.End}
		} else {
			util.Log.Critical("the first node not in the last position of hash ring")
			util.Log.Flush()
			syscall.Exit(1)
		}
	} else {
		if ServerCount >= 3 && int(self.parameterTotal)/ServerCount <= 100 {
			//每台机器都分不到100个parameter，则不需再增加server
			util.Log.Warnf("server is enough, need no more server")
			success = false
		} else {
			slotIdx := newRange.End
			prev := data_struct.BinarySearchSmaller(self.clusterInfo.RangeEnd, slotIdx)
			//util.Log.Debugf("slotIdx=%d prev=%d", slotIdx, prev)
			if prev == -1 { //成为第一个节点
				newNode.Index = 0
				servers := []*communicate.Node{newNode}
				for i, server := range self.clusterInfo.Servers {
					server.Index = int32(i) + 1 //后面的server.Index都得加1
					servers = append(servers, server)
				}
				rangeEnds := []int32{newRange.End}
				rangeEnds = append(rangeEnds, self.clusterInfo.RangeEnd...)
				self.clusterInfo.Servers = servers
				self.clusterInfo.RangeEnd = rangeEnds
			} else if self.clusterInfo.RangeEnd[prev] == slotIdx {
				success = false
			} else {
				//在servers数组中的顺序跟在hash ring上的顺序是一致的
				servers := []*communicate.Node{}
				rangeEnds := []int32{}
				servers = append(servers, self.clusterInfo.Servers[0:prev+1]...)
				newNode.Index = int32(prev) + 1
				servers = append(servers, newNode)
				for i := prev + 1; i < ServerCount; i++ {
					n := self.clusterInfo.Servers[i]
					n.Index = int32(i) + 1 //后面的server.Index都得加1
					servers = append(servers, n)
				}
				rangeEnds = append(rangeEnds, self.clusterInfo.RangeEnd[0:prev+1]...)
				rangeEnds = append(rangeEnds, newRange.End)
				rangeEnds = append(rangeEnds, self.clusterInfo.RangeEnd[prev+1:ServerCount]...)
				self.clusterInfo.Servers = servers
				self.clusterInfo.RangeEnd = rangeEnds
			}
		}
	}
	util.Log.Infof("new servers %v", self.clusterInfo.Servers)
	util.Log.Infof("new range ends %v", self.clusterInfo.RangeEnd)
	return success
}

//deleteNode 把changingNode删掉  TODO 此处有BUG
func (self *ServerManager) deleteNode() bool {
	if self.changeNodeTransaction.changingNode == nil {
		util.Log.Errorf("could not find delete node")
		return false
	}
	success := true
	ServerCount := len(self.clusterInfo.Servers)
	if ServerCount <= 1 {
		util.Log.Criticalf("no available server")
		success = false
	} else {
		servers := []*communicate.Node{}
		rangeEnds := []int32{}
		servers = append(servers, self.clusterInfo.Servers[0:self.changeNodeTransaction.changingNode.Index]...)
		if int(self.changeNodeTransaction.changingNode.Index)+1 < ServerCount {
			for _, ele := range self.clusterInfo.Servers[self.changeNodeTransaction.changingNode.Index+1 : ServerCount] {
				server := &communicate.Node{
					Index: ele.Index - 1, //后面的server.Index都得减1
					Host:  ele.Host,
					Ready: true,
					Role:  communicate.Role_SERVER,
				}
				servers = append(servers, server)
			}
		}
		if self.changeNodeTransaction.changingNode.Index == 0 {
			rangeEnds = self.clusterInfo.RangeEnd[1:ServerCount]
		} else {
			rangeEnds = append(rangeEnds, self.clusterInfo.RangeEnd[0:self.changeNodeTransaction.changingNode.Index-1]...)
			if int(self.changeNodeTransaction.changingNode.Index) < ServerCount {
				rangeEnds = append(rangeEnds, self.clusterInfo.RangeEnd[self.changeNodeTransaction.changingNode.Index:ServerCount]...)
			}
		}
		self.clusterInfo.Servers = servers
		self.clusterInfo.RangeEnd = rangeEnds
		
		util.Log.Infof("new servers %v", self.clusterInfo.Servers)
		util.Log.Infof("new range ends %v", self.clusterInfo.RangeEnd)
	}
	return success
}

func (self *ServerManager) updateMasterSlave() bool {
	success := true
	ServerCount := len(self.clusterInfo.Servers)
	if ServerCount <= 1 {
		//只有一台服务器时不存在主备关系
		self.clusterInfo.Masters = []*communicate.Nodes{}
		self.clusterInfo.Slaves = []*communicate.Nodes{}
	} else {
		masters := make([]*communicate.Nodes, ServerCount)
		slaves := make([]*communicate.Nodes, ServerCount)
		for i, _ := range self.clusterInfo.Servers {
			//只有两台服务器时，互为备份
			m1 := (ServerCount + i - 1) % ServerCount
			s1 := (ServerCount + i + 1) % ServerCount
			masters[i] = &communicate.Nodes{Nodes: []*communicate.Node{self.clusterInfo.Servers[m1]}}
			slaves[i] = &communicate.Nodes{Nodes: []*communicate.Node{self.clusterInfo.Servers[s1]}}
			//达到三台服务器时，每台机器都有两个备份
			if ServerCount >= 3 {
				m2 := (ServerCount + i - 2) % ServerCount
				s2 := (ServerCount + i + 2) % ServerCount
				masters[i].Nodes = append(masters[i].Nodes, self.clusterInfo.Servers[m2])
				slaves[i].Nodes = append(slaves[i].Nodes, self.clusterInfo.Servers[s2])
			}
		}
		self.clusterInfo.Masters = masters
		self.clusterInfo.Slaves = slaves
	}
	util.Log.Debugf("updateMasterSlave %t", success)
	return success
}

func (self *ServerManager) dealMsg(msg *communicate.Message) {
	if msg == nil {
		return
	}
	if msg.Command == communicate.Command_ADD_SERVER {
		self.changeNodeTransaction.Begin(msg.Sender, ADD)
		exists := false //确保server不重复
		existIndex := -1
		for i, server := range self.clusterInfo.Servers {
			if server.Host == msg.Sender.Host {
				server.Ready = true
				exists = true
				existIndex = i
				break
			}
		}
		var success bool
		var r communicate.Range
		if exists {
			success = true
			if existIndex == 0 {
				r = communicate.Range{
					Begin: 0,
					End:   self.clusterInfo.RangeEnd[existIndex],
				}
				util.Log.Debugf("exists node 0")
			} else {
				r = communicate.Range{
					Begin: self.clusterInfo.RangeEnd[existIndex-1],
					End:   self.clusterInfo.RangeEnd[existIndex],
				}
				util.Log.Debugf("exists node ")
			}
		} else {
			r, success = self.calNewNodeRange(msg.Sender.Host)
			if !success {
				self.changeNodeTransaction.End()
			}
		}
		respMsg := &communicate.Message{
			Sender:            &self.node,
			Receiver:          msg.Sender,
			Command:           communicate.Command_ADD_SERVER_ACK,
			RespondSuccess:    success,
			RespondMsgId:      msg.Id,
			KeyRange:          &r,
			ServerClusterInfo: &self.clusterInfo,
		}
		self.van.Send(respMsg)
	} else if msg.Command == communicate.Command_KEY_RANGE_CHANGE {
		if msg.RespondSuccess {
			if success := self.addNode(msg); success {
				//把KeyRange的变化广播出去
				for _, peer := range self.clusterInfo.Servers {
					respMsg := &communicate.Message{
						Sender:            &self.node,
						Receiver:          peer,
						ServerClusterInfo: &self.clusterInfo,
						Command:           communicate.Command_KEY_RANGE_CHANGE_ACK,
						RespondSuccess:    success,
						RespondMsgId:      msg.Id,
					}
					self.van.Send(respMsg)
					self.changeNodeTransaction.keyRangeChangeBarrier++
				}
			} else {
				util.Log.Critical("recalculate key range failed")
				self.changeNodeTransaction.End()
			}
		} else {
			self.changeNodeTransaction.End()
		}
	} else if msg.Command == communicate.Command_MASTER_SLAVE_CHANGE {
		self.changeNodeTransaction.keyRangeChangeBarrier--
		if !msg.RespondSuccess {
			self.van.ReSend(msg.RespondMsgId) //有任何一台server失败则让它重试
			util.Log.Criticalf("server %s exec nodeChange faild, resend command to let it tyr again", msg.Sender.Host)
			self.changeNodeTransaction.keyRangeChangeBarrier++
		} else if self.changeNodeTransaction.keyRangeChangeBarrier == 0 {
			if success := self.updateMasterSlave(); success {
				//把主从关系的变化广播出去
				for _, peer := range self.clusterInfo.Servers {
					respMsg := &communicate.Message{
						Sender:            &self.node,
						Receiver:          peer,
						ServerClusterInfo: &self.clusterInfo,
						Command:           communicate.Command_MASTER_SLAVE_CHANGE_ACK,
						RespondSuccess:    success,
						RespondMsgId:      msg.Id,
					}
					self.van.Send(respMsg)
					self.changeNodeTransaction.masterSlaveChangeBarrier++
				}
			} else {
				util.Log.Critical("recalculate master slave relation failed")
				self.changeNodeTransaction.End()
			}
		}
	} else if msg.Command == communicate.Command_CHANGE_SERVER_FINISH {
		self.changeNodeTransaction.masterSlaveChangeBarrier--
		if !msg.RespondSuccess {
			self.van.ReSend(msg.RespondMsgId) //有任何一台server失败则让它重试
			util.Log.Criticalf("server %s exec changeSlaveData faild, resend command to let it tyr again", msg.Sender.Host)
			self.changeNodeTransaction.keyRangeChangeBarrier++
		} else if self.changeNodeTransaction.masterSlaveChangeBarrier == 0 {
			util.Log.Criticalf("%s server %s, use time %d milliseconds", self.changeNodeTransaction.changeNodeType.String(),
				self.changeNodeTransaction.changingNode.Host,
				time.Since(self.changeNodeTransaction.changingNodeBegin).Nanoseconds()/1e6)
			self.changeNodeTransaction.End()
		}
		//把集群信息的变化广播给worker
		for _, worker := range self.workers {
			respMsg := &communicate.Message{
				Sender:            &self.node,
				Receiver:          worker,
				ServerClusterInfo: &self.clusterInfo,
				Command:           communicate.Command_KEY_RANGE_CHANGE_ACK,
			}
			self.van.Send(respMsg)
		}
	} else if msg.Command == communicate.Command_DELETE_SERVER_ACK {
		self.changeNodeTransaction.keyRangeChangeBarrier--
		if !msg.RespondSuccess {
			self.van.ReSend(msg.RespondMsgId) //有任何一台server失败则让它重试
			util.Log.Criticalf("server %s exec nodeDead faild, resend command to let it tyr again", msg.Sender.Host)
			self.changeNodeTransaction.keyRangeChangeBarrier++
		} else if self.changeNodeTransaction.keyRangeChangeBarrier == 0 {
			if success := self.deleteNode(); success {
				//把KeyRange的变化广播出去
				for _, peer := range self.clusterInfo.Servers {
					respMsg := &communicate.Message{
						Sender:            &self.node,
						Receiver:          peer,
						ServerClusterInfo: &self.clusterInfo,
						Command:           communicate.Command_KEY_RANGE_CHANGE_ACK,
						RespondSuccess:    success,
						RespondMsgId:      msg.Id,
					}
					self.van.Send(respMsg)
					self.changeNodeTransaction.keyRangeChangeBarrier++
				}
			} else {
				util.Log.Critical("recalculate key range failed")
				self.changeNodeTransaction.End()
			}
		}
	} else if msg.Command == communicate.Command_PING_ACK {
		if msg.RespondSuccess {
			msgId := msg.RespondMsgId
			if v, err := self.pingWait.Get(msgId); err == nil && v != nil {
				wg := v.(*sync.WaitGroup)
				wg.Done()
			}
		}
	} else if msg.Command == communicate.Command_ADD_WORKER {
		exists := false //确保worker不重复
		for _, worker := range self.workers {
			if worker.Host == msg.Sender.Host {
				worker.Ready = true
				exists = true
				break
			}
		}
		if !exists {
			self.workers = append(self.workers, msg.Sender)
			util.Log.Infof("add worker %s", msg.Sender.Host)
		}
		//把ServerClusterInfo告诉给新注册的worker
		respMsg := &communicate.Message{
			Sender:            &self.node,
			Receiver:          msg.Sender,
			ServerClusterInfo: &self.clusterInfo,
			Command:           communicate.Command_SERVER_CHANGE,
			RespondSuccess:    true,
			RespondMsgId:      msg.Id,
		}
		self.van.Send(respMsg)
	}
}

func (self *ServerManager) Work() {
	go func() {
		pingFailCount := make(map[string]int)
		var failServer []*communicate.Node
		for {
			failServer = []*communicate.Node{}
			for _, peer := range self.clusterInfo.Servers {
				if !peer.Ready {
					continue
				}
				msg := &communicate.Message{
					Sender:   &self.node,
					Receiver: peer,
					Command:  communicate.Command_PING,
				}
				msgId := self.van.Send(msg)
				wg := sync.WaitGroup{}
				wg.Add(1)
				self.pingWait.Put(msgId, &wg)
				
				ok := util.WaitTimeout(&wg, 500*time.Millisecond) //500毫秒没有回应，就认为本次ping对应无法回应
				if ok {
					pingFailCount[peer.Host] = 0
				} else {
					if _, exists := pingFailCount[peer.Host]; exists {
						pingFailCount[peer.Host] += 1
					} else {
						pingFailCount[peer.Host] = 1
					}
					if pingFailCount[peer.Host] >= 5 {
						//连续超过5次ping都没响应
						failServer = append(failServer, peer)
					}
				}
			}
			if len(failServer) > 0 {
				if len(failServer) == len(self.clusterInfo.Servers) {
					util.Log.Criticalf("all servers are dead")
					self.clusterInfo.Servers = []*communicate.Node{}
				} else {
					for _, peer := range failServer {
						peer.Ready = false
						util.Log.Debugf("server %s dead", peer.Host)
						//获得锁，确保add_node和delete_node顺序进行
						self.changeNodeTransaction.Begin(peer, DELETE)
						//把节点已死的信息广播出去
						for _, server := range self.clusterInfo.Servers {
							if !data_struct.Contain(peer.Host, failServer) {
								self.changeNodeTransaction.keyRangeChangeBarrier++
								deletemsg := &communicate.Message{
									Sender:     &self.node,
									Receiver:   server,
									Command:    communicate.Command_DELETE_SERVER,
									DeleteNode: peer,
								}
								self.van.Send(deletemsg)
							}
						}
					}
				}
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}()
	
	for {
		msg := self.van.PeekOneMessage()
		go func() { //异步处理接收到的消息
			self.dealMsg(msg)
		}()
	}
}

func (self *ServerManager) ShutDown() {
	self.van.Close()
}
