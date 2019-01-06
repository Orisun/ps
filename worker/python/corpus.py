# -*- coding:utf-8 -*-
# Author : zhangzhaoyang
# Date : 2018-12-11 17:52

from random import random


class CorpusGenerator(object):
    def __init__(self, infile, splitNum, index, xDim):
        '''
        :param infile: 样本所在的文件
        :param splitNum: 把训练数据分成splitNum份
        :param index: 这里只读取第index份
        :param xDim: X的维度
        '''
        self.infile = infile
        self.splitNum = splitNum
        self.index = index
        self.xDim = xDim
        self.f_in = open(self.infile)

    def __iter__(self):
        return self

    def next(self):
        while True:
            line = self.f_in.readline()
            if not line:
                raise StopIteration
            if int(random() * self.splitNum) == self.index:
                if line.startswith("#"):
                    continue
                arr = line.strip().split(",")
                if len(arr) == self.xDim + 1:
                    X = map(float, arr[:self.xDim])
                    Y = float(arr[-1])
                    return (X, Y)
                else:
                    print len(arr)
