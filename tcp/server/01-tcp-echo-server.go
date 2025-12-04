package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

/*
【示例1：TCP Echo 服务器 - Socket 编程基础】

这是最简单的 Socket 编程示例，用来理解以下核心概念：
1. Socket 是什么？- 网络通信的端点（endpoint）
2. TCP 连接的建立过程（三次握手在底层自动完成）
3. 服务器的基本流程：监听 -> 接受连接 -> 读写数据 -> 关闭连接

运行方式：
  go run 01-tcp-echo-server.go

测试方式：
  在另一个终端使用 telnet 或 nc 命令测试：
  telnet localhost 8888
  或
  nc localhost 8888

  然后输入任何文字，服务器会原样返回（Echo）
*/

func main() {
	// 第一步：创建监听器（Listener）
	// net.Listen 做了什么？
	// 1. 创建一个 Socket（系统调用：socket()）
	// 2. 绑定到指定地址和端口（系统调用：bind()）
	// 3. 开始监听连接请求（系统调用：listen()）
	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Printf("监听失败: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close() // 程序结束时关闭监听器

	fmt.Println("TCP Echo 服务器启动成功！")
	fmt.Println("监听地址: localhost:8888")
	fmt.Println("等待客户端连接...")
	fmt.Println("----------------------------------------")

	// 第二步：循环接受客户端连接
	for {
		// Accept() 会阻塞，直到有客户端连接进来
		// 底层系统调用：accept()
		// 当客户端连接时，会完成 TCP 三次握手，然后返回一个新的 Socket（conn）
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("接受连接失败: %v\n", err)
			continue // 继续等待下一个连接
		}

		fmt.Printf("新客户端连接: %s\n", conn.RemoteAddr())

		// 第三步：处理客户端请求
		// 注意：这里是同步处理，一次只能服务一个客户端
		// 后面的示例会演示如何并发处理多个客户端
		go handleClient(conn)
	}
}

// handleClient 处理单个客户端的连接
func handleClient(conn net.Conn) {
	defer conn.Close() // 函数结束时关闭连接
	defer fmt.Printf("客户端断开: %s\n", conn.RemoteAddr())

	// 创建一个带缓冲的读取器，方便按行读取
	reader := bufio.NewReader(conn)

	// 循环读取客户端发送的数据
	for {
		// 读取一行数据（以 \n 结尾）
		// 底层系统调用：read() 或 recv()
		message, err := reader.ReadString('\n')

		// 处理读取错误或客户端断开连接
		if err != nil {
			// err == io.EOF 表示客户端正常关闭连接
			if err.Error() != "EOF" {
				fmt.Printf("读取数据失败: %v\n", err)
			}
			return
		}

		// 打印接收到的消息
		fmt.Printf("收到消息: %s", message)

		// Echo：将收到的消息原样发送回客户端
		// 底层系统调用：write() 或 send()
		_, err = conn.Write([]byte("Echo: " + message))
		if err != nil {
			fmt.Printf("发送数据失败: %v\n", err)
			return
		}
	}
}

/*
【知识点总结】

1. Socket 编程的基本流程（服务端）：
   socket() -> bind() -> listen() -> accept() -> read()/write() -> close()

   在 Go 中被封装为：
   net.Listen() -> listener.Accept() -> conn.Read()/conn.Write() -> conn.Close()

2. TCP 连接是全双工的：
   - 可以同时读和写
   - 一个方向关闭不影响另一个方向（半关闭）

3. 这个服务器的局限性：
   - 同步处理：一次只能服务一个客户端
   - 如果一个客户端不断开，其他客户端无法连接
   - 下一个示例会用 goroutine 解决这个问题

4. 常见问题：
   Q: 为什么需要 defer conn.Close()？
   A: 防止连接泄漏。如果不关闭，会占用系统资源（文件描述符）

   Q: 端口被占用怎么办？
   A: 修改端口号，或者用 lsof -i :8888 查看占用进程并关闭

   Q: 如何停止服务器？
   A: Ctrl+C 发送 SIGINT 信号

【下一步学习】
运行这个服务器，用 telnet/nc 测试，观察：
1. 连接建立的过程
2. 数据的收发
3. 同时开两个客户端会发生什么？
*/
