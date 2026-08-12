package executor

import (
	"fmt"
	"time"

	"github.com/gorcon/rcon"
)

// Executor 封装 RCON 连接
type Executor struct {
	conn *rcon.Conn
}

// NewExecutor 建立 RCON 连接
func NewExecutor(address, password string, timeout int, skipErrors bool) (*Executor, error) {
	conn, err := rcon.Dial(address, password, rcon.SetDialTimeout(time.Duration(timeout)*time.Second), rcon.SetDeadline(time.Duration(timeout)*time.Second))
	if err != nil {
		return nil, err
	}
	return &Executor{conn: conn}, nil
}

// Execute 执行命令并返回响应
func (e *Executor) Execute(command string) (string, error) {
	response, err := e.conn.Execute(command)
	if err != nil {
		return "", err
	}
	if response == "" {
		return "", fmt.Errorf("empty response")
	}
	return response, nil
}

// Close 关闭连接
func (e *Executor) Close() {
	if e.conn != nil {
		_ = e.conn.Close()
	}
}
