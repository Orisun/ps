# -*- coding:utf-8 -*-
# Author : zhangchaoyang
# Date : 2018/12/16 9:50 PM

from __future__ import division

import os
import sys
import time

sys.path.insert(0, os.getcwd())
import numpy as np

from util.logger import LoggerFactory

LoggerFactory.setDefaultLogger("log/ps_client.log", "INFO")
logger = LoggerFactory.getLogger()
from corpus import CorpusGenerator

NEAR_0 = 1e-10


class LR_CLS(object):
    '''用于二分类的LR
    '''

    @staticmethod
    def fn(w, x):
        '''决策函数为sigmoid函数。
           用tanh间接表示sigmoid，防止出现RuntimeWarning: overflowencountered in exp
        '''
        return (0.5 * (1 + np.tanh(0.5 * np.dot(x, w)))).reshape(x.shape[0], 1)

    @staticmethod
    def loss(y, y_hat):
        '''交叉熵损失函数'''
        return np.sum(
            np.nan_to_num(-y * np.log(y_hat + NEAR_0) - (1 - y) * np.log(1 - y_hat + NEAR_0)))  # 加上NEAR_0避免出现log(0)

    @staticmethod
    def grad(y, y_hat, x):
        '''交叉熵损失函数对权重w的一阶导数'''
        return np.mean((y_hat - y) * x, axis=0)


class LR_REG(object):
    '''用回归的LR
    '''

    @staticmethod
    def fn(w, x):
        '''决策函数为sigmoid函数'''
        return 1.0 / (1.0 + np.exp(-np.dot(x, w))).reshape(x.shape[0], 1)

    @staticmethod
    def loss(y, y_hat):
        '''误差平方和损失函数'''
        return np.sum(np.nan_to_num((y_hat - y) ** 2)) / 2

    @staticmethod
    def grad(y, y_hat, x):
        '''误差平方和损失函数对权重w的一阶导数'''
        return np.mean((y_hat - y) * (1 - y_hat) * y_hat * x, axis=0)


class FTRL(object):

    def __init__(self, dim, l1, l2, alpha, beta, decisionFunc=LR_CLS):
        self.dim = dim
        self.decisionFunc = decisionFunc
        self.z = np.zeros(dim)
        self.n = np.zeros(dim)
        self.w = np.zeros(dim)
        self.l1 = l1
        self.l2 = l2
        self.alpha = alpha
        self.beta = beta

    def predict(self, x):
        return self.decisionFunc.fn(self.w, x)

    def update(self, x, y):
        self.w = np.array([0 if np.abs(self.z[i]) <= self.l1 else (np.sign(self.z[i]) * self.l1 - self.z[i]) / (
                self.l2 + (self.beta + np.sqrt(self.n[i])) / self.alpha) for i in xrange(self.dim)])
        y_hat = self.predict(x)
        g = self.decisionFunc.grad(y, y_hat, x)
        sigma = (np.sqrt(self.n + g * g) - np.sqrt(self.n)) / self.alpha
        self.z += g - sigma * self.w
        self.n += g * g
        return self.decisionFunc.loss(y, y_hat)

    def train(self, corpus_file, verbos=False, epochs=100, batch=64):
        begin = time.time()
        total = 0
        for itr in xrange(epochs):
            logger.info("Epoch={:d}".format(itr))
            # 尽量使用mini batch，充分发挥numpy的并行计算能力
            mini_batch_x = []
            mini_batch_y = []
            n = 0
            corpus_generator = CorpusGenerator(corpus_file, 1, 0, self.dim)
            for X, Y in corpus_generator:
                n += 1
                mini_batch_x.append(X)
                mini_batch_y.append([Y])
                if len(mini_batch_x) >= batch:
                    self.update(np.array(mini_batch_x), np.array(mini_batch_y))
                    if verbos:
                        Y_HAT = self.predict(np.array(mini_batch_x))
                        train_loss = self.decisionFunc.loss(np.array(mini_batch_y), Y_HAT) / len(mini_batch_x)
                        logger.info("{:d}/{:d} train loss: {:f}\n".format(n, total, train_loss))
                    mini_batch_x = []
                    mini_batch_y = []
            self.update(np.array(mini_batch_x), np.array(mini_batch_y))
            if total == 0:
                total = n
            if verbos:
                Y_HAT = self.predict(np.array(mini_batch_x))
                train_loss = self.decisionFunc.loss(np.array(mini_batch_y), Y_HAT) / len(mini_batch_x)
                logger.info("{:d}/{:d} train loss: {:f}\n".format(n, total, train_loss))

        logger.info("train lr with ftrl finished, use {} seconds".format(time.time() - begin))

    def verify(self, corpus_file, xdim):
        begin = time.time()
        batch_x = []
        batch_y = []
        n = 0
        corpus_generator = CorpusGenerator(corpus_file, 10, 0, xdim)
        for X, Y in corpus_generator:
            n += 1
            batch_x.append(X)
            batch_y.append([Y])
        x = np.array(batch_x)
        y = np.array(batch_y)
        y_hat = self.predict(x)
        loss = self.decisionFunc.loss(y, y_hat) / x.shape[0]
        logger.info(
            "predit {} samples, mean loss {}, use {} seconds".format(n, loss, time.time() - begin))


if __name__ == "__main__":
    ftrl = FTRL(dim=10000, l1=1.0, l2=1.0, alpha=0.1, beta=1.0, decisionFunc=LR_CLS)
    ftrl.train("example/data/binary_class.csv", verbos=False, epochs=20, batch=64)
    ftrl.verify("example/data/binary_class.csv", xdim=10000)
