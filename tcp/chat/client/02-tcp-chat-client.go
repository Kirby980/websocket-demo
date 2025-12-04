package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

/*
【示例2：TCP 聊天客户端 - 并发收发消息】

这个客户端配合 02-tcp-chat-server.go 使用，演示：
1. 如何同时接收和发送消息（并发处理）
2. 多个客户端之间的消息广播
3. 客户端的优雅退出

与示例1的区别：
- 示例1：同步模式，发送后等待响应
- 示例2：异步模式，可以随时收发消息（类似真实的聊天应用）

运行方式：
  1. 先运行服务器：go run 02-tcp-chat-server.go
  2. 运行多个客户端：go run 02-tcp-chat-client.go
  3. 在不同客户端输入消息，观察群聊效果
  4. 输入 "quit" 退出
*/

func main() {
	// 第一步：连接到聊天服务器
	conn, err := net.Dial("tcp", "localhost:9999")
	if err != nil {
		fmt.Printf("连接服务器失败: %v\n", err)
		fmt.Println("请确保服务器已启动（go run 02-tcp-chat-server.go）")
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("已连接到聊天服务器: localhost:9999")
	fmt.Println("========================================")

	// 第二步：输入用户名
	stdinReader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入你的名字: ")
	name, err := stdinReader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取名字失败: %v\n", err)
		return
	}

	// 发送名字到服务器
	_, err = conn.Write([]byte(name))
	if err != nil {
		fmt.Printf("发送名字失败: %v\n", err)
		return
	}

	fmt.Println("========================================")
	fmt.Println("进入聊天室！输入消息后回车发送，输入 'quit' 退出")
	fmt.Println("========================================")

	// 【关键点1】创建一个 channel 用于通知程序退出
	// 当用户输入 quit 或者连接断开时，通过这个 channel 通知主程序退出
	done := make(chan bool, 1)
	var wg sync.WaitGroup
	// 【关键点2】启动接收消息的 goroutine
	// 这个 goroutine 会一直监听服务器发来的消息
	// 与主 goroutine（处理用户输入）并行运行
	wg.Add(1)
	go func() {
		defer wg.Done()
		receiveMessages(conn, done)
	}()

	// 【关键点3】主 goroutine 处理用户输入
	// 读取用户输入并发送到服务器
	for {
		// 读取用户输入
		message, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Printf("\n读取输入失败: %v\n", err)
			break
		}

		// 去除首尾空白
		message = strings.TrimSpace(message)

		// 检查是否退出
		if message == "quit" {
			fmt.Println("\n再见！")
			// 【关键点4】通知接收 goroutine 退出
			done <- true
			break
		}

		// 忽略空消息
		if message == "" {
			continue
		}

		// 发送消息到服务器
		_, err = conn.Write([]byte(message + "\n"))
		if err != nil {
			fmt.Printf("\n发送消息失败: %v\n", err)
			break
		}
	}
	wg.Wait()
}

// receiveMessages 持续接收服务器发来的消息
// 这个函数在独立的 goroutine 中运行
func receiveMessages(conn net.Conn, done chan bool) {
	reader := bufio.NewReader(conn)

	for {
		// 检查是否需要退出
		select {
		case <-done:
			// 收到退出信号
			return
		default:
			// 继续接收消息
		}

		// 设置读取超时，避免阻塞太久
		// 这样可以定期检查 done channel
		//conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		// 从服务器读取消息
		message, err := reader.ReadString('\n')
		if err != nil {
			// 检查是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时不是错误，继续循环
				continue
			}

			// 其他错误（连接断开等）
			fmt.Printf("\n连接已断开: %v\n", err)
			break
		}
		// 打印收到的消息
		// 使用 \r 先回到行首，清除当前输入行，打印消息，再重新显示提示符
		fmt.Printf("\r%s", message)

	}
}

/*
【知识点总结】

1. 并发模型 - 两个 goroutine 协同工作：
   - 主 goroutine：读取用户输入 -> 发送到服务器
   - 接收 goroutine：从服务器接收消息 -> 显示到屏幕
   - 两者同时运行，互不阻塞

2. 为什么需要并发？
   - 如果不用并发：发送消息时无法接收，接收消息时无法发送
   - 用了并发：可以随时收发，就像真实的聊天应用

3. goroutine 之间的通信：
   - 使用 channel (done) 传递退出信号
   - "不要通过共享内存来通信，而要通过通信来共享内存" - Go 的并发哲学

4. select 语句的作用：
   - 同时等待多个 channel 操作
   - 这里用于检查是否收到退出信号
   - 类似于网络编程中的 select/epoll 系统调用

【与服务器的交互流程】

1. 客户端连接服务器
2. 客户端发送用户名
3. 服务器注册客户端，发送欢迎消息
4. 服务器广播"某某加入聊天室"给其他客户端
5. 客户端A发送消息 -> 服务器 -> 广播给客户端B、C、D...
6. 客户端断开 -> 服务器广播"某某离开聊天室"

【常见问题】

Q: 为什么打印消息时用 \r？
A: \r 将光标移到行首，可以覆盖当前正在输入的内容，
   避免新消息和用户输入混在一起

Q: 如果服务器断开会怎样？
A: reader.ReadString() 会返回错误，接收 goroutine 退出
   用户下次输入时，Write() 也会失败，主程序退出

Q: 为什么用 channel 而不是共享变量？
A: channel 是 Go 推荐的并发通信方式，更安全，避免竞态条件

Q: 能否同时运行多个客户端？
A: 可以！打开多个终端，每个运行一个客户端实例

【实验建议】

1. 基础测试：
   - 运行服务器和2个客户端
   - 在客户端A输入消息，观察客户端B是否收到
   - 关闭客户端A，观察客户端B是否收到"离开"通知

2. 并发测试：
   - 运行5个客户端
   - 所有客户端快速发送消息
   - 观察消息是否都能正确广播

3. 异常测试：
   - 客户端连接后，关闭服务器，看客户端如何处理
   - 输入很长的消息，测试缓冲区处理
   - 快速输入大量消息，测试网络拥塞处理

【与示例1的对比】

示例1（Echo客户端）：
  发送 -> 等待响应 -> 打印 -> 发送 -> ...
  （同步，单线程）

示例2（Chat客户端）：
  发送线程：用户输入 -> 发送 -> 用户输入 -> ...
  接收线程：接收 -> 打印 -> 接收 -> 打印 -> ...
  （异步，多线程）

【改进方向】

1. 添加命令支持：
   /list - 列出在线用户
   /quit - 退出
   /to <用户> <消息> - 私聊

2. 改进显示：
   - 使用终端控制库（如 termbox-go）
   - 分离输入区和消息区
   - 添加颜色和格式

3. 添加功能：
   - 消息历史记录
   - 表情支持
   - 文件传输

【下一步学习】
理解了 TCP 的并发模型后，可以学习：
- WebSocket：浏览器中的双向通信
- HTTP/2 Server Push：服务器主动推送
- gRPC：基于 HTTP/2 的 RPC 框架
*/
