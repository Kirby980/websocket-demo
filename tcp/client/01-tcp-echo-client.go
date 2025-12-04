package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
)

/*
【示例1：TCP Echo 客户端 - Socket 编程基础】

这个客户端配合 01-tcp-echo-server.go 使用，演示：
1. 客户端如何建立 TCP 连接
2. 如何发送和接收数据
3. 客户端的基本流程：连接 -> 读写数据 -> 关闭连接

运行方式：
  1. 先运行服务器：go run 01-tcp-echo-server.go
  2. 再运行客户端：go run 01-tcp-echo-client.go
  3. 输入消息，回车发送，输入 "quit" 退出
*/

func main() {
	// 第一步：连接到服务器
	// net.Dial 做了什么？
	// 1. 创建一个 Socket（系统调用：socket()）
	// 2. 发起连接请求（系统调用：connect()）
	// 3. 完成 TCP 三次握手
	// 4. 返回建立好的连接
	port := flag.String("port", "8888", "port端口号")
	flag.Parse()
	conn, err := net.Dial("tcp", "localhost:"+*port)
	if err != nil {
		fmt.Printf("连接服务器失败: %v\n", err)
		fmt.Println("请确保服务器已启动（go run 01-tcp-echo-server.go）")
		os.Exit(1)
	}
	defer conn.Close() // 程序结束时关闭连接

	fmt.Println("已连接到服务器: localhost:" + *port)
	fmt.Println("输入消息后回车发送，输入 'quit' 退出")
	fmt.Println("----------------------------------------")

	// 创建两个读取器：
	// 1. 从标准输入（键盘）读取用户输入
	// 2. 从网络连接读取服务器响应
	stdinReader := bufio.NewReader(os.Stdin)
	connReader := bufio.NewReader(conn)

	// 主循环：读取用户输入 -> 发送到服务器 -> 接收服务器响应
	for {
		fmt.Println("请输入:")
		// 第二步：读取用户输入
		message, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入失败: %v\n", err)
			return
		}

		// 检查是否退出
		if message == "quit\n" {
			fmt.Println("再见！")
			return
		}

		// 第三步：发送数据到服务器
		// 底层系统调用：write() 或 send()
		_, err = conn.Write([]byte(message))
		if err != nil {
			fmt.Printf("发送数据失败: %v\n", err)
			return
		}

		// 第四步：接收服务器的响应
		// 底层系统调用：read() 或 recv()
		response, err := connReader.ReadString('\n')
		if err != nil {
			fmt.Printf("接收响应失败: %v\n", err)
			return
		}

		// 打印服务器的响应
		fmt.Printf("服务器响应: %s", response)
		fmt.Println("----------------------------------------")
	}
}

/*
【知识点总结】

1. Socket 编程的基本流程（客户端）：
   socket() -> connect() -> read()/write() -> close()

   在 Go 中被封装为：
   net.Dial() -> conn.Read()/conn.Write() -> conn.Close()

2. 客户端与服务器的区别：
   - 服务器：被动等待连接（Listen + Accept）
   - 客户端：主动发起连接（Dial）

3. TCP 的可靠性保证：
   - 数据按顺序到达
   - 丢失的数据会自动重传
   - 有错误检测机制

4. 常见问题：
   Q: 如果服务器没启动会怎样？
   A: net.Dial() 会返回 "connection refused" 错误

   Q: 如果网络中断会怎样？
   A: Read/Write 会返回错误，可以捕获并处理

   Q: 数据会丢失吗？
   A: TCP 保证可靠传输，但应用层要处理网络错误

【实验建议】

1. 先运行服务器，再运行客户端，观察连接建立过程

2. 尝试：
   - 先运行客户端（服务器未启动）- 看连接失败
   - 客户端连接后，关闭服务器 - 看连接断开的表现
   - 同时运行多个客户端 - 发现只有一个能工作（服务器是同步的）

3. 思考：如何让服务器同时处理多个客户端？
   提示：下一个示例会用 goroutine（Go 的协程）解决这个问题

【下一步学习】
02-tcp-chat-server.go - 支持多客户端的聊天服务器
学习如何用 goroutine 实现并发处理
*/
