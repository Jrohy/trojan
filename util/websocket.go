package util

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// http升级websocket协议的配置
var wsUpgrader = websocket.Upgrader{
	// 允许所有CORS跨域请求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WsMessage websocket消息
type WsMessage struct {
	MessageType int
	Data        []byte
}

// WsConnection 封装websocket连接
type WsConnection struct {
	wsSocket *websocket.Conn // 底层websocket
	inChan   chan *WsMessage // 读取队列
	outChan  chan *WsMessage // 发送队列

	mutex     sync.Mutex // 避免重复关闭管道
	isClosed  bool
	CloseChan chan byte // 关闭通知
}

// 读取协程
func (wsConn *WsConnection) wsReadLoop() {
	var (
		msgType int
		data    []byte
		msg     *WsMessage
		err     error
	)
	for {
		// 读一个message
		if msgType, data, err = wsConn.wsSocket.ReadMessage(); err != nil {
			fmt.Println("Read error: " + err.Error())
			goto CLOSED
		}
		msg = &WsMessage{
			msgType,
			data,
		}
		// 放入请求队列
		select {
		case wsConn.inChan <- msg:
			if string(data) == "exit" {
				goto CLOSED
			}
		case <-wsConn.CloseChan:
			goto CLOSED
		}
	}
CLOSED:
	wsConn.WsClose()
}

// 发送协程
func (wsConn *WsConnection) wsWriteLoop() {
	var (
		msg *WsMessage
		err error
	)
	for {
		select {
		// 取一个应答
		case msg = <-wsConn.outChan:
			// 写给websocket
			if err = wsConn.wsSocket.WriteMessage(msg.MessageType, msg.Data); err != nil {
				fmt.Println(err)
				goto CLOSED
			}
		case <-wsConn.CloseChan:
			goto CLOSED
		}
	}
CLOSED:
	wsConn.WsClose()
}

// InitWebsocket 初始化ws
func InitWebsocket(resp http.ResponseWriter, req *http.Request) (wsConn *WsConnection, err error) {
	var (
		wsSocket *websocket.Conn
	)
	// 应答客户端告知升级连接为websocket
	if wsSocket, err = wsUpgrader.Upgrade(resp, req, nil); err != nil {
		return
	}
	wsConn = &WsConnection{
		wsSocket:  wsSocket,
		inChan:    make(chan *WsMessage, 1000),
		outChan:   make(chan *WsMessage, 1000),
		CloseChan: make(chan byte),
		isClosed:  false,
	}

	// 读协程
	go wsConn.wsReadLoop()
	// 写协程
	go wsConn.wsWriteLoop()

	return
}

// WsWrite 发送消息
func (wsConn *WsConnection) WsWrite(messageType int, data []byte) (err error) {
	select {
	case wsConn.outChan <- &WsMessage{messageType, data}:
	case <-wsConn.CloseChan:
		err = errors.New("websocket closed")
	}
	return
}

// WsRead 读取消息
func (wsConn *WsConnection) WsRead() (msg *WsMessage, err error) {
	select {
	case msg = <-wsConn.inChan:
		return
	case <-wsConn.CloseChan:
		err = errors.New("websocket closed")
	}
	return
}

// WsClose 关闭连接
func (wsConn *WsConnection) WsClose() {
	wsConn.wsSocket.Close()
	wsConn.mutex.Lock()
	defer wsConn.mutex.Unlock()
	if !wsConn.isClosed {
		wsConn.isClosed = true
		close(wsConn.CloseChan)
	}
}
