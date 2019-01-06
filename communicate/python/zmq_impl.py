# -*- coding:utf-8 -*-
# Author : zhangchaoyang
# Date : 2018/12/7 7:57 PM

import signal
import time

from util import snowflake
from util.logger import LoggerFactory

logger = LoggerFactory.getLogger()
from collections import deque
from threading import Thread
from message_pb2 import Message, PING_ACK, PING
from expiringdict import ExpiringDict
import traceback
import zmq
from util import singleton
from van import Van
from van import sendMsgKeepTime, sendMsgKeepCount, ackKeepTime, ackKeepCount, msgKeepCount


class ZMQ(Van):
    # ZMQ是单例
    __metaclass__ = singleton.Singleton

    init = False
    closed = False

    @classmethod
    def Init(self, port):
        if self.init:
            return
        self.init = True
        self.port = port
        context = zmq.Context.instance()
        # REQ--REP是同步的，即必须收到reply后才能send下一条消息；DEALER--ROUTER是完全异步的
        self.receiveSocket = context.socket(zmq.ROUTER)
        self.receiveSocket.bind("tcp://*:" + str(port))
        time.sleep(0.1)  # 等待receiveSocket监听成功
        self.sendHistory = ExpiringDict(max_len=sendMsgKeepCount, max_age_seconds=sendMsgKeepTime)
        self.waitedMsgCache = ExpiringDict(max_len=ackKeepCount, max_age_seconds=ackKeepTime)
        self.msgChannel = deque(maxlen=msgKeepCount)  # 当新的元素加入并且这个队列已满的时候， 最老的元素会自动被移除掉，不会阻塞add操作
        signal.signal(signal.SIGTERM, self.Close)
        th = self.ReceiveThread(self)
        th.start()

    class ReceiveThread(Thread):
        def __init__(self, van):
            Thread.__init__(self)
            self.van = van

        def run(self):
            while True:
                msg = Message()
                try:
                    data = self.van.receiveSocket.recv()
                    msg.ParseFromString(data)
                except:
                    if self.van.closed:
                        return
                    logger.error(
                        "receive message error: " + traceback.format_exc(0) + " messgae length {}".format(len(data)))
                else:
                    if msg.Command != PING_ACK and msg.Command != PING:
                        pass
                        # logger.debug("receive command {} for msg {} from {}, data length {}".format(msg.Command,msg.RespondMsgId,msg.Sender.Host,len(data)))
                    if msg.RespondMsgId > 0:
                        self.van.waitedMsgCache[msg.RespondMsgId] = msg
                    self.van.msgChannel.append(msg)

    @classmethod
    def Send(self, msg):
        """
        发送消息，返回消息ID
        :param Message:
        :return:
        """
        msg.Id = snowflake.snowflake()  # 临发送的时候才给MessageID赋值
        context = zmq.Context.instance()
        sendSocket = context.socket(zmq.DEALER)
        sendSocket.connect("tcp://{}:{}".format(msg.Receiver.Host, self.port))
        data = msg.SerializeToString()
        sendSocket.send(data)
        self.sendHistory[msg.Id] = msg
        if msg.Command != PING_ACK and msg.Command != PING:
            pass
            # logger.debug("send command {} to {} msgid {} resp msgid {}".format(msg.Command, msg.Receiver.Host, msg.Id,msg.RespondMsgId))
        sendSocket.close()
        return msg.Id

    @classmethod
    def ReSend(self, msgId):
        """
        重发消息
        :param MsgId:
        :return:
        """
        msg = self.sendHistory.get(msgId)
        if msg is not None:
            return self.Send(msg)
        else:
            logger.error("ask resend message {}, but not cache it".format(msgId))
            return 0

    @classmethod
    def WaitAck(self, msgId, timeOut):
        """
        等待响应
        :param MsgId:
        :param TimeOut: 单位秒。TimeOut=0表示采用黑夜的默认时时间
        :return:
        """
        if timeOut == 0:
            timeOut = ackKeepTime
        max_wait_ms = int(timeOut * 1000)
        for i in xrange(max_wait_ms):
            msg = self.waitedMsgCache.get(msgId)
            if msg is not None:
                return msg
            time.sleep(0.001)
        return None

    @classmethod
    def PeekOneMessage(self):
        """
        取出一条消息，如果没有请求则会一直阻塞
        :return:
        """
        while True:
            try:
                msg = self.msgChannel.popleft()
                return msg
            except:  # 如果deque已空，pop操作会抛出异常
                pass

    @classmethod
    def Close(self):
        """
        关闭网络连接
        :return:
        """
        if self.receiveSocket:
            self.receiveSocket.close()
            self.closed = True


if __name__ == "__main__":
    inst = ZMQ()
    inst.Init(40010)  # 开始监听，准备接收消息
    msg = Message()
    msg.Sender.Host = "localhost"
    msg.Receiver.Host = "127.0.0.1"
    msg.Command = PING
    msgId = inst.Send(msg)

    time.sleep(1)
    msg2 = inst.WaitAck(msgId, 0.1)
    if msg2:
        print msg2.Id
        print msg2.Sender
        print msg2.Receiver
        print msg2.Command
