package xs

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ninggf/xs4go/cmd"
	"github.com/ninggf/xs4go/schema"
	"github.com/ninggf/xs4go/server"
	"github.com/ninggf/xs4go/tokenizer"
)

// Indexer indicates a index server
type Indexer struct {
	conn       *server.Connection
	setting    *schema.Setting
	schema     *schema.Schema
	cfg        *schema.Config
	buffer     *bytes.Buffer
	tokenizer  tokenizer.Tokenizer
	bufferSize uint32
	rebuilding bool
}

// NewIndexer creates a Indexer
func NewIndexer(conf string) (*Indexer, error) {
	indexer := new(Indexer)
	setting, err := schema.LoadConf(conf)
	if err != nil {
		return nil, err
	}
	indexer.setting = setting
	indexer.cfg = setting.Conf
	indexer.schema = setting.Schema

	conn, err := server.NewConnection(setting.Conf.IndexServer)
	if err != nil {
		return nil, err
	}
	indexer.conn = conn
	indexer.conn.SetTimeout(0)

	return indexer.setProject(setting.Conf.Name)
}

// Schema of current indexer hold
func (indexer *Indexer) Schema() *schema.Schema {
	return indexer.schema
}

// SetTokenizer sets your tokenizer
func (indexer *Indexer) SetTokenizer(tokenizer tokenizer.Tokenizer) {
	if tokenizer != nil {
		indexer.tokenizer = tokenizer
	}
}

// SetDB to set custom database
func (indexer *Indexer) SetDB(db string) error {
	cmdx := cmd.XsCommand{}
	cmdx.Cmd = cmd.XS_CMD_INDEX_SET_DB
	cmdx.Buf = db

	_, err := indexer.conn.ExecOK(&cmdx, cmd.XS_CMD_OK_DB_CHANGED)
	return err
}

// FlushLogging 强制刷新服务端当前项目的搜索日志
func (indexer *Indexer) FlushLogging() error {
	cmdx := cmd.XsCommand{}
	cmdx.Cmd = cmd.XS_CMD_FLUSH_LOGGING
	_, err := indexer.conn.ExecOK(&cmdx, cmd.XS_CMD_OK_LOG_FLUSHED)
	return err
}

// FlushIndex 强制刷新服务端当前项目的搜索日志
func (indexer *Indexer) FlushIndex() error {
	cmdx := cmd.XsCommand{}
	cmdx.Cmd = cmd.XS_CMD_INDEX_COMMIT
	_, err := indexer.conn.ExecOK(&cmdx, cmd.XS_CMD_OK_DB_COMMITED)
	return err
}

// Add document to index server
func (indexer *Indexer) Add(doc map[string]string) error {
	return indexer.update(doc, true)
}

// Update document by id on index server
func (indexer *Indexer) Update(doc map[string]string) error {
	return indexer.update(doc, false)
}

// Del deletes Document from index server
func (indexer *Indexer) Del(terms ...string) error {
	return indexer.DelByField("", terms...)
}

// DelByField deletes by field from index server
func (indexer *Indexer) DelByField(field string, terms ...string) error {
	if field == "" {
		field = indexer.schema.StrId
	}
	idField, ok := indexer.schema.FieldMetas[field]

	if !ok {
		return fmt.Errorf("field '%s' is not defined", field)
	}

	ln := len(terms)
	var cmdx *cmd.XsCommand
	if ln == 1 {
		cmdx = cmd.NewCommand2(cmd.XS_CMD_INDEX_REMOVE, 0, idField.Vno, strings.ToLower(terms[0]), "")
	} else {
		buf := bytes.NewBuffer([]byte{})
		for _, te := range terms {
			cmds := cmd.NewCommand2(cmd.XS_CMD_INDEX_REMOVE, 0, idField.Vno, strings.ToLower(te), "")
			buf.Write(cmds.Encode(false)[:])
		}
		bufStr := string(buf.Bytes())
		cmdx = cmd.NewCommand(cmd.XS_CMD_INDEX_EXDATA, 0, bufStr, "")
	}
	_, err := indexer.conn.ExecOK(cmdx, cmd.XS_CMD_OK_RQST_FINISHED)
	return err
}

// AddSynonym adds 添加同义词
func (indexer *Indexer) AddSynonym(word string, synonyms ...string) error {
	if word == "" || len(synonyms) == 0 {
		return nil
	}
	for _, synonym := range synonyms {
		if synonym == "" {
			continue
		}
		cmdx := cmd.NewCommand2(cmd.XS_CMD_INDEX_SYNONYMS, cmd.XS_CMD_INDEX_SYNONYMS_ADD, 0, word, synonym)
		if err := indexer.bufferExec(cmdx, cmd.XS_CMD_OK_RQST_FINISHED); err != nil {
			return err
		}
	}
	return nil
}

// DelSynonym delete synonym on index server
func (indexer *Indexer) DelSynonym(word string, synonyms ...string) error {
	if word == "" {
		return nil
	}
	if len(synonyms) == 0 {
		cmdx := cmd.NewCommand2(cmd.XS_CMD_INDEX_SYNONYMS, cmd.XS_CMD_INDEX_SYNONYMS_DEL, 0, word, "")
		return indexer.bufferExec(cmdx, cmd.XS_CMD_OK_RQST_FINISHED)
	}
	for _, synonym := range synonyms {
		if synonym == "" {
			continue
		}
		cmdx := cmd.NewCommand2(cmd.XS_CMD_INDEX_SYNONYMS, cmd.XS_CMD_INDEX_SYNONYMS_DEL, 0, word, synonym)
		if err := indexer.bufferExec(cmdx, cmd.XS_CMD_OK_RQST_FINISHED); err != nil {
			return err
		}
	}
	return nil
}

// OpenBuffer open buffer for improving performance
func (indexer *Indexer) OpenBuffer(size uint32) error {
	if size > 32 {
		size = 32
	}
	if indexer.buffer != nil {
		if err := indexer.flushBuffer(); err != nil {
			indexer.buffer = nil
			indexer.bufferSize = 0
			return err
		}

		if indexer.bufferSize == (size << 20) {
			return nil
		}
	}

	if size > 0 {
		indexer.bufferSize = 30 //(size << 20)
		indexer.buffer = bytes.NewBuffer(make([]byte, 0, indexer.bufferSize))
	} else {
		indexer.bufferSize = 0
		indexer.buffer = nil
	}
	return nil
}

// Submit submits buffer to index server and checks the return
func (indexer *Indexer) Submit() error {
	return indexer.OpenBuffer(0)
}

// Clean index database. 如果当前数据库处于重建过程中将禁止清空
func (indexer *Indexer) Clean() error {
	cmdx := cmd.NewCommand(cmd.XS_CMD_INDEX_CLEAN_DB, 0, "", "")
	_, err := indexer.conn.ExecOK(cmdx, cmd.XS_CMD_OK_DB_CLEAN)
	return err
}

// BeginRebuild 开始重建索引
// 此后所有的索引更新指令将写到临时库, 而不是当前搜索库,
// 重建完成后调用 {EnRebuild} 实现平滑重建索引, 重建过程仍可搜索旧的索引库,
// 如直接用 Clean 清空数据, 则会导致重建过程搜索到不全的数据
func (indexer *Indexer) BeginRebuild() error {
	cmdx := cmd.XsCommand{cmd.XS_CMD_INDEX_REBUILD, 0, 0, "", ""}
	if _, err := indexer.conn.ExecOK(&cmdx, cmd.XS_CMD_OK_DB_REBUILD); err != nil {
		return err
	}
	indexer.rebuilding = true
	return nil
}

// EndRebuild 完成并关闭重建索引
// 重建完成后调用, 用重建好的索引数据代替旧的索引数据
func (indexer *Indexer) EndRebuild() error {
	cmdx := cmd.XsCommand{cmd.XS_CMD_INDEX_REBUILD, 1, 0, "", ""}
	_, err := indexer.conn.ExecOK(&cmdx, cmd.XS_CMD_OK_DB_REBUILD)
	return err
}

// StopRebuild 中止索引重建
// 丢弃重建临时库的所有数据, 恢复成当前搜索库, 主要用于偶尔重建意外中止的情况
func (indexer *Indexer) StopRebuild() error {
	cmdx := cmd.XsCommand{cmd.XS_CMD_INDEX_REBUILD, 2, 0, "", ""}
	_, err := indexer.conn.ExecOK(&cmdx, cmd.XS_CMD_OK_DB_REBUILD)
	return err
}

// Close connection
func (indexer *Indexer) Close() {
	if indexer.conn != nil {
		if indexer.buffer != nil {
			indexer.flushBuffer()
		}
		indexer.conn.Close()
		indexer.conn = nil
	}
}

func (indexer *Indexer) setProject(project string) (*Indexer, error) {
	cmdx := cmd.UseProjectCmd(project)

	_, err := indexer.conn.ExecOK(cmdx, cmd.XS_CMD_OK_PROJECT)

	if err != nil {
		indexer.conn.Close()
		indexer.conn = nil
		indexer.cfg = nil
		return nil, err
	}
	var tokenizer tokenizer.Tokenizer = tokenizer.DefaultTokenizer{"default"}
	indexer.tokenizer = tokenizer
	return indexer, nil
}

func (indexer *Indexer) update(doc map[string]string, add bool) error {
	idField := indexer.setting.Schema.Id // id
	key, ok := doc[idField.Name]
	// check primary key of document
	if !ok || key == "" {
		return fmt.Errorf("Missing value of primary key (FIELD:%s)", idField.Name)
	}

	// request cmd
	cmdx := &cmd.XsCommand{}
	cmdx.Cmd = cmd.XS_CMD_INDEX_REQUEST
	if add {
		cmdx.Arg1 = cmd.XS_CMD_INDEX_REQUEST_ADD
	} else {
		cmdx.Arg1 = cmd.XS_CMD_INDEX_REQUEST_UPDATE
		cmdx.Arg2 = idField.Vno
		cmdx.Buf = key
	}

	cmds := make(map[int]*cmd.XsCommand)
	cmds[len(cmds)] = cmdx
	indexer.buildCmd(idField.Name, idField, doc, cmds)

	for f, v := range indexer.schema.FieldMetas {
		if v.Type == "id" {
			continue
		}
		indexer.buildCmd(f, v, doc, cmds)
	}
	for i := 0; i < len(cmds); i++ {
		icmd := cmds[i]
		_, err := indexer.conn.ExecOK(icmd, cmd.XS_CMD_NONE)
		if err != nil {
			return err
		}
	}
	// todo: submit cmd
	submitCmd := &cmd.XsCommand{}
	submitCmd.Cmd = cmd.XS_CMD_INDEX_SUBMIT
	_, err := indexer.conn.ExecOK(submitCmd, cmd.XS_CMD_OK_RQST_FINISHED)
	return err
}

func (indexer *Indexer) buildCmd(f string, v *schema.FieldMeta, doc map[string]string, cmds map[int]*cmd.XsCommand) {
	value, ok := doc[f]
	// 索引操作
	if ok && value != "" { //找到对应的值
		varg := uint8(0)
		if v.IsNumeric() {
			varg = cmd.XS_CMD_VALUE_FLAG_NUMERIC
		}

		if v.HasIndex() { //启用索引
			terms := indexer.tokenizer.GetTokens(value)
			if len(terms) > 0 && v.HasIndexSelf() {
				wdf := uint8(1)
				if !v.IsBoolIndex() {
					wdf = uint8(v.Weight | cmd.XS_CMD_INDEX_FLAG_CHECKSTEM)
				}
				for _, term := range terms {
					if len(term) > 200 {
						continue
					}
					term = strings.ToLower(term)
					cmds[len(cmds)] = cmd.NewCommand2(cmd.XS_CMD_DOC_TERM, wdf, v.Vno, term, "")
				}
			}

			if len(terms) > 0 && v.HasIndexMixed() {
				mtext := strings.Join(terms, " ")
				cmds[len(cmds)] = cmd.NewCommand2(cmd.XS_CMD_DOC_INDEX, uint8(v.Weight), schema.MIXED_VNO, mtext, "")
			}
		}
		// add value
		cmds[len(cmds)] = cmd.NewCommand2(cmd.XS_CMD_DOC_VALUE, varg, v.Vno, value, "")
	}
	// TODO: process add terms
	// todo: process add text
}

func (indexer *Indexer) bufferExec(cmdx *cmd.XsCommand, resArg uint16) error {
	if indexer.buffer != nil {
		ln := indexer.buffer.Len()
		buf := cmdx.Encode(false)
		ln1 := len(buf)
		if uint32(ln1+ln) > indexer.bufferSize {
			err := indexer.flushBuffer()
			if err != nil {
				return err
			}
		}
		indexer.buffer.Write(buf[:])
		return nil
	}
	_, err := indexer.conn.ExecOK(cmdx, resArg)
	return err
}

func (indexer *Indexer) flushBuffer() error {
	if indexer.buffer != nil {
		buf := string(indexer.buffer.Bytes())
		if buf != "" {
			cmdx := cmd.NewCommand(cmd.XS_CMD_INDEX_EXDATA, 0, buf, "")
			_, err := indexer.conn.ExecOK(cmdx, cmd.XS_CMD_OK_RQST_FINISHED)
			indexer.buffer.Reset()
			return err
		}
	}

	return nil
}
