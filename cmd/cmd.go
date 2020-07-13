package cmd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// XsCommand can be sent to xunsearch indexer/searcher server
type XsCommand struct {
	Cmd  uint8
	Arg1 uint8  //其值为 0~255, 具体含义视不同 CMD 而确定
	Arg2 uint8  //其值为 0~255, 具体含义视不同 CMD 而确定, 常用于存储 value no
	Buf  string //字符串内容, 最大长度为 2GB
	Buf1 string //字符串内容1, 最大长度为 255字节
}

// NewCommand creates a XsCommand instance
func NewCommand(cmd uint8, arg uint16, buf ...string) *XsCommand {
	command := new(XsCommand)
	command.Cmd = cmd
	command.SetArg(arg)
	if len(buf) >= 1 {
		command.Buf = buf[0]
		if len(buf) >= 2 {
			command.Buf1 = buf[1]
		}
	}
	return command
}

// NewCommand2
func NewCommand2(cmd uint8, arg1, arg2 uint8, buf ...string) *XsCommand {
	command := new(XsCommand)
	command.Cmd = cmd
	command.Arg1 = arg1
	command.Arg2 = arg2
	if len(buf) >= 1 {
		command.Buf = buf[0]
		if len(buf) >= 2 {
			command.Buf1 = buf[1]
		}
	}

	return command
}

// GetArg return the the
func (xsCommand *XsCommand) GetArg() uint16 {
	var arg uint16
	arg = uint16(xsCommand.Arg1)
	arg = (arg << 8) | uint16(xsCommand.Arg2)
	return arg
}

// SetArg sets the arg1 and arg2 by on value
func (xsCommand *XsCommand) SetArg(arg uint16) {
	xsCommand.Arg1 = uint8(arg >> 8)
	xsCommand.Arg2 = uint8(arg & 0xff)
}

// String of this command
func (xsCommand *XsCommand) String() string {
	return fmt.Sprintf("cmd:%d, arg1:%d, arg2:%d, buf:%s, buf1:%s", xsCommand.Cmd, xsCommand.Arg1, xsCommand.Arg2, xsCommand.Buf, xsCommand.Buf1)
}

// Encode this command to byte array
// param: bigEndian bool
func (xsCommand *XsCommand) Encode(bigEndian bool) []byte {
	lenOfbuf1 := len(xsCommand.Buf1)
	if lenOfbuf1 > 0xff {
		xsCommand.Buf1 = xsCommand.Buf1[0:0xff]
		lenOfbuf1 = 0xff
	}
	lenOfbuf := len(xsCommand.Buf)

	buf := make([]byte, 8+lenOfbuf+lenOfbuf1)
	buf[0] = xsCommand.Cmd    //1
	buf[1] = xsCommand.Arg1   //1
	buf[2] = xsCommand.Arg2   //1
	buf[3] = uint8(lenOfbuf1) //1

	bint := uint32(lenOfbuf)
	bytesBuff := bytes.NewBuffer([]byte{})

	if bigEndian {
		binary.Write(bytesBuff, binary.BigEndian, bint)
	} else {
		binary.Write(bytesBuff, binary.LittleEndian, bint)
	}
	copy(buf[4:], bytesBuff.Bytes()) //4
	idx := 8
	if lenOfbuf > 0 {
		copy(buf[idx:], []byte(xsCommand.Buf))
		idx += lenOfbuf
	}
	if lenOfbuf1 > 0 {
		copy(buf[idx:], []byte(xsCommand.Buf1))
	}
	return buf
}

// Decode byte buffer into XsCommand
func (xsCommand *XsCommand) Decode(buf []byte, bigEndian bool) error {
	bufLen := len(buf)
	if buf == nil || bufLen < 8 {
		return errors.New("invalid response data")
	}
	xsCommand.Cmd = buf[0]
	xsCommand.Arg1 = buf[1]
	xsCommand.Arg2 = buf[2]
	lenOfbuf1 := int(buf[3])
	bytesBuffer := bytes.NewBuffer(buf[4:8])
	var lenOfbuf uint32
	if bigEndian {
		binary.Read(bytesBuffer, binary.BigEndian, &lenOfbuf)
	} else {
		binary.Read(bytesBuffer, binary.LittleEndian, &lenOfbuf)
	}

	if lenOfbuf > uint32(bufLen-8) {
		return errors.New("invalid length of buffer")
	}

	idx := uint32(8)
	if lenOfbuf > 0 {
		idx += lenOfbuf
		xsCommand.Buf = string(buf[8:idx])
	}
	if lenOfbuf1 > 0 {
		xsCommand.Buf1 = string(buf[idx:])
	}
	return nil
}

// DecodeHead of response from server
func DecodeHead(buf []byte) (uint32, uint8, error) {
	bufLen := len(buf)
	if buf == nil || bufLen < 8 {
		return 0, 0, errors.New("invalid response data")
	}

	lenOfbuf1 := uint8(buf[3])
	bytesBuffer := bytes.NewBuffer(buf[4:8])
	var lenOfbuf uint32
	binary.Read(bytesBuffer, binary.LittleEndian, &lenOfbuf)

	return lenOfbuf, lenOfbuf1, nil
}

// Pack date into String
func Pack(format string, args ...interface{}) (string, error) {
	if len(format) != len(args) {
		return "", fmt.Errorf("format length %d != args length %d", len(format), len(args))
	}
	buf := bytes.NewBuffer([]byte{})
	for i := 0; i < len(format); i++ {
		t := fmt.Sprintf("%T", args[i])
		if m, _ := regexp.Match("^u?int(8|16|32|64)?$", []byte(t)); !m {
			return "", fmt.Errorf("%v type is %T, it's an invalid type for Pakc", args[i], args[i])
		}
		switch format[i] {
		case 'I':
			binary.Write(buf, binary.LittleEndian, args[i].(uint32))
			break
		case 'C':
			buf.WriteByte(args[i].(uint8))
			break
		case 'n':
			binary.Write(buf, binary.BigEndian, args[i].(uint16))
			break
		default:
		}
	}
	return string(buf.Bytes()), nil
}

// UnPack data by format
func UnPack(format, data string) (map[string]interface{}, error) {
	chks := strings.Split(format, "/")
	unpacked := make(map[string]interface{})
	buf := bytes.NewBufferString(data)

	for i, ch := range chks {
		na := strconv.Itoa(i)
		if len(ch) > 1 {
			na = ch[1:]
		}
		switch ch[0] {
		case 'I':
			var i uint32
			if err := binary.Read(buf, binary.LittleEndian, &i); err != nil {
				return unpacked, err
			}
			unpacked[na] = i
			break
		case 'C':
			var i uint8
			if err := binary.Read(buf, binary.LittleEndian, &i); err != nil {
				return unpacked, err
			}
			unpacked[na] = i
			break
		case 'i':
			var i int32
			if err := binary.Read(buf, binary.LittleEndian, &i); err != nil {
				return unpacked, err
			}
			unpacked[na] = i
			break
		case 'f':
			var i float32
			if err := binary.Read(buf, binary.LittleEndian, &i); err != nil {
				return unpacked, err
			}
			unpacked[na] = i
			break
		default:
			unpacked[na] = nil
		}
	}
	return unpacked, nil
}

// ReplaceAllStringSubmatchFunc like php preg_replace_callback
func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]string) string) string {
	result := ""
	lastIndex := 0
	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			groups = append(groups, str[v[i]:v[i+1]])
		}
		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}
	return result + str[lastIndex:]
}

func MaxLimit(limits ...uint8) uint8 {
	limit := uint8(10)
	if len(limits) > 0 {
		if limits[0] > 20 {
			limit = 20
		} else {
			limit = limits[0]
		}
	}
	return limit
}
