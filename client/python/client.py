# -*- coding:utf-8 -*-
# Author : zhangzhaoyang
# Date : 2018-12-12 13:31

from util.logger import LoggerFactory

logger = LoggerFactory.getLogger()
from communicate.python.message_pb2 import Node, Message, ADD_WORKER, PULL, PULL_ACK, PUSH, \
    INC, Value, WORKER, SERVER_CHANGE
import sys
from util.net import get_host
from communicate.python.zmq_impl import ZMQ
from expiringdict import ExpiringDict
from threading import Thread
import threading
import time
import random


class PsClient(object):
    def __init__(self, manager, port, parameter_total, key_range):
        if key_range.Begin < 0 or key_range.End <= key_range.Begin or key_range.End > parameter_total:
            logger.error("invalid key range [{}, {}), total parameter {}".format(key_range.Begin, key_range.End,
                                                                                 parameter_total))
            sys.exit(1)
        self.node = Node()
        self.node.Index = -1
        self.node.Host = get_host()
        self.node.Ready = True
        self.node.Role = WORKER
        self.van = ZMQ()
        self.van.Init(port)
        self.parameterTotal = parameter_total
        self.servers = []
        self.keyRange = key_range
        self.parameters = [Value()] * parameter_total
        self.pullResponded = ExpiringDict(max_len=1000, max_age_seconds=600)  # pull请求是否有返回

        # 向ServerManager注册worker，获取Servers
        msg = Message()
        msg.Sender.Host = self.node.Host
        msg.Receiver.Host = manager
        msg.Command = ADD_WORKER
        msgId = self.van.Send(msg)

        # 开始接收server集群的变更信息
        th = self.ReceiveThread(self)
        th.setDaemon(True)
        th.start()

        # 等worker注册成功
        if self.van.WaitAck(msgId, 5) is Node:
            logger.error("regist worker failed")
            sys.exit(1)

        # 必须等待取到Servers，否则程序就退出
        for i in xrange(10):
            if len(self.servers) == 0:
                time.sleep(1)
            else:
                break
        if len(self.servers) == 0:
            logger.error("could not get servers")
            sys.exit(1)

        logger.info("init ps client ok")

    class ReceiveThread(Thread):
        def __init__(self, psclient):
            Thread.__init__(self)
            self.psclient = psclient

        def run(self):
            while True:
                msg = self.psclient.van.PeekOneMessage()
                t = threading.Thread(target=self.psclient.dealMsg, args=(msg,))
                t.setDaemon(True)
                t.start()

    def dealMsg(self, msg):
        if not msg:
            return
        if msg.Command == SERVER_CHANGE:
            self.servers = msg.ServerClusterInfo.Servers
            logger.info("server count change to {}".format(len(self.servers)))
        elif msg.Command == PULL_ACK:
            if msg.RespondMsgId > 0:
                self.pullResponded[msg.RespondMsgId] = True
                if msg.RespondSuccess:
                    if len(msg.Values) == len(self.parameters):
                        for i, vs in enumerate(msg.Values):
                            if len(self.parameters[i].Values) == 0:
                                if len(vs.Values) > 0:
                                    self.parameters[i].Values.extend([0.0] * len(vs.Values))
                            for j, v in enumerate(vs.Values):
                                self.parameters[i].Values[j] = v
                    else:
                        logger.error(
                            "respond values length {} not match self parameter length {}".format(len(msg.Values),
                                                                                                 len(self.parameters)))

    def Push(self):
        if len(self.servers) == 0:
            logger.error("no available server")
            sys.exit(1)
        Values = self.parameters[self.keyRange.Begin:self.keyRange.End]
        if len(Values) != int(self.keyRange.End - self.keyRange.Begin):
            logger.error("update value count {} not match self self key range [{}, {})", len(Values),
                         self.keyRange.Begin, self.keyRange.End)
            return -1

        # 随机获取一台server
        idx = random.randint(0, len(self.servers) - 1)
        server = self.servers[idx]
        # push数据
        msg = Message()
        msg.Sender.Host = self.node.Host
        msg.Receiver.Host = server.Host
        msg.KeyRange.Begin = self.keyRange.Begin
        msg.KeyRange.End = self.keyRange.End
        msg.Values.extend(self.parameters[self.keyRange.Begin:self.keyRange.End])
        msg.Command = PUSH
        msgId = self.van.Send(msg)
        return msgId

    def Inc(self, Deltas):
        if len(self.servers) == 0:
            logger.error("no available server")
            sys.exit(1)
        if len(Deltas) != int(self.keyRange.End - self.keyRange.Begin):
            logger.error("add delta value count {} not match self self key range [{}, {})", len(Deltas),
                         self.keyRange.Begin, self.keyRange.End)
            return -1

        # 随机获取一台server
        idx = random.randint(0, len(self.servers) - 1)
        server = self.servers[idx]
        # push数据
        msg = Message()
        msg.Sender.Host = self.node.Host
        msg.Receiver.Host = server.Host
        msg.KeyRange.Begin = self.keyRange.Begin
        msg.KeyRange.End = self.keyRange.End
        msg.Values.extend(Deltas)
        msg.Command = INC
        msgId = self.van.Send(msg)
        return msgId

    def Pull(self):
        """
        从PS集群上拉取全Range的参数
        :return:
        """
        if len(self.servers) == 0:
            logger.error("no available server")
            sys.exit(1)
        # 随机获取一台server
        idx = random.randint(0, len(self.servers) - 1)
        server = self.servers[idx]
        # pull数据
        msg = Message()
        msg.Sender.Host = self.node.Host
        msg.Receiver.Host = server.Host
        msg.KeyRange.Begin = 0
        msg.KeyRange.End = self.parameterTotal
        msg.Command = PULL
        msgId = self.van.Send(msg)
        return msgId

    def UpdateLocalRangedParameter(self, values):
        """
        更新本机负责训练的那部分参数，返回是否更新成功
        :param values:
        :return:
        """
        if len(values) != self.keyRange.End - self.keyRange.Begin:
            logger.error(
                "update value count {} not match self self key range [{}, {})".format(len(values), self.keyRange.Begin,
                                                                                      self.keyRange.End))
            return False
        for i, vs in enumerate(values):
            self.parameters[i + self.keyRange.Begin] = vs
        return True

    def UpdateLocalParameter(self, values):
        """
        更新本机的所有参数，返回是否更新成功
        :param values:
        :return:
        """
        if len(values) != self.parameterTotal:
            logger.error(
                "update value count {} not match total parameter count %d".format(len(values), self.parameterTotal))
            return False
        for i, vs in enumerate(values):
            self.parameters[i] = vs
        return True

    def GetAllParameter(self):
        """
        获取模型的所有参数
        :return:
        """
        return self.parameters

    def WaitPull(self, msgId, timeOut):
        """
        等待某一次的Pull完成。如果超时则返回false。如果不设超时则最多等5分钟
        :param msgId:
        :param timeOut: 单位秒
        :return:
        """
        if timeOut == 0:
            timeOut = 300
        max_wait_ms = int(timeOut * 1000)
        for i in xrange(max_wait_ms):
            ok = self.pullResponded.get(msgId)
            if ok:
                return True
            time.sleep(0.001)
        return False

    def WaitPush(self, msgId, timeOut):
        """
        等待某一次的Push完成。如果超时则返回false
        :param msgId:
        :param timeOut:
        :return:
        """
        msg = self.van.WaitAck(msgId, timeOut)
        if msg is None:
            return False
        return msg.RespondSuccess

    def WaitInc(self, msgId, timeOut):
        """
        等待某一次的Inc完成。如果超时则返回false
        :param msgId:
        :param timeOut:
        :return:
        """
        msg = self.van.WaitAck(msgId, timeOut)
        if msg is None:
            return False
        return msg.RespondSuccess

    def ShutDown(self):
        self.van.Close()
