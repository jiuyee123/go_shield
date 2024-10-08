# go_shield
基于Golang实现的一个高可用、高并发的TCP网关服务。

## 系统架构图
![alt text](images/image.png)

基于这个架构图，我们可以看到，goshield 是一个基于Golang实现的TCP网关服务。它主要由以下几个部分组成：
- 负载均衡器： 负责请求的负载均衡，将请求分发到多个服务器。
    - 负载均衡算法： 负责请求的负载均衡，将请求分发到多个服务器。
    - 健康检查机制： 负责检查服务器的健康状态，如果服务器不健康，则将请求分发到其他的服务器。
- 网关实例： 负责处理请求，并将请求转发到对应的服务，需要实现的核心功能包括：
    - 流量控制：使用令牌桶算法，实现请求的限流。
    - 连接池管理：维护与后端服务的长链接池。
    - 协议转换：如果需要，在不同协议间进行转换。
    - 安全性：使用TLS加密，实现数据的安全传输。

- 分布式存储:

    - 使用Redis集群或etcd存储网关的配置信息和运行时状态。
    - 确保网关实例之间的状态一致性。


- 监控系统:
    - 使用Prometheus + Grafana监控整个系统的性能指标。
    - 设置告警阈值,及时发现并处理异常情况。


- 高可用性:
    - 使用Keepalived实现负载均衡器的故障转移。
    - 网关实例采用无状态设计,便于水平扩展。


- 高并发:
    - 利用异步I/O和事件驱动编程模型。
    - 使用连接池复用TCP连接。
    - 采用多线程处理,充分利用多核CPU资源。


## 负载均衡器的实现
常用的负载均衡算法有轮询、最小连接数、最小响应时间等。
在我们系统的实际业务中，使用轮询就能满足了。因此，我们只需要实现轮询算法。
### 轮询
将请求依次分配给每个后端服务器，依次循环，直到所有服务器都被分配一遍，然后重新开始。
- 优点：
    - 实现简单，不需要额外的状态信息。
    - 可以保证请求的均匀分配。
- 缺点：
    - 不能保证请求的均匀分配。
    - 不能保证请求的响应时间。

### 最小连接数
将请求分配给当前连接数最少的服务器。
- 优点：
    - 动态平衡负载，适用于请求处理时间不均的场景。
    - 可以保证请求的均匀分配。
- 缺点：
    - 维护当前连接数的开销较大，可能无法处理短连接高并发的场景。

### 最小响应时间 
将请求分配给响应时间最短的服务器，
- 优点：
    - 可以保证请求的均匀分配。
    - 可以保证请求的响应时间。
- 缺点：
    - 维护响应时间的开销较大，可能无法处理短连接高并发的场景。

## 令牌桶算法实现请求限流
令牌桶算法是一种常见的限流技术，用于控制请求的速率

### 令牌桶算法简介
令牌桶算法通过以下方式控制请求的速率：

- 令牌生成：系统以恒定的速率往桶中添加令牌，直到桶满为止。
- 请求处理：每个请求到来时，必须从桶中取出一个令牌才能被处理。如果桶中没有令牌，则请求被拒绝或等待。
- 桶的容量：桶的容量限制了在短时间内可以处理的最大请求数。

## 关于心跳检测（待实现1）
网关层需要实现客户端连接的长连接保持，如果客户端长时间没有发送请求或者心跳，则需要关闭这个连接。
网关层主要实现的连接管理策略：
- 连接空闲超时: 在网关层设置一个全局的空闲超时时间（比如60秒、120秒等），如果客户端在此时间内没有发送任何数据（包括请求和心跳包），则自动关闭连接。
- 心跳机制: 通过心跳包来维持连接的活跃状态。如果在多个心跳周期内没有收到客户端的心跳包，可以认为连接已经失效，进行关闭。

## 关于鉴权（待实现2）
首先需要说明的是，这里的鉴权并不是业务系统里面的用户登录鉴权，这里的鉴权主要是通过主要是通过请求里面添加了一系列的约定的操作流程以及一些必要的微信信息降低非法请求的风险，提高系统的安全性。

为了提高安全性，要求业务系统中需要实现以下功能：
- 客户端首次连接时，基于设备的完整MAC地址，通过SHA-256哈希生成一个 client_id。
- 服务端生成一个UUID作为Token，绑定 client_id 和 Token 及其过期时间，并下发给客户端。
- 客户端后续请求中携带Token，服务端校验Token的有效性、匹配性和过期时间，校验失败则关闭连接。
- 心跳包中，服务端可以随机发放新的Token，同时使旧Token失效。
- 整个通信过程通过TLS/SSL加密，并且对所有鉴权相关的操作进行日志记录和监控。


## Demo的使用
代码放在`demo`目录下，分为`server`和`client`两个目录。
- `server`目录下是服务端的代码，使用`net`包的`Listen`函数创建一个TCP监听器，然后在一个goroutine中接受连接请求，并将这些请求传递给`TCPLoadBalancer`的`Accept`方法。
- `client`目录下是客户端的代码，使用`net`包的`Dial`函数创建一个TCP连接，然后在一个goroutine中发送请求，并将这些请求传递给`TCPLoadBalancer`的`Send`方法。    

首先启动server下的代码
```
go run server.go 8081
go run server.go 8082
go run server.go 8083
```

然后启动load_balancer下的代码
```
go run load_balancer.go
```

然后启动client下的代码
```
go run client.go localhost 8080
```

Demo 里面的消息格式为;
| 版本号(1字节) | 消息类型(1字节) | 数据长度(2字节) | 数据内容(可变长度) |
可以选择使用xml，json，还有protobuf等，这里使用的是json