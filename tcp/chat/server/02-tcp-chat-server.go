package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/*
【示例2：TCP 多客户端聊天服务器 - 并发处理】

这个示例演示：
1. 如何使用 goroutine 并发处理多个客户端
2. 客户端之间的消息广播（群聊）
3. 连接管理和资源清理

与示例1的区别：
- 示例1：同步处理，一次只能服务一个客户端
- 示例2：并发处理，可以同时服务多个客户端

运行方式：
  go run 02-tcp-chat-server.go

测试方式：
  打开多个终端，每个都运行：
  telnet localhost 9999
  或
  nc localhost 9999

  在任意一个终端输入消息，所有其他终端都能收到！
*/

// 客户端结构体
type Client struct {
	conn    net.Conn      // TCP 连接
	name    string        // 客户端名称
	writer  *bufio.Writer // 带缓冲的写入器，提高性能
	msgChan chan string
}

// 全局变量：管理所有连接的客户端
var (
	// clients 存储所有在线客户端
	//clients    = make(map[*Client]bool)
	newClients = sync.Map{}

	// clientsMutex 保护 clients map 的并发访问
	// 因为多个 goroutine 会同时读写这个 map
	//clientsMutex sync.Mutex

	// 客户端计数器，用于生成客户端名称
	clientCounter int32
)

func main() {
	listener, err := net.Listen("tcp", ":9999")
	if err != nil {
		fmt.Printf("监听失败: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("TCP 聊天服务器启动成功！")
	fmt.Println("监听地址: localhost:9999")
	fmt.Println("等待客户端连接...")
	fmt.Println("========================================")

	// 循环接受客户端连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("接受连接失败: %v\n", err)
			continue
		}

		// 【关键点1】为每个客户端启动一个新的 goroutine
		// 这样可以同时处理多个客户端，不会互相阻塞
		go handleClient(conn)
	}
}

// handleClient 处理单个客户端的连接
func handleClient(conn net.Conn) {
	reader := bufio.NewReader(conn)
	name, _ := reader.ReadString('\n')
	atomic.AddInt32(&clientCounter, 1)

	// 创建客户端对象
	//clientCounter++
	client := &Client{
		conn:    conn,
		name:    strings.TrimSpace(name),
		writer:  bufio.NewWriter(conn),
		msgChan: make(chan string, 100),
	}

	// 【关键点2】注册客户端到全局列表
	// 需要加锁，因为可能有多个 goroutine 同时修改 clients map

	// clientsMutex.Lock()
	// clients[client] = true
	// clientsMutex.Unlock()
	go client.writePump()
	// 连接建立时的提示
	fmt.Printf("[%s] %s 加入聊天室，来自 %s\n",
		getCurrentTime(), client.name, conn.RemoteAddr())

	// 向客户端发送欢迎消息
	client.sendMessage(fmt.Sprintf("欢迎来到聊天室！你的名字是：%s\n", client.name))
	client.sendMessage(fmt.Sprintf("当前在线人数：%d\n", clientCounter))
	newClients.Store(client, true)

	// 广播：通知所有其他客户端有新人加入
	broadcast(fmt.Sprintf("系统消息：%s 加入了聊天室\n", client.name), client)
	broadcast(fmt.Sprintf("当前在线人数：%d\n", clientCounter), client)

	// 【关键点3】确保连接关闭时清理资源
	defer func() {
		// 从客户端列表中移除
		// clientsMutex.Lock()
		// delete(clients, client)
		// clientsMutex.Unlock()
		atomic.AddInt32(&clientCounter, -1)
		newClients.Delete(client)
		close(client.msgChan)
		fmt.Printf("[%s] %s 离开聊天室\n", getCurrentTime(), client.name)

		// 广播：通知所有客户端有人离开
		broadcast(fmt.Sprintf("系统消息：%s 离开了聊天室\n", client.name), nil)
		broadcast(fmt.Sprintf("当前在线人数：%d\n", clientCounter), nil)

	}()

	// 循环读取客户端消息
	reader = bufio.NewReader(conn)
	//reader := bufio.NewReaderSize(conn, 5)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			// 客户端断开连接
			return
		}

		// 去除首尾空白字符
		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}

		// 打印到服务器控制台
		fmt.Printf("[%s] %s: %s\n", getCurrentTime(), client.name, message)

		// 【关键点4】广播消息给所有其他客户端
		broadcast(fmt.Sprintf("%s: %s\n", client.name, message), client)
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.msgChan {
		c.writer.WriteString(msg)
		c.writer.Flush()
	}
}

// broadcast 向所有客户端广播消息（可选择排除某个客户端）
func broadcast(message string, exclude *Client) {
	newClients.Range(func(key, value any) bool {
		client := key.(*Client)
		if client == exclude {
			return true
		}
		select {
		case client.msgChan <- message:
		default:
			fmt.Printf("client %s channel full, skip\n", client.name)
		}
		//client.sendMessage(message)
		return true
	})

	// // 遍历所有在线客户端
	// for client := range newClients {
	// 	// 排除指定的客户端（通常是消息发送者自己）
	// 	if client == exclude {
	// 		continue
	// 	}
	// 	// 发送消息
	// 	client.sendMessage(message)
	// }
}

// sendMessage 向单个客户端发送消息
func (c *Client) sendMessage(message string) {
	select {
	case c.msgChan <- message:
	default:
		// 队列满，丢弃消息或关闭连接
	}
	// // 使用带缓冲的 writer 提高性能
	// c.writer.WriteString(message)
	// c.writer.Flush() // 立即刷新缓冲区，确保消息发送出去
}

// getCurrentTime 获取当前时间的格式化字符串
func getCurrentTime() string {
	return time.Now().Format("15:04:05")
}

/*
【知识点总结】

1. goroutine 实现并发：
   - 每个客户端连接在独立的 goroutine 中处理
   - 不会互相阻塞，可以同时服务多个客户端
   - Go 的调度器会自动管理 goroutine

2. 并发安全问题：
   - 多个 goroutine 访问共享数据（clients map）
   - 必须用 Mutex（互斥锁）保护，防止数据竞争
   - Lock() 加锁，Unlock() 解锁，defer 确保一定会解锁

3. 资源管理：
   - defer 确保连接关闭和清理
   - 从 clients map 中移除断开的客户端
   - 防止内存泄漏

4. 消息广播模式：
   - 服务器作为中心节点
   - 接收任意客户端的消息
   - 转发给所有其他客户端

【常见问题】

Q: 为什么需要 Mutex？
A: map 不是并发安全的，多个 goroutine 同时读写会导致程序崩溃

Q: 如果不用 goroutine 会怎样？
A: 同一时间只能处理一个客户端，其他客户端会被阻塞

Q: 如何测试并发？
A: 同时开启多个 telnet/nc 客户端，互相发送消息

Q: 性能瓶颈在哪里？
A: 这个简单实现中，广播消息时会持有锁，高并发时可能成为瓶颈
   改进方案：使用 channel 进行消息传递（更高级的并发模式）

【实验建议】

1. 同时运行 3-5 个客户端，观察消息广播
2. 尝试断开某个客户端，看其他客户端是否收到通知
3. 发送大量消息，观察服务器的并发处理能力
4. 思考：如何添加私聊功能？如何限制消息长度？

【下一步学习】
理解了 TCP Socket 和并发处理后，可以学习 WebSocket 了
03-websocket-echo.go - WebSocket 协议的升级过程
*/
