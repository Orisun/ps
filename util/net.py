# -*- coding:utf-8 -*-
# Author : zhangzhaoyang
# Date : 2018-12-12 10:33

import socket
import uuid


def get_mac():
    mac_num = hex(uuid.getnode()).replace('0x', '').upper()
    mac = '-'.join(mac_num[i: i + 2] for i in range(0, 11, 2))
    return mac


def get_ipv4():
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(('8.8.8.8', 80))
        ip = s.getsockname()[0]
    finally:
        s.close()

    return ip


def get_host():
    return socket.gethostname()


def ip2int(ip):
    rect = 0
    arr = ip.split(".")
    if len(arr) == 4:
        try:
            for i, ele in enumerate(arr):
                n = int(ele)
                if n >= 0 and n < 256:
                    rect += n << (3 - i) * 8
        except:
            pass
    return rect


def mac2int(mac):
    try:
        s = mac.replace("-", "")
        return int(s, 16)
    except:
        return 0


if __name__ == "__main__":
    print get_mac(), mac2int(get_mac())
    print get_ipv4(), ip2int(get_ipv4())
    print get_host()
