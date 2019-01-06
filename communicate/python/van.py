# -*- coding:utf-8 -*-
# Author : zhangchaoyang
# Date : 2018/12/7 7:46 PM


sendMsgKeepTime = 60 * 5  # 发出去的消息要保留5分钟，因为对方可能会要求重发
sendMsgKeepCount = 100  # 发出去的消息最多保留100条
ackKeepTime = 60 * 5
ackKeepCount = 100
msgKeepCount = 1000


class Van(object):

    def Send(self, Message):
        """
        发送消息，返回消息ID
        :param Message:
        :return:
        """
        pass

    def ReSend(self, MsgId):
        """
        重发消息
        :param MsgId:
        :return:
        """
        pass

    def WaitAck(self, MsgId, TimeOut):
        """
        等待响应
        :param MsgId:
        :param TimeOut: TimeOut=0表示不设置超时
        :return:
        """
        pass

    def PeekOneMessage(self):
        """
        取出一条消息，如果没有请求则会一直阻塞
        :return:
        """
        pass

    def Close(self):
        """
        关闭网络连接
        :return:
        """
        pass
