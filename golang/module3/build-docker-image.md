1. 构建镜像

   ```shell
   $ vim Dockerfile
   $ sudo docker build -t gohttpserver:v1 .
   ```

2. 推送镜像至hub.docker.com

   官方给出了推送的命令

   ```shell
   docker tag local-image:tagname new-repo:tagname
   docker push new-repo:tagname
   ```

   ```shell
   $ sudo docker login
   
   $ sudo docker tag gohttpserver:v1 helenwami/gohttpserver:v1
   $ sudo docker push helenwami/gohttpserver:v1
   The push refers to repository [docker.io/helenwami/gohttpserver]
   28a09da6317b: Pushed
   9f54eef41275: Pushed
   v1: digest: sha256:a92849ec983cd51988aa92462b0f55e73fdb77202975b7e38b3322853152a735 size: 740
   ```



3. 启动容器镜像

   ```shell
   sudo docker run -p 81:81 helenwami/gohttpserver:v1
   ```

   浏览器访问http://xx.xx.xx.xx:81/healthz

4. 查看docker镜像PID，根据PID进入Docker容器的网络命名空间

   ```shell
   $ sudo docker ps -a
   [sudo] password for minwang:
   CONTAINER ID   IMAGE                                               COMMAND                  CREATED         STATUS                     PORTS                                       NAMES
   d718300d5f7d   helenwami/gohttpserver:v1                                "/bin/sh -c /gohttps…"   3 minutes ago   Up 3 minutes
   
   $ sudo docker container top d718300d5f7d
   UID                 PID                 PPID                C                   STIME               TTY                 TIME                CMD
   root                3227002             3226975             0                   21:50               ?                   00:00:00            /bin/sh -c /gohttpserver
   root                3227034             3227002             0                   21:50               ?                   00:00:00            /gohttpserver
   
   $ sudo docker inspect -f {{.State.Pid}} d718300d5f7d
   3227002
   
   $ sudo nsenter -n -t3227002
   root@master:/home/minwang/go/src/cncamp/golang/module3# ip a
   1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
       link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
       inet 127.0.0.1/8 scope host lo
          valid_lft forever preferred_lft forever
   58: eth0@if59: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default
       link/ether 02:42:ac:11:00:05 brd ff:ff:ff:ff:ff:ff link-netnsid 0
       inet 172.17.0.5/16 brd 172.17.255.255 scope global eth0
          valid_lft forever preferred_lft forever
   ```

   

5. 重启容器PID有变化，容器ip无变化

   

