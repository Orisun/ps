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
3. 启动多台Client，指定工作端口port以及ServerManager的Host。

说明
1. ServerManager、Servers、Clients构成一组ParameterServer，一台机器上只能部一个实例。
2. 同一组ParameterServer内，ServerManager、Servers、Clients的工作端口需要一致。
3. 两组ParameterServer的服务器可以存在交集，同一台服务器上的端口不要冲突就行。
## 关闭流程
1. 关闭所有Client。
2. 关闭所有Server。
    ```Shell
    ./stop.sh $port
    ```
3. 关闭ServerManager。
    ```Shell
    ./stop.sh $port
    ```