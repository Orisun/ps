# -*- coding:utf-8 -*-
# Author : zhangchaoyang
# Date : 2018/12/16 3:16 PM

import numpy as np

# 每个样本1万维，1万个样本

xDim = 10000
w = np.random.uniform(0, 1, xDim)
population = 10000

positive_num = 0
with open("binary_class.csv", "w") as f_out:
    f_out.write("# w = {}\n".format(",".join(map(str, w.tolist()))))
    for i in xrange(population):
        x = np.random.uniform(-5, 5, xDim)
        y = 0.5 * (1 + np.tanh(0.5 * np.dot(x, w)))
        output = ",".join(map(str, x.tolist()))
        if y <= 0.5:
            f_out.write(output + ",0\n")
        else:
            f_out.write(output + ",1\n")
            positive_num += 1
