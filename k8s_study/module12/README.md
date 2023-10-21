## 题目

把我们的 httpserver 服务以 Istio Ingress Gateway 的形式发布出来。以下是你需要考虑的几点：

- 如何实现安全保证；
- 七层路由规则；
- 考虑 open tracing 的接入。

## 操作

1. kubernetes集群中安装Istio

```shell
$ curl -L https://istio.io/downloadIstio | sh -
Downloading istio-1.19.3 from https://github.com/istio/istio/releases/download/1.19.3/istio-1.19.3-linux-amd64.tar.gz ...

$ cd istio-1.19.3/
$ cp bin/istioctl /usr/local/bin/
$ istioctl install --set profile=demo -y

The Kubernetes version v1.22.2 is not supported by Istio 1.19.3. The minimum supported Kubernetes version is 1.25.
Proceeding with the installation, but you might experience problems. See https://istio.io/latest/docs/setup/platform-setup/ for a list of supported versions.

The Kubernetes version v1.22.2 is not supported by Istio 1.19.3. The minimum supported Kubernetes version is 1.25.
Proceeding with the installation, but you might experience problems. See https://istio.io/latest/docs/setup/platform-setup/ for a list of supported versions.

✔ Istio core installed
✔ Istiod installed
✔ Ingress gateways installed
✔ Egress gateways installed
✔ Installation complete                                                                                                   Made this installation the default for injection and validation.

# 查看istio安装部署是否成功
$ kubectl get all -n istio-system
NAME                                       READY   STATUS    RESTARTS   AGE
pod/istio-egressgateway-b79dd7654-tfjpt    1/1     Running   0          3m51s
pod/istio-ingressgateway-b48c8d45b-6k62q   1/1     Running   0          3m51s
pod/istiod-6f856689fc-89cs8                1/1     Running   0          4m13s

NAME                           TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)                                                                      AGE
service/istio-egressgateway    ClusterIP      10.103.151.100   <none>        80/TCP,443/TCP                                                               3m51s
service/istio-ingressgateway   LoadBalancer   10.103.155.226   <pending>     15021:30469/TCP,80:30090/TCP,443:32386/TCP,31400:32153/TCP,15443:32054/TCP   3m51s
service/istiod                 ClusterIP      10.100.19.206    <none>        15010/TCP,15012/TCP,443/TCP,15014/TCP                                        4m13s

NAME                                   READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/istio-egressgateway    1/1     1            1           3m51s
deployment.apps/istio-ingressgateway   1/1     1            1           3m51s
deployment.apps/istiod                 1/1     1            1           4m13s

NAME                                             DESIRED   CURRENT   READY   AGE
replicaset.apps/istio-egressgateway-b79dd7654    1         1         1       3m51s
replicaset.apps/istio-ingressgateway-b48c8d45b   1         1         1       3m51s
replicaset.apps/istiod-6f856689fc                1         1         1       4m13s

```



2. 创建个新的namespace注入sidecar，用于后续httpserver部署

```shell
# 创建namespace模版
$ kubectl create namespace httpserver-istio --dry-run=client -o yaml
apiVersion: v1
kind: Namespace
metadata:
  creationTimestamp: null
  name: httpserver-istio
spec: {}
status: {}

# 修改namespace yaml 文件, 为了让envoy的sidecar容器可以自动注入到业务pod中，需要添加label "istio-injection: enable", 后续在该namespace上发布的Pod都会自动注入envoy
$ vi httpserver-istio-ns.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: httpserver-istio
  labels: 
    istio-injection: enabled
    
$ kubectl create -f httpserver-istio-ns.yaml
namespace/httpserver-istio created

```



3. 证书生成

```shell
# 假设对外暴露的域名是gohttpserver.io
$ openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -subj '/O=cncamp Inc./CN=*.gohttpserver.io' -keyout gohttpserver.io.key -out gohttpserver.io.crt

# 将证书发布到集群的namespace istio-system之下
$ kubectl create -n istio-system secret tls gohttpserver-credential --key=./ca/gohttpserver.io.key --cert=./ca/gohttpserver.io.crt
secret/gohttpserver-credential created

$ kubectl get secret -n istio-system -owide
NAME                                               TYPE                                  DATA   AGE
default-token-2gnkf                                kubernetes.io/service-account-token   3      35m
gohttpserver-credential                            kubernetes.io/tls                     2      35s
istio-ca-secret                                    istio.io/ca-root                      5      35m
istio-egressgateway-service-account-token-wrn5f    kubernetes.io/service-account-token   3      35m
istio-ingressgateway-service-account-token-8mngd   kubernetes.io/service-account-token   3      35m
istio-reader-service-account-token-94r6f           kubernetes.io/service-account-token   3      35m
istiod-token-pwxss                                 kubernetes.io/service-account-token   3      35m

```



4. 部署httpserver到namespace: httpserver-istio中

```shell
$ vi httpserver-istio-dep.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: gohttpserver
  name: gohttpserver
  # 定义命名空间为gohttpserver
  namespace: httpserver-istio
spec:
  # 为应用配置多个副本，确保应用的高可用
  replicas: 2
  selector:
    matchLabels:
      app: gohttpserver
  template:
    metadata:
      labels:
        app: gohttpserver
      annotations:
        prometheus.io/port: server-port
        prometheus.io/scrape: "true"
    spec:
      containers:
      - image: helenwami/gohttpserver:v1.3
        name: gohttpserver
        ports:
        - name: server-port
          containerPort: 81

$ vi httpserver-istio-cm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: httpserver-conf
  namespace: httpserver-istio
data:
  greeting.conf: |
    Welcome to gohttpserver

$ kubectl create -f httpserver-istio-cm.yaml
configmap/httpserver-conf created

$ kubectl create -f httpserver-istio-dep.yaml
deployment.apps/gohttpserver created

$ kubectl get pod -n httpserver-istio
NAME                            READY   STATUS    RESTARTS   AGE
gohttpserver-6bd6d4ddb7-k59wl   2/2     Running   0          12m
gohttpserver-6bd6d4ddb7-xdfsj   2/2     Running   0          12m

```



5. 服务发布，创建VirtualService and Ingress gateway

```shell
$ vi httpserver-istio-svc.yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    app: gohttpserver
  name: gohttpserver
  namespace: httpserver-istio
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 81
  selector:
    app: gohttpserver

$ kubectl create -f httpserver-istio-svc.yaml
service/gohttpserver created

# Create Istio Virtual Service
$ vi httpserver-istio-vs.yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: gohttpserver-svc
  namespace: httpserver-istio
spec:
  gateways:
    - gohttpserver-svc
  hosts:
    - gohttpserver.io
  http:
    - match:
        - uri:
            exact: /healthz
      route:
        - destination:
            host: gohttpserver
            port:
              number: 80

# 在gateway中指定对外暴露的是https服务，并为其指定的证书为证书生成步骤导入的证书gohttpserver-credential
$ vi httpserver-istio-gw.yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: gohttpserver-svc
  namespace: httpserver-istio
spec:
  selector:
    istio: ingressgateway
  servers:
  - hosts:
      - gohttpserver.io
    port:
      name:  gohttpserver-svc
      number: 443
      protocol: HTTPS
    tls:
      mode: SIMPLE
      credentialName: gohttpserver-credential



$ kubectl create -f httpserver-istio-vs.yaml
virtualservice.networking.istio.io/gohttpserver-svc created

$ kubectl create -f httpserver-istio-gw.yaml
gateway.networking.istio.io/gohttpserver-svc created


$ kubectl get svc -n istio-system
NAME                   TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)                                                                      AGE
istio-egressgateway    ClusterIP      10.103.151.100   <none>        80/TCP,443/TCP                                                               91m
istio-ingressgateway   LoadBalancer   10.103.155.226   <pending>     15021:30469/TCP,80:30090/TCP,443:32386/TCP,31400:32153/TCP,15443:32054/TCP   91m
istiod                 ClusterIP      10.100.19.206    <none>        15010/TCP,15012/TCP,443/TCP,15014/TCP                                        91m

# 查看istio-ingressgateway集群内的IP地址，并修改本机的/etc/hosts文件以便以域名的方式访问服务
$ vi /etc/hosts
10.103.155.226 gohttpserver.io

# 验证ssl访问
$ curl https://gohttpserver.io/healthz -v -k
*   Trying 10.103.155.226:443...
* TCP_NODELAY set
* Connected to gohttpserver.io (10.103.155.226) port 443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* successfully set certificate verify locations:
*   CAfile: /etc/ssl/certs/ca-certificates.crt
  CApath: /etc/ssl/certs
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
* TLSv1.3 (IN), TLS handshake, Server hello (2):
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (8):
* TLSv1.3 (IN), TLS handshake, Certificate (11):
* TLSv1.3 (IN), TLS handshake, CERT verify (15):
* TLSv1.3 (IN), TLS handshake, Finished (20):
* TLSv1.3 (OUT), TLS change cipher, Change cipher spec (1):
* TLSv1.3 (OUT), TLS handshake, Finished (20):
* SSL connection using TLSv1.3 / TLS_AES_256_GCM_SHA384
* ALPN, server accepted to use h2
* Server certificate:
*  subject: O=cncamp Inc.; CN=*.gohttpserver.io
*  start date: Oct 21 09:48:14 2023 GMT
*  expire date: Oct 20 09:48:14 2024 GMT
*  issuer: O=cncamp Inc.; CN=*.gohttpserver.io
*  SSL certificate verify result: self signed certificate (18), continuing anyway.
* Using HTTP2, server supports multi-use
* Connection state changed (HTTP/2 confirmed)
* Copying HTTP/2 data in stream buffer to connection buffer after upgrade: len=0
* Using Stream ID: 1 (easy handle 0x564540987320)
> GET /healthz HTTP/2
> Host: gohttpserver.io
> user-agent: curl/7.68.0
> accept: */*
>
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
* old SSL session ID is stale, removing
* Connection state changed (MAX_CONCURRENT_STREAMS == 2147483647)!
< HTTP/2 200
< date: Sat, 21 Oct 2023 11:08:04 GMT
< content-length: 3
< content-type: text/plain; charset=utf-8
< x-envoy-upstream-service-time: 6
< server: istio-envoy
<
* Connection #0 to host gohttpserver.io left intact
```



6. Open Telemetry的接入

```shell
# 安装Jaeger组件
$ kubectl apply -f jaeger.yaml
deployment.apps/jaeger created
service/tracing created
service/zipkin created
service/jaeger-collector created

# 查看istio-system中jaeger组件安装成功
$ kubectl get all -n istio-system
NAME                                       READY   STATUS    RESTARTS   AGE
pod/istio-egressgateway-b79dd7654-tfjpt    1/1     Running   0          119m
pod/istio-ingressgateway-b48c8d45b-6k62q   1/1     Running   0          119m
pod/istiod-6f856689fc-89cs8                1/1     Running   0          120m
pod/jaeger-5d44bc5c5d-jqsqr                1/1     Running   0          38s

NAME                           TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)                                                                      AGE
service/istio-egressgateway    ClusterIP      10.103.151.100   <none>        80/TCP,443/TCP                                                               119m
service/istio-ingressgateway   LoadBalancer   10.103.155.226   <pending>     15021:30469/TCP,80:30090/TCP,443:32386/TCP,31400:32153/TCP,15443:32054/TCP   119m
service/istiod                 ClusterIP      10.100.19.206    <none>        15010/TCP,15012/TCP,443/TCP,15014/TCP                                        120m
service/jaeger-collector       ClusterIP      10.106.23.147    <none>        14268/TCP,14250/TCP,9411/TCP                                                 38s
service/tracing                ClusterIP      10.103.244.231   <none>        80/TCP,16685/TCP                                                             38s
service/zipkin                 ClusterIP      10.105.47.201    <none>        9411/TCP                                                                     38s

NAME                                   READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/istio-egressgateway    1/1     1            1           119m
deployment.apps/istio-ingressgateway   1/1     1            1           119m
deployment.apps/istiod                 1/1     1            1           120m
deployment.apps/jaeger                 1/1     1            1           38s

NAME                                             DESIRED   CURRENT   READY   AGE
replicaset.apps/istio-egressgateway-b79dd7654    1         1         1       119m
replicaset.apps/istio-ingressgateway-b48c8d45b   1         1         1       119m
replicaset.apps/istiod-6f856689fc                1         1         1       120m
replicaset.apps/jaeger-5d44bc5c5d                1         1         1       38s
```



7. 配置采样率需要更新的是 istio-system 下的configmap中的istio配置中，在tracing中加入"sampling: 100.0"，代表追踪所有的请求

```shell
$ kubectl get cm -n istio-system
NAME                                  DATA   AGE
istio                                 2      124m
istio-ca-root-cert                    1      124m
istio-gateway-status-leader           0      124m
istio-leader                          0      124m
istio-namespace-controller-election   0      124m
istio-sidecar-injector                2      124m
kube-root-ca.crt                      1      124m
$ kubectl describe cm istio -n istio-system
Name:         istio
Namespace:    istio-system
Labels:       install.operator.istio.io/owning-resource=installed-state
              install.operator.istio.io/owning-resource-namespace=istio-system
              istio.io/rev=default
              operator.istio.io/component=Pilot
              operator.istio.io/managed=Reconcile
              operator.istio.io/version=1.19.3
              release=istio
Annotations:  <none>

Data
====
mesh:
----
accessLogFile: /dev/stdout
defaultConfig:
  discoveryAddress: istiod.istio-system.svc:15012
  proxyMetadata: {}
  tracing:
    zipkin:
      address: zipkin.istio-system:9411
defaultProviders:
  metrics:
  - prometheus
enablePrometheusMerge: true
extensionProviders:
- envoyOtelAls:
    port: 4317
    service: opentelemetry-collector.istio-system.svc.cluster.local
  name: otel
- name: skywalking
  skywalking:
    port: 11800
    service: tracing.istio-system.svc.cluster.local
- name: otel-tracing
  opentelemetry:
    port: 4317
    service: opentelemetry-collector.otel-collector.svc.cluster.local
rootNamespace: istio-system
trustDomain: cluster.local
meshNetworks:
----
networks: {}

BinaryData
====

Events:  <none>

$ kubectl edit cm istio -n istio-system
apiVersion: v1
data:
    mesh: |-
        accessLogFile: /dev/stdout
        defaultConfig:
        discoveryAddress: istiod.istio-system.svc:15012
        proxyMetadata: {}
        tracing:
            sampling: 100.0
```



8. 打开Jaeger的Dashboard，查看Jaeger记录的链路追踪

```shell
# 多访问几遍httpserver
   curl https://gohttpserver.io/healthz -v
   curl https://gohttpserver.io/healthz -v -k
   curl https://gohttpserver.io/healthz -v -k
   curl https://gohttpserver.io/metrics -v -k
   curl https://gohttpserver.io/getVersion -v -k
   curl https://gohttpserver.io/metrics -v -k
   curl https://gohttpserver.io/healthz -v -k
   curl https://gohttpserver.io/metrics -v -k
   curl https://gohttpserver.io/healthz -v -k
   curl https://gohttpserver.io/metrics -v -k
   
$ istioctl dashboard jaeger
http://localhost:16686
```

