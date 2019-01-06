# -*- coding:utf-8 -*-
# Author : zhangchaoyang
# Date : 2018/12/7 8:04 PM

class Singleton(type):
    '''想让一个类只有一个实例，只需要把它的__metaclass__设为Singleton即可
    '''
    def __init__(cls, class_name, base_classes, attr_dict):
        cls.__instance = None
        super(Singleton, cls).__init__(class_name, base_classes, attr_dict)

    def __call__(cls, *args, **kwargs):
        if cls.__instance is None:
            cls.__instance = super(Singleton, cls).__call__(*args, **kwargs)
            return cls.__instance
        else:
            return cls.__instance