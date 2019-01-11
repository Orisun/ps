# -*- coding:utf-8 -*-
# Author : zhangchaoyang

from telegraf.client import TelegrafClient

INFLUX_SERVER_HOST = "influxdb.int.taou.com"
INFLUX_SERVER_PORT = 8089  # UDP端口


def rm_blank_tag(tags):
    """
    在influxdb中tags是用来作索引的，以string形式存储，所以tags的value不能是空字符串
    """
    rect = {}
    if not tags:
        return rect
    for k, v in tags.items():
        if v != "":
            rect[k] = v
    return rect


class Metric(object):
    def __init__(self, prefix):
        '''
        :param prefix: 所有上报都会带上这个前缀，通常设为项目名
        '''
        self.client = TelegrafClient(host=INFLUX_SERVER_HOST, port=INFLUX_SERVER_PORT)
        self.prefix = prefix

    def OccurOnce(self, name, tags={}):
        '''
        发生一次，上报计数1
        :param name:
        :param tags:
        :return:
        '''
        self.client.metric(self.prefix + "_" + name + "_occur", 1, tags=rm_blank_tag(tags))

    def ReportCount(self, name, value, tags={}):
        '''
        上报一个计数
        :param name:
        :param value:
        :param tags:
        :return:
        '''
        self.client.metric(self.prefix + "_" + name, value, tags=rm_blank_tag(tags))

    def UseTime(self, name, time_elapse, tags={}):
        '''
        上报一个耗时
        :param name:
        :param time_elapse: 类型float，单位秒
        :param tags:
        :return:
        '''
        self.client.metric(self.prefix + "_" + name + "_usetime", int(time_elapse * 1E9),
                           tags=rm_blank_tag(tags))  # 耗时单位转换为纳秒
