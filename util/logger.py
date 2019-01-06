# -*- coding:utf-8 -*-
# Author : zhangzhaoyang
# Date : 2018-12-11 20:04


import logging
import inspect
import logging.handlers
import time
import os
import traceback
import requests
import json

LEVEL_2_NAME = {logging.DEBUG: 'DEBUG',
                logging.INFO: 'INFO',
                logging.WARNING: 'WARN',
                logging.ERROR: 'ERROR',
                logging.FATAL: 'FATAL'}

NAME_2_LEVEL = {'DEBUG': logging.DEBUG,
                'INFO': logging.INFO,
                'WARN': logging.WARNING,
                'ERROR': logging.ERROR,
                'FATAL': logging.FATAL}


class LoggerFactory(object):
    _cache = {}

    @classmethod
    def addLogger(cls, logName, logFile, minLevel='DEBUG', errorKey=None, subject="Too Much Error Log!", receivers=[]):
        logPath = os.path.dirname(logFile)
        if logPath:
            if not os.path.exists(logPath):
                os.mkdir(logPath)
        if logName in cls._cache:
            log = cls._cache[logName]
            log.config(logFile, minLevel, errorKey, subject, receivers)
        else:
            log = Logger(logName)
            log.config(logFile, minLevel, errorKey, subject, receivers)
            cls._cache[logName] = log

    @classmethod
    def setDefaultLogger(cls, logFile, minLevel='DEBUG', errorKey=None, subject="Too Much Error Log!", receivers=[]):
        logPath = os.path.dirname(logFile)
        if not os.path.exists(logPath):
            os.mkdir(logPath)
        cls.addLogger("default", logFile, minLevel,
                      errorKey, subject, receivers)

    @classmethod
    def getLogger(cls, logName="default"):
        if logName in cls._cache:
            return cls._cache[logName]
        else:
            log = Logger(logName)
            cls._cache[logName] = log
            return log


class Logger(object):

    def __init__(self, logName):
        self.logName = logName
        self.ready = False
        self.fromtime = -1
        self.errnum = 0
        self.havealert = False

    def config(self, logFile, minLevel, errorKey, subject, receivers):
        infoHandler = logging.handlers.TimedRotatingFileHandler(
            logFile, 'midnight')
        infoHandler.suffix = "%Y-%m-%d"
        self.LOG = logging.getLogger(self.logName)
        self.LOG.addHandler(infoHandler)
        self.errorKey = errorKey
        if minLevel.upper() in NAME_2_LEVEL:
            self.LOG.setLevel(NAME_2_LEVEL[minLevel.upper()])
        else:
            self.LOG.setLevel(logging.DEBUG)  # 默认情况下,最低日志级别为DEBUG
        self.subject = subject
        self.receivers = receivers
        self.ready = True

    def _printfNow(self):
        '''打印当前时刻
        '''
        now = time.localtime()
        return "{year}-{month:02d}-{day:02d} {hour:02d}:{minute:02d}:{second:02d}".format(year=now.tm_year,
                                                                                          month=now.tm_mon,
                                                                                          day=now.tm_mday,
                                                                                          hour=now.tm_hour,
                                                                                          minute=now.tm_min,
                                                                                          second=now.tm_sec)

    def _getStack(self):
        '''获取上一层的调用堆栈,返回“[时间][文件名:行号]”
        '''
        frame, filename, lineNo, functionName, code, unknowField = inspect.stack()[2]
        path_arr = filename.split(os.path.sep)
        return "[%s][%s:%s]" % (self._printfNow(), os.path.sep.join(path_arr[-2:]), lineNo)

    def _getMsg(self, level, message):
        '''日志格式：[日志级别][时间][文件名:行号]信息
        '''
        return "[{}]{}".format(LEVEL_2_NAME[level], message)

    def _alert(self, msg):
        data_dict = {}
        data_dict['subject'] = self.subject
        data_dict['recip'] = self.receivers
        data_dict['content'] = msg
        url = 'http://inapi.oss.com/v1/send/mail/'
        requests.post(url, data=json.dumps(data_dict))

    def _check_alert(self, msg):
        self.errnum += 1
        now = time.time()
        if self.fromtime > 0:
            if now - self.fromtime < 300:
                if self.errnum > 100 and not self.havealert:
                    self._alert(msg)
                    self.havealert = True
            else:
                self.fromtime = now
                self.havealert = False
        else:
            self.fromtime = now

    def debug(self, msg, exception=None):
        if self.ready:
            if exception:
                msg += ' ' + traceback.format_exc()
            self.LOG.debug(self._getMsg(
                logging.DEBUG, self._getStack() + msg))

    def info(self, msg, exception=None):
        if self.ready:
            if exception:
                msg += ' ' + traceback.format_exc()
            self.LOG.info(self._getMsg(
                logging.INFO, self._getStack() + msg))

    def warn(self, msg, exception=None):
        if self.ready:
            if exception:
                msg += ' ' + traceback.format_exc()
            self.LOG.warning(self._getMsg(
                logging.WARNING, self._getStack() + msg))

    def error(self, msg, exception=None):
        cont = msg
        if self.ready:
            if exception:
                msg += ' ' + traceback.format_exc()
            cont = self._getMsg(logging.ERROR, self._getStack() + msg)
            self.LOG.error(cont)
        # self._check_alert(cont)

    def fatal(self, msg, exception=None):
        cont = msg
        if self.ready:
            if exception:
                msg += ' ' + traceback.format_exc()
            cont = self._getMsg(logging.FATAL, self._getStack() + msg)
            self.LOG.fatal(cont)
        # self._check_alert(cont)


if __name__ == '__main__':
    # 在addLogger之前进行getLogger，是无法写入日志文件的
    logger = LoggerFactory.getLogger('P1')
    logger.info("I could not written to file")

    LoggerFactory.addLogger('P1', 'log1.txt')
    logger.debug("I'm debug log")
    logger.info("I'm info log")
    logger.warn("I'm warn log")
    logger.error("I'm error log")
    logger.fatal("I'm fatal log")

    LoggerFactory.addLogger('P2', 'log2.txt', 'error')
    logger = LoggerFactory.getLogger('P2')
    logger.debug("I'm debug log")
    logger.info("I'm info log")
    logger.warn("I'm warn log")
    logger.error("I'm error log")
    logger.fatal("I'm fatal log")
