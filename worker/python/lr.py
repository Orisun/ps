# -*- coding:utf-8 -*-
# Author : zhangzhaoyang
# Date : 2018-12-12 18:22
# Description : 用Pull-Push模式

from __future__ import division

import os
import sys

sys.path.insert(0, os.getcwd())
from util.logger import LoggerFactory

LoggerFactory.setDefaultLogger("log/ps_client.log", "INFO")
logger = LoggerFactory.getLogger()

import time
import random

import numpy as np
from client.python.client import PsClient
from communicate.python.message_pb2 import Value, Range

from corpus import CorpusGenerator
from absl import flags
from absl import app

NEAR_0 = 1e-10

FLAGS = flags.FLAGS
flags.DEFINE_integer("port", 0, "work port")
flags.DEFINE_string("manager", "", "manager host")
flags.DEFINE_integer("parameter_count", 0, "total parameter count")
flags.DEFINE_string("corpus_file", "", "corpus file")
flags.DEFINE_integer("corpus_split_num", 1, "corpus split count")
flags.DEFINE_integer("corpus_split_index", 0, "read which index of corpus splits")
flags.DEFINE_integer("feature_split_num", 1, "feature split count")
flags.DEFINE_integer("feature_split_index", 0, "train which index of feature splits")


class LR(object):

    def __init__(self, manager, port, parameter_total, key_range, init_param_func, valuesEachKey):
        self.psClient = PsClient(manager, port, parameter_total, key_range)
        self.KeyRange = key_range
        # 初始化参数，从PS上取
        msgId = self.psClient.Pull()
        pullOk = self.psClient.WaitPull(msgId, 0)
        w = self.psClient.GetAllParameter()
        if not pullOk or w[key_range.Begin] is None or len(w[key_range.Begin].Values) == 0:
            initValues = []
            for i in xrange(parameter_total):
                value = Value()
                value.Values.extend([init_param_func()] * valuesEachKey)
                initValues.append(value)
            self.psClient.UpdateLocalParameter(initValues)
            msgId = self.psClient.Push()  # 把初始化好的参数push到PS集群上
            self.psClient.WaitPush(msgId, 5)
            logger.info("init parameter by init function")
        else:
            logger.info("init parameter from ps")

    def fn(self, w, x):
        '''决策函数为sigmoid函数。
           用tanh间接表示sigmoid，防止出现RuntimeWarning: overflowencountered in exp
        '''
        return (0.5 * (1 + np.tanh(0.5 * np.dot(x, w)))).reshape(x.shape[0], 1)

    def loss(self, y, y_hat):
        '''交叉熵损失函数'''
        return np.sum(
            np.nan_to_num(-y * np.log(y_hat + NEAR_0) - (1 - y) * np.log(1 - y_hat + NEAR_0)))  # 加上NEAR_0避免出现log(0)

    def grad(self, y, y_hat, x):
        '''交叉熵损失函数对权重w的一阶导数'''
        return np.mean((y_hat - y) * x, axis=0)


def TrainLrWithGD(corpusFile, splitNum, splitIndex, epoch, batch, eta, manager, port, ParameterTotal, KeyRange,
                  synRound):
    begin = time.time()
    lr = LR(manager, port, ParameterTotal, KeyRange, random.random, 1)
    iter = 1
    WaitedPullMsgId = 0
    use_time = []
    for ep in xrange(epoch):
        logger.info("epoch={}".format(ep))
        corpusGenerator = CorpusGenerator(corpusFile, splitNum, splitIndex, ParameterTotal)
        xBatch = []
        yBatch = []
        for X, Y in corpusGenerator:
            xBatch.append(X)
            yBatch.append(Y)
            if len(xBatch) >= batch:
                if WaitedPullMsgId > 0:
                    if not lr.psClient.WaitPull(WaitedPullMsgId, 1):  # 等之前的Pull命令完成
                        logger.error("wait pull timeout")
                msgId = lr.psClient.Pull()
                if iter % synRound == 0:
                    WaitedPullMsgId = msgId
                iter += 1
                t1 = time.time()
                x = np.array(xBatch)
                w = []
                for value in lr.psClient.GetAllParameter():
                    if len(value.Values)>=1:
                        w.append(value.Values[0])
                    else:
                        logger.error("parameters of one key less than 2: {}".format(len(value.Values)))
                w = np.array(w)
                y_hat = lr.fn(w, x)
                y = np.array(yBatch).reshape(len(yBatch), 1)
                g = lr.grad(y, y_hat, x[:, KeyRange.Begin:KeyRange.End])  # 只需要计算部分梯度
                w[KeyRange.Begin:KeyRange.End] -= eta * g  # 梯度下降法的核心公式，只更新自己负责的区间段
                Values = []
                for i in xrange(KeyRange.Begin, KeyRange.End):
                    value = Value()
                    value.Values.append(w[i])
                    Values.append(value)
                t2 = time.time()
                use_time.append(t2 - t1)
                lr.psClient.UpdateLocalRangedParameter(Values)
                lr.psClient.Push()
                xBatch = []
                yBatch = []
        logger.debug("update paramter {} times, mean use time {}".format(len(use_time), np.mean(np.array(use_time))))
        if len(xBatch) > 0:
            if WaitedPullMsgId > 0:
                if not lr.psClient.WaitPull(WaitedPullMsgId, 1):  # 等之前的Pull命令完成
                    logger.error("wait pull timeout")
            msgId = lr.psClient.Pull()
            if iter % synRound == 0:
                WaitedPullMsgId = msgId
            iter += 1
            x = np.array(xBatch)
            w = []
            for value in lr.psClient.GetAllParameter():
                if len(value.Values)>=1:
                    w.append(value.Values[0])
                else:
                    logger.error("parameters of one key less than 2: {}".format(len(value.Values)))
            w = np.array(w)
            y_hat = lr.fn(w, x)
            y = np.array(yBatch).reshape(len(yBatch), 1)
            g = lr.grad(y, y_hat, x[:, KeyRange.Begin:KeyRange.End])  # 只需要计算部分梯度
            w[KeyRange.Begin:KeyRange.End] -= eta * g  # 梯度下降法的核心公式，只更新自己负责的区间段
            Values = []
            for i in xrange(KeyRange.Begin, KeyRange.End):
                value = Value()
                value.Values.append(w[i])
                Values.append(value)
            lr.psClient.UpdateLocalRangedParameter(Values)
            lr.psClient.Push()

    logger.info("train lr with gd finished, use {} seconds".format(time.time() - begin))
    return lr


def init_zero():
    return 0.0


def TrainLrWithFTRL(corpusFile, splitNum, splitIndex, epoch, batch, manager, port, ParameterTotal, KeyRange,
                    alpha, beta, l1, l2, synRound):
    begin = time.time()
    lr = LR(manager, port, ParameterTotal, KeyRange, init_zero, 2)#TRL中的z和n初始化为0，注意n一定不能是负数
    iter = 1
    WaitedPullMsgId = 0
    use_time = []
    for ep in xrange(epoch):
        logger.info("epoch={}".format(ep))
        corpusGenerator = CorpusGenerator(corpusFile, splitNum, splitIndex, ParameterTotal)
        xBatch = []
        yBatch = []
        for X, Y in corpusGenerator:
            xBatch.append(X)
            yBatch.append(Y)
            if len(xBatch) >= batch:
                if WaitedPullMsgId > 0:
                    if not lr.psClient.WaitPull(WaitedPullMsgId, 1):  # 等之前的Pull命令完成
                        logger.error("wait pull timeout")
                msgId = lr.psClient.Pull()
                if iter % synRound == 0:
                    WaitedPullMsgId = msgId
                iter += 1
                t1 = time.time()
                x = np.array(xBatch)
                z = []
                n = []
                for v in lr.psClient.GetAllParameter():
                    if len(v.Values) >= 2:
                        z.append(v.Values[0])
                        n.append(v.Values[1])
                    else:
                        logger.error("parameters of one key less than 2: {}".format(len(v.Values)))
                z = np.array(z)
                n = np.array(n)
                # FTRL核心公式
                w = np.array(
                    [0 if np.abs(z[i]) <= l1 else (np.sign(z[i]) * l1 - z[i]) / (l2 + (beta + np.sqrt(n[i])) / alpha)
                     for i in xrange(len(z))])
                # print "w after", w[KeyRange.Begin:min(KeyRange.Begin + 10, KeyRange.End)]
                y_hat = lr.fn(w, x)
                y = np.array(yBatch).reshape(len(yBatch), 1)
                g = lr.grad(y, y_hat, x[:, KeyRange.Begin:KeyRange.End])  # 只需要计算部分梯度
                # print "g", g[0:min(10, g.shape[0])]
                sigma = (np.sqrt(n[KeyRange.Begin:KeyRange.End] + g * g) - np.sqrt(
                    n[KeyRange.Begin:KeyRange.End])) / alpha
                z[KeyRange.Begin:KeyRange.End] += g - sigma * w[KeyRange.Begin:KeyRange.End]  # 只更新自己负责的区间段
                # print "z after", z[KeyRange.Begin:min(KeyRange.Begin + 10, KeyRange.End)]
                n[KeyRange.Begin:KeyRange.End] += g * g  # 只更新自己负责的区间段
                Values = []
                for i in xrange(KeyRange.Begin, KeyRange.End):
                    value = Value()
                    value.Values.extend([z[i], n[i]])
                    Values.append(value)
                t2 = time.time()
                use_time.append(t2 - t1)
                lr.psClient.UpdateLocalRangedParameter(Values)
                lr.psClient.Push()
                xBatch = []
                yBatch = []
        logger.debug("update paramter {} times, mean use time {}".format(len(use_time), np.mean(np.array(use_time))))
        if len(xBatch) > 0:
            if WaitedPullMsgId > 0:
                if not lr.psClient.WaitPull(WaitedPullMsgId, 1):  # 等之前的Pull命令完成
                    logger.error("wait pull timeout")
            msgId = lr.psClient.Pull()
            if iter % synRound == 0:
                WaitedPullMsgId = msgId
            iter += 1
            x = np.array(xBatch)
            z = []
            n = []
            for v in lr.psClient.GetAllParameter():
                if len(v.Values) >= 2:
                    z.append(v.Values[0])
                    n.append(v.Values[1])
                else:
                    logger.error("parameters of one key less than 2: {}".format(len(v.Values)))
            z = np.array(z)
            n = np.array(n)
            # FTRL核心公式
            w = np.array(
                [0 if np.abs(z[i]) <= l1 else (np.sign(z[i]) * l1 - z[i]) / (l2 + (beta + np.sqrt(n[i])) / alpha)
                 for i in xrange(len(z))])
            y_hat = lr.fn(w, x)
            y = np.array(yBatch).reshape(len(yBatch), 1)
            g = lr.grad(y, y_hat, x[:, KeyRange.Begin:KeyRange.End])  # 只需要计算部分梯度
            sigma = (np.sqrt(n[KeyRange.Begin:KeyRange.End] + g * g) - np.sqrt(n[KeyRange.Begin:KeyRange.End])) / alpha
            z[KeyRange.Begin:KeyRange.End] += g - sigma * w[KeyRange.Begin:KeyRange.End]  # 只更新自己负责的区间段
            n[KeyRange.Begin:KeyRange.End] += g * g  # 只更新自己负责的区间段
            Values = []
            for i in xrange(KeyRange.Begin, KeyRange.End):
                value = Value()
                value.Values.extend([z[i], n[i]])
                Values.append(value)
            lr.psClient.UpdateLocalRangedParameter(Values)
            lr.psClient.Push()

    logger.info("train lr with ftrl finished, use {} seconds".format(time.time() - begin))
    return lr


def PreditByLR(lr, corpusFile, splitNum, splitIndex, alpha, beta, l1, l2):
    begin = time.time()
    key_range = Range()
    key_range.Begin = 0
    key_range.End = lr.psClient.parameterTotal
    corpusGenerator = CorpusGenerator(corpusFile, splitNum, splitIndex, lr.psClient.parameterTotal)
    loss = 0.0
    population = 0
    w = []
    z = []
    n = []
    for value in lr.psClient.GetAllParameter():
        if len(value.Values) == 1:
            w.append(value.Values[0])
        else:
            z.append(value.Values[0])
            n.append(value.Values[1])
    if len(z) > 0:
        logger.debug("model is trained by ftrl")
        z = np.array(z)
        n = np.array(n)
        # 根据n和z计算得到w
        w = np.array(
            [0 if np.abs(z[i]) <= l1 else (np.sign(z[i]) * l1 - z[i]) / (l2 + (beta + np.sqrt(n[i])) / alpha)
             for i in xrange(len(z))])

    else:
        logger.debug("model is trained by gd")
        w = np.array(w)
    for X, Y in corpusGenerator:
        x = np.array(X).reshape(1, len(X))
        y_hat = lr.fn(w, x)
        loss += lr.loss(np.array(Y), y_hat)
        population += 1

    logger.info(
        "predit {} samples, mean loss {}, use {} seconds".format(population, loss / population, time.time() - begin))


def main(argv):
    del argv

    port = FLAGS.port
    manager = FLAGS.manager
    parameterCount = FLAGS.parameter_count
    corpusFile = FLAGS.corpus_file
    corpusSplitNum = FLAGS.corpus_split_num
    corpusSplitIndex = FLAGS.corpus_split_index
    featureSplitNum = FLAGS.feature_split_num
    featreSplitIndex = FLAGS.feature_split_index
    epoch = FLAGS.epoch

    print("work port ", port)
    print("manager host ", manager)
    print("total parameter count ", parameterCount)
    print("corpus file ", corpusFile)
    print("corpus split count ", corpusSplitNum)
    print("read which index of corpus splits ", corpusSplitIndex)
    print("feature split count ", featureSplitNum)
    print("train which index of feature splits ", featreSplitIndex)
    print("train epoch ", epoch)

    if port <= 10000:
        print "could not bind to port less than 10000"
        sys.exit(1)

    featureSplitSize = int(parameterCount / featureSplitNum)
    keyBegin = featreSplitIndex * featureSplitSize
    keyEnd = (featreSplitIndex + 1) * featureSplitSize
    if featreSplitIndex + 1 == featureSplitNum:
        keyEnd = parameterCount

    KeyRange = Range()
    KeyRange.Begin = keyBegin
    KeyRange.End = keyEnd
    synRound = 1

    # lr = TrainLrWithGD(corpusFile, corpusSplitNum, corpusSplitIndex, epoch, 64, 1, manager, port, parameterCount,
    # KeyRange,synRound)
    lr = TrainLrWithFTRL(corpusFile, corpusSplitNum, corpusSplitIndex, epoch, 64,
                         manager, port, parameterCount, KeyRange, 0.1, 1.0, 1.0, 1.0, synRound)
    PreditByLR(lr, corpusFile, 10, 0, 0.1, 1.0, 1.0, 1.0)  # 取1/10的样本做预测
    lr.psClient.ShutDown()


if __name__ == "__main__":
    app.run(main)

# python example/python/lr.py --port 40010 --manager jobflume --parameter_count 10000 --corpus_file
# example/data/binary_class.csv --corpus_split_num 3 --corpus_split_index 0 --feature_split_num 3 --feature_split_index 0 --epoch 20
