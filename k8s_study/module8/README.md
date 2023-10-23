### 编写 Kubernetes 部署脚本将 httpserver 部署到 Kubernetes 集群

1. 创建deployment YAML模版

   ```shell
   root@master:~# kubectl create deploy httpserver --image=helenwami/gohttpserver:v1 --dry-run=client -o yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     creationTimestamp: null
     labels:
       app: httpserver
     name: httpserver
   spec:
     replicas: 1
     selector:
       matchLabels:
         app: httpserver
     strategy: {}
     template:
       metadata:
         creationTimestamp: null
         labels:
           app: httpserver
       spec:
         containers:
         - image: helenwami/gohttpserver:v1
           name: gohttpserver
           resources: {}
   status: {}
   ```

   

2. 查看gohttpserver image使用的资源情况

   ```shell
   # docker run --name gohttpserver -p 81:81 -d helenwami/gohttpserver:v1
   # docker stats gohttpserver
   CONTAINER ID   NAME           CPU %     MEM USAGE / LIMIT     MEM %     NET I/O     BLOCK I/O   PIDS
   1c063f06f865   gohttpserver   0.00%     2.309MiB / 14.96GiB   0.02%     866B / 0B   0B / 0B     7
   ```

   

3. 修改deployment模版文件，思考维度
   * 优雅启动
   * 优雅终止
   * 资源需求和 QoS 保证
   * 探活
   * 日常运维需求，日志等级 (未实现)
   * 配置和代码分离

     ```yaml
     # vi httpserver-dep.yaml
     apiVersion: apps/v1
     kind: Deployment
     metadata:
       labels:
         app: httpserver
       name: httpserver
       # 定义命名空间为gohttpserver
       namespace: gohttpserver
     spec:
       # 为应用配置多个副本，确保应用的高可用
       replicas: 2
       selector:
         matchLabels:
           app: httpserver
       template:
         metadata:
           labels:
             app: httpserver
         spec:
           containers:
           - image: helenwami/gohttpserver:v1
             name: gohttpserver
             ports:
             - name: server-port
               containerPort: 81
             # 以volume挂载的方式分离配置文件,明文配置文件放入ConfigMap中
             volumeMounts:
             - mountPath: /etc/httpserver.d
               name: httpserver-conf-vol
             resources:
               requests:
                 cpu: "4m"
                 memory: "20Mi"
               limits:
                 memory: "1Gi"
                 cpu: "100m"
             # 探活指针，判断pod是否存活
             livenessProbe:
               httpGet:
                 path: /healthz
                 port: server-port
               initialDelaySeconds: 1
               periodSeconds: 5
             # Readiness指针，判断容器是否准备好接收访问请求
             readinessProbe:
               httpGet:
                 path: /healthz
                 port: server-port
               initialDelaySeconds: 2
               periodSeconds: 4
             # 优雅终止时间50秒
           terminationGracePeriodSeconds: 50
           volumes:
           - name: httpserver-conf-vol
             configMap:
               name: httpserver-conf
     ```



4. 生成configMap YAML模版文件

   ```shell
   # kubectl create cm httpserver-conf --from-literal=k=v --dry-run=client -o yaml
   apiVersion: v1
   data:
     k: v
   kind: ConfigMap
   metadata:
     creationTimestamp: null
     name: httpserver-conf
   ```

   

5. 修改ConfigMap文件, 并创建configmap "httpserver-conf"

   ```shell
   # vi httpserver-cm.yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: httpserver-conf
     namespace: gohttpserver
   data:
     greeting.conf: |
       Welcome to gohttpserver
       
   # kubectl create namespace gohttpserver
   namespace/gohttpserver created
   # kubectl apply -f httpserver-cm.yaml
   configmap/httpserver-conf created
   # kubectl get cm -n gohttpserver
   NAME               DATA   AGE
   httpserver-conf    1      68s
   kube-root-ca.crt   1      86s
   ```

   

6. 使用完善后的deployment yaml创建httpserver pod

   ```shell
   # kubectl apply -f httpserver-dep.yaml
   deployment.apps/httpserver created
   # kubectl get pod -n gohttpserver
   NAME                         READY   STATUS    RESTARTS   AGE
   httpserver-9d78c8655-6zpts   1/1     Running   0          2m14s
   httpserver-9d78c8655-qdmwp   1/1     Running   0          2m14s
   ```

   

7. 除了将 httpServer 应用优雅的运行在 Kubernetes 之上，我们还应该考虑如何将服务发布给对内和对外的调用方。
   来尝试用 Service, Ingress 将你的服务发布给集群外部的调用方吧。
   在第一部分的基础上提供更加完备的部署 spec，包括（不限于）：

   - Service
   - Ingress

   可以考虑的细节

   - 如何确保整个应用的高可用。
   - 如何通过证书保证 httpServer 的通讯安全。

8. 为httpserver生成service yaml模版，修改后创建service

   ```shell
   # kubectl expose deploy httpserver --port=80 --target-port=81 -n gohttpserver --dry-run=client -o yaml
   apiVersion: v1
   kind: Service
   metadata:
     creationTimestamp: null
     labels:
       app: httpserver
     name: httpserver
     namespace: gohttpserver
   spec:
     ports:
     - port: 80
       protocol: TCP
       targetPort: 81
     selector:
       app: httpserver
   status:
     loadBalancer: {}
     
   # vi httpserver-svc.yaml
   apiVersion: v1
   kind: Service
   metadata:
     labels:
       app: httpserver
     name: httpserver
     namespace: gohttpserver
   spec:
     ports:
     - port: 80
       protocol: TCP
       targetPort: 81
     selector:
       app: httpserver
   
   # kubectl apply -f httpserver-svc.yaml
   service/httpserver created
   
   # kubectl get svc -n gohttpserver
   NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
   httpserver   ClusterIP   10.111.136.12   <none>        80/TCP    38s
   
   # kubectl describe svc -n gohttpserver
   Name:              httpserver
   Namespace:         gohttpserver
   Labels:            app=httpserver
   Annotations:       <none>
   Selector:          app=httpserver
   Type:              ClusterIP
   IP Family Policy:  SingleStack
   IP Families:       IPv4
   IP:                10.111.136.12
   IPs:               10.111.136.12
   Port:              <unset>  80/TCP
   TargetPort:        81/TCP
   Endpoints:         192.168.219.95:81,192.168.219.96:81
   Session Affinity:  None
   Events:            <none>
   
   # kubectl get pod -o wide -n gohttpserver
   NAME                         READY   STATUS    RESTARTS   AGE   IP               NODE     NOMINATED NODE   READINESS GATES
   httpserver-9d78c8655-6zpts   1/1     Running   0          26m   192.168.219.96   master   <none>           <none>
   httpserver-9d78c8655-qdmwp   1/1     Running   0          26m   192.168.219.95   master   <none>           <none>
   
   # curl 10.111.136.12/healthz
   200root@master:~/ymlfile#
   
   ```

   

8. 创建ingress，生成证书并添加到ingress中

   ```shell
   # 部署nginx-ingress-controller
   
   # kubectl apply -f nginx-ingress-deployment.yaml
   namespace/ingress-nginx created
   serviceaccount/ingress-nginx created
   configmap/ingress-nginx-controller created
   clusterrole.rbac.authorization.k8s.io/ingress-nginx created
   clusterrolebinding.rbac.authorization.k8s.io/ingress-nginx created
   role.rbac.authorization.k8s.io/ingress-nginx created
   rolebinding.rbac.authorization.k8s.io/ingress-nginx created
   service/ingress-nginx-controller-admission created
   service/ingress-nginx-controller created
   deployment.apps/ingress-nginx-controller created
   ingressclass.networking.k8s.io/nginx created
   validatingwebhookconfiguration.admissionregistration.k8s.io/ingress-nginx-admission created
   serviceaccount/ingress-nginx-admission created
   clusterrole.rbac.authorization.k8s.io/ingress-nginx-admission created
   clusterrolebinding.rbac.authorization.k8s.io/ingress-nginx-admission created
   role.rbac.authorization.k8s.io/ingress-nginx-admission created
   rolebinding.rbac.authorization.k8s.io/ingress-nginx-admission created
   job.batch/ingress-nginx-admission-create created
   job.batch/ingress-nginx-admission-patch created
   
   # kubectl create ingress httpserver-ing --class=nginx --rule="httpserver.test/=httpserver:80" --dry-run=client -o yaml -n gohttpserver
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     creationTimestamp: null
     name: httpserver-ing
     namespace: gohttpserver
   spec:
     ingressClassName: nginx
     rules:
     - host: httpserver.test
       http:
         paths:
         - backend:
             service:
               name: httpserver
               port:
                 number: 80
           path: /
           pathType: Exact
   status:
     loadBalancer: {}
   
   # 生成证书
   # openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=httpserver.test/O=httpserver" -addext "subjectAltName = DNS:httpserver.test"
   Generating a RSA private key
   ...............+++++
   ......................................+++++
   writing new private key to 'tls.key'
   -----
   root@master:~/ymlfile# ls -al
   total 32
   drwxr-xr-x  2 root root 4096 Oct 17 11:54 .
   drwx------ 12 root root 4096 Oct 17 11:43 ..
   -rw-r--r--  1 root root  144 Oct 17 10:20 httpserver-cm.yaml
   -rw-r--r--  1 root root 1478 Oct 17 10:57 httpserver-dep.yaml
   -rw-r--r--  1 root root  318 Oct 17 11:43 httpserver-ing.yaml
   -rw-r--r--  1 root root  211 Oct 17 11:21 httpserver-svc.yaml
   -rw-r--r--  1 root root 1224 Oct 17 11:54 tls.crt
   -rw-------  1 root root 1708 Oct 17 11:54 tls.key
   
   # 证书存放到secret中
   # kubectl create secret tls httpserver-tls --cert=./tls.crt --key=./tls.key -n gohttpserver
   secret/httpserver-tls created
   
   # ingress中添加tls证书
   # vi httpserver-ing.yaml
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     name: httpserver-ing
     annotations:
       kubernetes.io/ingress.class: "nginx"
     namespace: gohttpserver
   spec:
     tls:
       - hosts:
           - httpserver.test
         secretName: httpserver-tls
     rules:
     - host: httpserver.test
       http:
         paths:
         - backend:
             service:
               name: httpserver
               port:
                 number: 80
           path: /
           pathType: Prefix
   
   # kubectl apply -f httpserver-ing.yaml
   Error from server (InternalError): error when creating "httpserver-ing.yaml": Internal error occurred: failed calling webhook "validate.nginx.ingress.kubernetes.io": Post "https://ingress-nginx-controller-admission.ingress-nginx.svc:443/networking/v1/ingresses?timeout=10s": dial tcp 10.97.27.231:443: connect: connection refused
   
   # kubectl get validatingwebhookconfigurations
   NAME                           WEBHOOKS   AGE
   ingress-nginx-admission        1          22m
   istio-validator-istio-system   1          42h
   istiod-default-validator       1          42h
   
   # kubectl delete -A ValidatingWebhookConfiguration ingress-nginx-admission
   validatingwebhookconfiguration.admissionregistration.k8s.io "ingress-nginx-admission" deleted
   
   # kubectl apply -f httpserver-ing.yaml
   ingress.networking.k8s.io/httpserver-ing created
   
   # # kubectl get ingress httpserver-ing -n gohttpserver
   NAME             CLASS    HOSTS             ADDRESS   PORTS     AGE
   httpserver-ing   <none>   httpserver.test             80, 443   82s
   
   # kubectl describe ingress httpserver-ing -n gohttpserver
   Name:             httpserver-ing
   Namespace:        gohttpserver
   Address:
   Default backend:  default-http-backend:80 (<error: endpoints "default-http-backend" not found>)
   TLS:
     httpserver-tls terminates httpserver.test
   Rules:
     Host             Path  Backends
     ----             ----  --------
     httpserver.test
                      /   httpserver:80 (192.168.219.114:81,192.168.219.115:81)
   Annotations:       kubernetes.io/ingress.class: nginx
   Events:            <none>
   
   # 在本地hosts中添加httpserver.test域名解析
   # vi /etc/hosts
   10.111.136.12 httpserver.test
   
   # curl http://httpserver.test/healthz -v -k
   *   Trying 10.111.136.12:80...
   * TCP_NODELAY set
   * Connected to httpserver.test (10.111.136.12) port 80 (#0)
   > GET /healthz HTTP/1.1
   > Host: httpserver.test
   > User-Agent: curl/7.68.0
   > Accept: */*
   >
   * Mark bundle as not supporting multiuse
   < HTTP/1.1 200 OK
   < Date: Tue, 17 Oct 2023 04:10:37 GMT
   < Content-Length: 3
   < Content-Type: text/plain; charset=utf-8
   <
   * Connection #0 to host httpserver.test left intact
   200
   
   # kubectl describe secret httpserver-tls -n gohttpserver
   Name:         httpserver-tls
   Namespace:    gohttpserver
   Labels:       <none>
   Annotations:  <none>
   
   Type:  kubernetes.io/tls
   
   Data
   ====
   tls.crt:  1224 bytes
   tls.key:  1708 bytes
   
   # curl https://httpserver.test/healthz -v -k
   *   Trying 10.111.136.12:443...
   * TCP_NODELAY set
   
   
   ```
   
   

