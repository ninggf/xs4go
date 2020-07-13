package server

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/ninggf/xs4go/cmd"
)

// Connection to indexer or searcher server
type Connection struct {
	conn        *net.TCPConn
	addr        *net.TCPAddr
	IsBigEndian bool
	buffer      *bytes.Buffer
	mux         sync.Mutex
	cmds        chan *cmd.XsCommand
	reader      *bufio.Reader
}

// NewConnection to server
func NewConnection(addr string) (*Connection, error) {
	var connection *Connection
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	connection = &Connection{}
	connection.addr = raddr

	conn, err := net.DialTCP("tcp", nil, connection.addr)

	if err != nil {
		return nil, err
	}

	conn.SetReadBuffer(10240)
	connection.conn = conn
	connection.buffer = bytes.NewBuffer([]byte{})
	connection.reader = bufio.NewReader(connection.conn)
	//connection.cmds = make(chan *cmd.XsCommand, 20)
	//connection.execSync()
	return connection, nil
}

// SetTimeout of this connection
func (connection *Connection) SetTimeout(timeout uint16) error {
	if connection.conn != nil {
		command := cmd.XsCommand{}
		command.Cmd = cmd.XS_CMD_TIMEOUT
		command.SetArg(timeout)
		_, err := connection.ExecOK(&command, cmd.XS_CMD_OK_TIMEOUT_SET)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("do not connect to server yet, please connect to server first")
}

// Close the connection
func (connection *Connection) Close() {
	if connection.conn != nil {
		connection.conn.Close()
		connection.conn = nil
		connection.reader = nil
	}
}

// Exec XsCommand on server and get then response
func (connection *Connection) Exec(command *cmd.XsCommand, resArg uint16, resCmd uint8) (*cmd.XsCommand, error) {
	connection.mux.Lock()
	defer connection.mux.Unlock()
	if connection.conn != nil {
		buf := command.Encode(connection.IsBigEndian)
		if _, err := connection.buffer.Write(buf); err != nil {
			connection.buffer.Reset()
			return nil, err
		}
		if (command.Cmd & 0x80) > 0 { // just cache the cmd for those need not answer
			return new(cmd.XsCommand), nil
		}
		data := connection.buffer.Bytes()
		_, err := connection.conn.Write(data)
		connection.buffer.Reset()
		if err != nil {
			return nil, err
		}
		response, err := connection.getResponse()
		if err != nil {
			return nil, err
		}
		if response.Cmd == cmd.XS_CMD_ERR && resCmd != cmd.XS_CMD_ERR {
			return nil, errors.New(response.Buf)
		}
		if response.Cmd != resCmd || (resArg != cmd.XS_CMD_NONE && resArg != response.GetArg()) {
			return nil, fmt.Errorf("unexpected respond: %v", response)
		}
		return response, nil
	}
	return nil, errors.New("do not connect to server yet, please connect to server first")
}

// ExecOK needs response a XS_CMD_OK cmd
func (connection *Connection) ExecOK(command *cmd.XsCommand, resArg uint16) (*cmd.XsCommand, error) {
	return connection.Exec(command, resArg, cmd.XS_CMD_OK)
}

// AsyncExec execute a command in async mode
func (connection *Connection) AsyncExec(command *cmd.XsCommand) {
	connection.cmds <- command
}

// Send XsCommand to server
func (connection *Connection) Send(command *cmd.XsCommand) error {
	conn := connection.conn
	if conn != nil {
		buf := command.Encode(connection.IsBigEndian)
		_, err := conn.Write(buf)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("do not connect to server yet, please connect to server first")
}

// GetSearchResponse return the search response
func (connection *Connection) GetSearchResponse(pcmd *cmd.XsCommand) (*cmd.XsCommand, error) {
	if pcmd.GetArg() != cmd.XS_CMD_OK_RESULT_BEGIN {
		return nil, fmt.Errorf("previous command is not XS_CMD_SEARCH_GET_RESULT")
	}
	return connection.getResponse()
}

func (connection *Connection) getResponse() (*cmd.XsCommand, error) {
	reader := connection.reader
	buffer := bytes.NewBuffer([]byte{})
	head := make([]byte, 8)
	len, err := reader.Read(head)
	if len != 8 {
		return nil, fmt.Errorf("no more data")
	}
	buffer.Write(head[:])
	len1, len2, err := cmd.DecodeHead(head)
	if err != nil {
		return nil, err
	}
	if len1 > 0 {
		buf := make([]byte, len1)
		len, err := reader.Read(buf)
		if err != nil || uint32(len) != len1 {
			return nil, fmt.Errorf("read data error")
		}
		buffer.Write(buf[:])
	}
	if len2 > 0 {
		buf := make([]byte, len2)
		len, err := reader.Read(buf)
		if err != nil || uint8(len) != len2 {
			return nil, fmt.Errorf("read data error")
		}
		buffer.Write(buf[:])
	}
	// 解析返回
	resp := buffer.Bytes()
	rcmd := new(cmd.XsCommand)
	err = rcmd.Decode(resp, connection.IsBigEndian)
	if err != nil {
		return nil, err
	}

	return rcmd, nil
}

// ExecSync run in concurrency
func (connection *Connection) execSync() {
	go func() {
		for cmdx := range connection.cmds {
			connection.ExecOK(cmdx, 0)
		}
	}()
}
