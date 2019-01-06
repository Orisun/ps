# -*- coding:utf-8 -*-
# Author : zhangzhaoyang
# Date : 2018-12-12 13:29

import os
import random
import sys
import time

sys.path.insert(0, os.getcwd())

from client.python.client import PsClient
from communicate.python.message_pb2 import Range, Value
from util.logger import LoggerFactory

if __name__ == "__main__":
    LoggerFactory.setDefaultLogger("log/ps_client.log", "DEBUG")
    parameter_total = 10000  # 多少个key
    ParameterCountOfKey = 10  # 一个key对应多个参数
    SelfRange = Range()
    SelfRange.Begin = 1000
    SelfRange.End = 9000
    client = PsClient("jobflume", 40010, parameter_total, SelfRange)

    WaitedPushMsgId = 0
    WaitedPullMsgId = 0

    for i in xrange(10):
        print "iter", i
        # 等待上上次的Push完成
        if WaitedPushMsgId > 0:
            client.WaitPush(WaitedPushMsgId, 0)

        # 等待上上次的Pull完成
        if WaitedPullMsgId > 0:
            client.WaitPull(WaitedPullMsgId, 0)

        # Pull
        msgId = client.Pull()
        if i % 3 == 0:
            WaitedPullMsgId = msgId

        # Push
        values = []
        for key in xrange(SelfRange.Begin, SelfRange.End):
            ele = Value()
            for j in xrange(ParameterCountOfKey):
                ele.Values.append(random.random())
            values.append(ele)
        client.UpdateLocalRangedParameter(values)
        msgId = client.Push()
        if i % 3 == 0:
            WaitedPushMsgId = msgId

        time.sleep(0.1)

    params = client.GetAllParameter()
    for i in xrange(SelfRange.Begin, SelfRange.End):
        l = params[i]
        print  i, "param", l.Values
        if i >= SelfRange.Begin + 10:
            break

    client.ShutDown()
