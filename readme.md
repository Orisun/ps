## 部署前的准备
### 安装ZeroMQ
1. 下载zeromq-4.2.5.tar.gz
2. tar zxvf zeromq-4.2.5.tar.gz
3. cd zeromq-4.2.5
4. ./configure
5. make
6. sudo make install
7. sudo vim /etc/profile \
    export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/local/lib \
    export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
8. source /etc/profile
9. 如果要使用python客户端，还需要pip install pyzmq/expiringdict/requests
## 启动流程
1. 启动一台ServerManager，指定工作端口port以及模型参数的总数。
    ```Shell
    ./start.sh -r manager -p $port -g $group_name -c $parameter_total
    ```
2. 启动多台Server，指定工作端口port以及ServerManager的Host。
    ```Shell
    ./start.sh -r server -p $port -m $manager_host
    ```
3. 启动多台Worker。worker/go/lr_push.go（或worker/python/lr_push.py）里有demo。

说明
1. 训练一个模型时ServerManager、Servers、Workers要部在不同的机器上，使用相同的工作端口。
2. 一台机器上可以部署多个ServerManager/Servers/Workers实体，用于训练多个模型。
## 关闭流程
1. 关闭所有Worker。
2. 关闭所有Server。
    ```Shell
    ./stop.sh $port
    ```
3. 关闭ServerManager。
    ```Shell
    ./stop.sh $port
    ```