package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Server 数据结构
type Server struct {
	Ip        string
	Port      int
	OnlineMap map[string]*User
	mapLock   sync.RWMutex
	Message   chan string
}

// 接口1 创建Server
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 接口2 启动服务
func (this *Server) Start() {
	// 监听socket
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}

	// defer一个关闭，在函数结束return之前会关闭listener
	defer listener.Close()

	// 启动监听广播信息的goroutine
	go this.ListenMessager()

	// 循环监听
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err:", err)
			continue
		}

		// goroutine 开辟协程
		go this.Handler(conn)
	}
}

// 接口3 处理业务
func (this *Server) Handler(conn net.Conn) {
	// 将用户加入OnlineMap中
	user := NewUser(conn, this)
	user.Online()
	// 监听用户活跃的channel
	isLive := make(chan bool)

	// 接受客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err", err)
				return
			}
			// 提取用户消息
			msg := string(buf[:n-1])

			user.DoMessage(msg)

			// 任意消息代表用户是活跃的
			isLive <- true
		}
	}()

	// 当前阻塞s
	for {
		select {
		case <-isLive:
			// 重置定时器

		case <-time.After(time.Second * 60):
			// 将当前客户端关闭
			user.SendMsg("连接超时")
			close(user.C)
			conn.Close()
			close(isLive)
			return
		}
	}
}

// 接口4 发送广播某用户消息
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg
}

// 接口5 监听广播信息
func (this *Server) ListenMessager() {
	for {
		msg := <-this.Message
		// 发给全部在线User
		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}
