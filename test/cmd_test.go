package test

import (
	"io/ioutil"
	"testing"

	"github.com/ninggf/xs4go/cmd"
	"github.com/ninggf/xs4go/server"
)

func check(e error, t *testing.T) {
	if e != nil {
		t.Error(e)
	}
}
func Test_setArg(t *testing.T) {
	cmdx := new(cmd.XsCommand)
	cmdx.SetArg(65535)

	if cmdx.GetArg() != 65535 {
		t.Fatal()
	}
}

func Test_UseCmd(t *testing.T) {
	buf, err := ioutil.ReadFile("./a.dat")
	check(err, t)
	cmdx := new(cmd.XsCommand)
	cmdx.Decode(buf, false)
	if cmdx.Cmd != 1 {
		t.Fatalf("cmd expected 1, %d got\n", cmdx.Cmd)
	}
	if cmdx.Buf != "demox" {
		t.Fatalf("buf expected %s, %s got", "demox", cmdx.Buf)
	}
}

func Test_setProject(t *testing.T) {
	conn, err := server.NewConnection("127.0.0.1:8383")
	check(err, t)

	useCmd := cmd.UseProjectCmd("demox")

	res, err := conn.ExecOK(useCmd, cmd.XS_CMD_OK_PROJECT)

	check(err, t)

	if res.GetArg() != cmd.XS_CMD_OK_PROJECT {
		t.Fatalf("response arg expected 201, %d given", res.GetArg())
	}

	conn.Close()
}
