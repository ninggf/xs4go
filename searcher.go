package xs

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/ninggf/xs4go/cmd"
	"github.com/ninggf/xs4go/schema"
	"github.com/ninggf/xs4go/server"
	"github.com/ninggf/xs4go/tokenizer"
)

const (
	logDB string = "log_db"
)

// Facet for document filed
type Facet map[string]int32

// Searcher indicates a index server
type Searcher struct {
	Facets      map[string]Facet
	conn        *server.Connection
	setting     *schema.Setting
	schema      *schema.Schema
	cfg         *schema.Config
	buffer      *bytes.Buffer
	tokenizer   tokenizer.Tokenizer
	bufferSize  uint32
	defaultOp   uint8
	queryPrefix map[string]bool
	lastCount   uint32
	count       uint32
	limit       uint32
	offset      uint32
	curDB       string
	curDBs      map[string]bool
	lastDB      string
	lastHlQuery string
	query       string
	terms       []string
}

// NewSearcher creates a searcher that connect to search server
func NewSearcher(conf string) (*Searcher, error) {
	searcher := new(Searcher)
	setting, err := schema.LoadConf(conf)
	if err != nil {
		return nil, err
	}
	searcher.setting = setting
	searcher.cfg = setting.Conf
	searcher.schema = setting.Schema
	searcher.count = math.MaxUint32
	conn, err := server.NewConnection(setting.Conf.SearchServer)
	if err != nil {
		return nil, err
	}
	searcher.conn = conn
	searcher.conn.SetTimeout(0)

	return searcher.setProject(setting.Conf.Name)
}

// Fuzzy mode
func (searcher *Searcher) Fuzzy(fuzzy bool) {
	if fuzzy {
		searcher.defaultOp = cmd.XS_CMD_QUERY_OP_OR
	} else {
		searcher.defaultOp = cmd.XS_CMD_QUERY_OP_AND
	}
}

// SetCutOff 设置百分比/权重剔除参数
// 通常是在开启 Fuzzy 或使用 OR 连接搜索语句时才需要设置此项
func (searcher *Searcher) SetCutOff(percent uint8, weight float32) *Searcher {
	if percent > 100 {
		percent = 100
	}
	if weight < 0.1 {
		weight = 0.1
	} else if weight > 25.5 {
		weight = 25.5
	}
	rWeight := uint8(weight * 10)
	cmdx := cmd.XsCommand{cmd.XS_CMD_SEARCH_SET_CUTOFF, percent, rWeight, "", ""}
	searcher.conn.ExecOK(&cmdx, 0)
	return searcher
}

// SetRequireMatchedTerm 设置在搜索结果文档中返回匹配词表
// 请在 {@link search} 前调用本方法, 然后使用 {@link XSDocument::matched} 获取
func (searcher *Searcher) SetRequireMatchedTerm(required bool) *Searcher {
	cmdx := cmd.XsCommand{}
	cmdx.Cmd = cmd.XS_CMD_SEARCH_SET_MISC
	cmdx.Arg1 = cmd.XS_CMD_SEARCH_MISC_MATCHED_TERM
	if required {
		cmdx.Arg2 = 1
	}

	searcher.conn.ExecOK(&cmdx, 0)
	return searcher
}

// SetWeightingScheme 设置检索匹配的权重方案
// 目前支持三种权重方案: 0=BM25/1=Bool/2=Trad
func (searcher *Searcher) SetWeightingScheme(policy uint8) *Searcher {
	switch policy {
	case 0:
	case 1:
	case 2:
		cmdx := cmd.XsCommand{}
		cmdx.Cmd = cmd.XS_CMD_SEARCH_SET_MISC
		cmdx.Arg1 = cmd.XS_CMD_SEARCH_MISC_WEIGHT_SCHEME
		cmdx.Arg2 = policy
		searcher.conn.ExecOK(&cmdx, 0)
		break
	default:
		break
	}
	return searcher
}

// SetAutoSynonyms 开启自动同义词搜索功能
func (searcher *Searcher) SetAutoSynonyms(auto bool) *Searcher {
	flag := cmd.XS_CMD_PARSE_FLAG_BOOLEAN | cmd.XS_CMD_PARSE_FLAG_PHRASE | cmd.XS_CMD_PARSE_FLAG_LOVEHATE
	if auto {
		flag |= cmd.XS_CMD_PARSE_FLAG_AUTO_MULTIWORD_SYNONYMS
	}
	cmdx := cmd.NewCommand(cmd.XS_CMD_QUERY_PARSEFLAG, uint16(flag))
	searcher.conn.ExecOK(cmdx, 0)
	return searcher
}

// SetSynonymScale 设置同义词搜索的权重比例 取值范围 0.01-2.55, 1 表示不调整
func (searcher *Searcher) SetSynonymScale(scale float32) *Searcher {
	if scale < 0.01 {
		scale = 0.01
	} else if scale > 2.55 {
		scale = 2.55
	}
	rWeight := uint8(scale * 10)
	cmdx := cmd.XsCommand{cmd.XS_CMD_SEARCH_SET_CUTOFF, cmd.XS_CMD_SEARCH_MISC_SYN_SCALE, rWeight, "", ""}
	searcher.conn.ExecOK(&cmdx, 0)
	return searcher
}

// GetAllSynonyms 获取当前库内的全部同义词列表
func (searcher *Searcher) GetAllSynonyms(limit, offset uint32, stemmed bool) map[string][]string {
	cmdx := cmd.XsCommand{}
	cmdx.Cmd = cmd.XS_CMD_SEARCH_GET_SYNONYMS
	if limit > 0 {
		if page, err := cmd.Pack("II", offset, limit); err != nil {
			cmdx.Buf1 = page
		}
	}
	if stemmed {
		cmdx.Arg1 = 1
	}
	synonyms := make(map[string][]string)
	res, err := searcher.conn.ExecOK(&cmdx, cmd.XS_CMD_OK_RESULT_SYNONYMS)

	if err == nil && res.Buf != "" {
		bufs := strings.Split(res.Buf, "\n")
		for _, sydef := range bufs {
			sys := strings.Split(sydef, "\t")
			synonyms[sys[0]] = sys[1:]
		}
	}

	return synonyms
}

// GetSynonyms 获取指定词汇的同义词列表
func (searcher *Searcher) GetSynonyms(word string) []string {
	if word != "" {
		cmdx := cmd.NewCommand2(cmd.XS_CMD_SEARCH_GET_SYNONYMS, 2, 0, word)
		res, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_RESULT_SYNONYMS)
		if err == nil && res.Buf != "" {
			return strings.Split(res.Buf, "\n")
		}
	}

	return []string{}
}

// Count 估算搜索语句的匹配数据量
//
// 如果搜索语句和最近一次
//	searcher.Search 的语句一样, 请改用 {@link GetLastCount} 以提升效率
// 最大长度为 80 字节
func (searcher *Searcher) Count(query string) uint32 {
	if query != "" {
		query = searcher.preQueryString(query)
	}
	if query == "" && searcher.count != math.MaxUint32 {
		return searcher.count
	}
	cmdx := cmd.NewCommand2(cmd.XS_CMD_SEARCH_GET_TOTAL, 0, searcher.defaultOp, query)
	if res, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_SEARCH_TOTAL); err == nil {
		if rtn, err1 := cmd.UnPack("Icnt", res.Buf); err1 == nil {
			cnt := rtn["cnt"].(uint32)
			if query == "" {
				searcher.count = cnt
			}
			return cnt
		}
	}
	return 0
}

// GetLastCount 获取最近那次搜索的匹配总数估值
func (searcher *Searcher) GetLastCount() uint32 {
	return searcher.lastCount
}

// Limit the result set
func (searcher *Searcher) Limit(limit ...uint32) *Searcher {
	if len(limit) > 0 {
		searcher.limit = limit[0]
		searcher.offset = 0
		if len(limit) > 1 {
			searcher.offset = limit[1]
		}
	}
	if searcher.limit == 0 {
		searcher.limit = 10
	}
	return searcher
}

// Search return results
func (searcher *Searcher) Search(queries ...string) ([]*schema.Document, error) {
	query := strings.Join(queries, " AND ")
	if query != "" {
		query = searcher.preQueryString(query)
	}
	if searcher.limit == 0 {
		searcher.limit = 10
	}
	page, _ := cmd.Pack("II", searcher.offset, searcher.limit)
	cmdx := cmd.NewCommand2(cmd.XS_CMD_SEARCH_GET_RESULT, 0, searcher.defaultOp, query, page)
	res, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_RESULT_BEGIN)
	if err != nil {
		return []*schema.Document{}, err
	}
	resx, err1 := cmd.UnPack("Icount", res.Buf)
	if err1 != nil {
		return []*schema.Document{}, err1
	}
	searcher.lastCount = resx["count"].(uint32)
	searcher.limit = 10
	searcher.offset = 0
	var currSchema *schema.Schema
	result := make([]*schema.Document, searcher.lastCount)
	if searcher.curDB == logDB {
		currSchema = searcher.setting.Logger
	} else {
		currSchema = searcher.setting.Schema
	}
	vnomap := currSchema.VnoMap()

	var (
		doc    *schema.Document
		docIdx uint32
		vno    uint8
		vlen   uint32
		num    int32
	)

	for {
		mres, merr := searcher.conn.GetSearchResponse(res)
		if merr != nil {
			return []*schema.Document{}, merr
		}
		if mres.Cmd == cmd.XS_CMD_SEARCH_RESULT_FACETS {
			off := 0
			ln := len(mres.Buf)
			for (off + 6) < ln {
				facts, err2 := cmd.UnPack("Cvno/Cvlen/Inum", mres.Buf[off:off+6])
				if err2 != nil {
					break
				}
				vno = facts["vno"].(uint8)
				vlen = facts["vlen"].(uint32)
				if fname, ok := vnomap[vno]; ok {
					num = facts["num"].(int32)
					value := mres.Buf[off+6 : off+6+int(vlen)]
					facet, ok1 := searcher.Facets[fname]
					if !ok1 {
						facet = Facet{}
					}
					facet[value] = num
				}
				off += int(vlen) + 6
			}
		} else if mres.Cmd == cmd.XS_CMD_SEARCH_RESULT_DOC {
			doc, err = schema.NewDocument(mres.Buf)
			if err != nil {
				break
			}
			result[docIdx] = doc
			docIdx++
		} else if mres.Cmd == cmd.XS_CMD_SEARCH_RESULT_FIELD {
			if doc != nil {
				fname, ok := vnomap[uint8(mres.GetArg())]
				if !ok {
					fname = strconv.Itoa(int(mres.GetArg()))
				}
				doc.Fields[fname] = mres.Buf
			}
		} else if mres.Cmd == cmd.XS_CMD_SEARCH_RESULT_MATCHED {
			if doc != nil {
				doc.Matched = strings.Split(" ", mres.Buf)
			}
		} else if mres.Cmd == cmd.XS_CMD_OK && cmd.XS_CMD_OK_RESULT_END == mres.GetArg() {
			break
		} else {
			return []*schema.Document{}, fmt.Errorf("Unexpected respond in search :%v", mres)
		}
	}
	if query == "" && searcher.curDB != logDB {
		searcher.count = searcher.lastCount
		searcher.logQuery()
	}
	return result, nil
}

// SetQuery 设置默认搜索语句
//
// 用于不带参数的 {@link Count} 或 {@link Search} 以及 {@link Terms} 调用
// 可与 {@link AddWeight} 组合运用
func (searcher *Searcher) SetQuery(query string) *Searcher {
	if query != "" {
		searcher.query = query
		searcher.AddQueryString(query, cmd.XS_CMD_QUERY_OP_AND, 1)
	}
	return searcher
}

// AddQueryString 增加默认搜索语句并返回修正后的搜索语句
//
// addOp可选值:
//	XS_CMD_QUERY_OP_AND, XS_CMD_QUERY_OP_OR,XS_CMD_QUERY_OP_AND_NOT,XS_CMD_QUERY_OP_XOR,XS_CMD_QUERY_OP_AND_MAYBE,XS_CMD_QUERY_OP_FILTER
//
// scale: 权重计算缩放比例, 默认为 1表示不缩放, 其它值范围 0.xx ~ 655.35
func (searcher *Searcher) AddQueryString(query string, addOp uint8, scale float32) string {
	query = searcher.preQueryString(query)
	bscale := ""
	if scale > 0 && scale != 1 && scale < 655.35 {
		if pd, err := cmd.Pack("n", int(scale*100)); err == nil {
			bscale = pd
		}
	}
	cmdx := cmd.NewCommand2(cmd.XS_CMD_QUERY_PARSE, addOp, searcher.defaultOp, query, bscale)
	searcher.conn.ExecOK(cmdx, 0)
	return query
}

// AddQueryTerm 增加默认搜索词汇
// 索引词所属的字段, 若为混合区词汇可设为 "" 或 body 型的字段名
func (searcher *Searcher) AddQueryTerm(field string, addOp uint8, scale float32, terms ...string) {
	bscale := ""
	if scale > 0 && scale != 1 && scale < 655.35 {
		if pd, err := cmd.Pack("n", int(scale*100)); err == nil {
			bscale = pd
		}
	}
	vno := uint8(schema.MIXED_VNO)
	if field != "" {
		f, ok := searcher.schema.FieldMetas[field]
		if ok {
			vno = f.Vno
		}
	}
	ln := len(terms)
	cmd1 := uint8(cmd.XS_CMD_QUERY_TERM)
	if ln == 0 {
		return
	} else if ln > 1 {
		cmd1 = cmd.XS_CMD_QUERY_TERMS
	}
	cmdx := cmd.NewCommand2(cmd1, addOp, vno, strings.Join(terms, "\t"), bscale)
	searcher.conn.ExecOK(cmdx, 0)
}

// GetQuery return a parased string
func (searcher *Searcher) GetQuery(query string) (string, error) {
	if query != "" {
		query = searcher.preQueryString(query)
	}
	cmdx := cmd.NewCommand2(cmd.XS_CMD_QUERY_GET_STRING, 0, searcher.defaultOp, query)
	res, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_QUERY_STRING)
	if err != nil {
		return "", err
	}
	// TODO 处理 VALUE_RANGE VALUE_GE VALUE_LE

	return res.Buf, nil
}

// SetDB to search
func (searcher *Searcher) SetDB(db string) error {
	cmdx := cmd.XsCommand{}
	cmdx.Cmd = cmd.XS_CMD_SEARCH_SET_DB
	cmdx.Buf = db

	_, err := searcher.conn.ExecOK(&cmdx, cmd.XS_CMD_OK_DB_CHANGED)
	if err == nil {
		searcher.lastDB, searcher.curDB = searcher.curDB, db
	}
	return err
}

// AddDB to search
func (searcher *Searcher) AddDB(db string) error {
	cmdx := cmd.XsCommand{}
	cmdx.Cmd = cmd.XS_CMD_SEARCH_ADD_DB
	cmdx.Buf = db

	_, err := searcher.conn.ExecOK(&cmdx, cmd.XS_CMD_OK_DB_CHANGED)

	if err == nil {
		searcher.curDBs[db] = true
	}

	return err
}

// GetDbTotal return the total database
func (searcher *Searcher) GetDbTotal() uint32 {
	cmdx := cmd.NewCommand(cmd.XS_CMD_SEARCH_DB_TOTAL, 0)
	if res, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_DB_TOTAL); err == nil {
		if total, err1 := cmd.UnPack("Itotal", res.Buf); err1 == nil {
			return total["total"].(uint32)
		}
	}
	return 0
}

// Terms 获取搜索语句中的高亮词条列表
func (searcher *Searcher) Terms(query string) []string {
	if query != "" {
		query = searcher.preQueryString(query)
	}

	if query == "" && searcher.terms != nil {
		return searcher.terms
	}

	cmdx := cmd.NewCommand2(cmd.XS_CMD_QUERY_GET_TERMS, 0, searcher.defaultOp, query)
	res, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_QUERY_TERMS)
	if err != nil {
		return []string{}
	}

	terms := strings.Split(res.Buf, " ")
	rterms := []string{}
	for i := 0; i < len(terms); i++ {
		if terms[i] == "" || strings.Index(terms[i], ":") > 0 {
			continue
		}
		rterms = append(rterms, terms[i])
	}
	searcher.terms = rterms
	return rterms
}

// GetCorrectedQuery  获取修正后的搜索词列表
//
// 通常当某次检索结果数量偏少时, 可以用该函数设计 "你是不是要找: ..." 功能
func (searcher *Searcher) GetCorrectedQuery(queries ...string) []string {
	query := strings.Join(queries, " ")
	if query == "" {
		if searcher.count > 0 && float64(searcher.count) > math.Ceil(0.001*float64(searcher.GetDbTotal())) {
			return []string{}
		}
		query = searcher.cleanFieldQuery(searcher.query)
	}
	if query == "" || strings.Index(query, ":") >= 0 {
		return []string{}
	}
	cmdx := cmd.NewCommand(cmd.XS_CMD_QUERY_GET_CORRECTED, 0, query)
	res, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_QUERY_CORRECTED)
	if err != nil {
		return []string{}
	}
	return strings.Split(res.Buf, "\n")
}

// GetExpandedQuery 获取展开的搜索词列表
//
// 需要展开的前缀, 可为拼音、英文、中文,需要返回的搜索词数量上限, 默认为 10, 最大值为 20
func (searcher *Searcher) GetExpandedQuery(query string, limits ...uint8) []string {
	limit := cmd.MaxLimit(limits...)
	cmdx := cmd.NewCommand2(cmd.XS_CMD_QUERY_GET_EXPANDED, limit, 0, query)
	res, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_RESULT_BEGIN)
	result := []string{}
	if err != nil {
		return result
	}
	for {
		mres, merr := searcher.conn.GetSearchResponse(res)
		if merr != nil {
			break
		}
		if mres.Cmd == cmd.XS_CMD_SEARCH_RESULT_FIELD {
			result = append(result, mres.Buf)
		} else if mres.Cmd == cmd.XS_CMD_OK && cmd.XS_CMD_OK_RESULT_END == mres.GetArg() {
			break
		} else {
			break
		}
	}
	return result
}

// GetHotQuery 获取热门搜索词列表
func (searcher *Searcher) GetHotQuery(hotType string, limits ...uint8) map[string]uint32 {
	result := make(map[string]uint32)
	limit := cmd.MaxLimit(limits...)
	if hotType == "" || (hotType != "lastnum" && hotType != "currnum") {
		hotType = "total"
	}
	if searcher.SetDB(logDB) != nil {
		return result
	}
	searcher.Limit(uint32(limit))
	if docs, err := searcher.Search(hotType + ":1"); err == nil {
		for _, doc := range docs {
			body := doc.Fields["body"]
			if v, ok := doc.Fields[hotType]; ok {
				if vv, err := strconv.Atoi(v); err == nil {
					result[body] = uint32(vv)
				} else {
					result[body] = 0
				}
			}
		}
	}
	searcher.restoreDb()
	return result
}

// GetRelatedQuery 获取相关搜索词列表
func (searcher *Searcher) GetRelatedQuery(query string, limits ...uint8) []string {
	result := []string{}
	limit := cmd.MaxLimit(limits...)
	if query == "" {
		query = searcher.cleanFieldQuery(searcher.query)
	}
	if query == "" || strings.Index(query, ":") >= 0 {
		return result
	}
	op := searcher.defaultOp
	if searcher.SetDB(logDB) != nil {
		return result
	}
	searcher.Limit(uint32(limit + 1))
	searcher.Fuzzy(true)
	docs, err := searcher.Search(query)
	if err == nil {
		for _, doc := range docs {
			body := doc.Fields["body"]
			if strings.Compare(query, body) == 0 {
				continue
			}
			result = append(result, body)
			if len(result) == int(limit) {
				break
			}
		}
	}
	searcher.defaultOp = op
	searcher.restoreDb()
	return result
}

// Close connection
func (searcher *Searcher) Close() {
	if searcher.conn != nil {
		searcher.conn.Close()
		searcher.conn = nil
	}
}

func (searcher *Searcher) setProject(project string) (*Searcher, error) {
	cmdx := cmd.UseProjectCmd(project)

	_, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_PROJECT)

	if err != nil {
		searcher.conn.Close()
		searcher.conn = nil
		searcher.cfg = nil
		return nil, err
	}
	var tokenizer tokenizer.Tokenizer = tokenizer.DefaultTokenizer{"default"}
	searcher.tokenizer = tokenizer
	searcher.queryPrefix = make(map[string]bool)
	searcher.Facets = make(map[string]Facet)
	searcher.curDBs = make(map[string]bool)
	searcher.initSpecialField()
	return searcher, nil
}

func (searcher *Searcher) clearQuery() {
	cmdx := cmd.XsCommand{}
	cmdx.Cmd = cmd.XS_CMD_QUERY_INIT

	searcher.conn.ExecOK(&cmdx, 0)
	searcher.query = ""
	searcher.count = 0
	searcher.terms = nil
}

func (searcher *Searcher) logQuery() {
	if searcher.query == "" {
		return
	}
	query := searcher.query
	if searcher.lastCount == 0 || (searcher.defaultOp == cmd.XS_CMD_QUERY_OP_OR && strings.Index(query, " ") >= 0) || strings.Index(query, " OR ") >= 0 || strings.Index(query, " NOT ") >= 0 || strings.Index(query, " XOR ") >= 0 {
		return
	}
	terms := searcher.Terms("")
	sbuf := bytes.NewBufferString("")
	var pos, max int
	for i := 0; i < len(terms); i++ {
		pos1 := pos
		if pos > 3 && len(terms[i]) == 6 {
			pos1 = pos - 3
		}

		pos2 := strings.Index(query[pos1:], terms[i])
		if pos2 < 0 {
			continue
		}

		if pos2 == pos {
			sbuf.WriteString(terms[i])
		} else if pos2 < pos {
			sbuf.WriteString(terms[i][3:])
		} else {
			max++
			if max > 3 || sbuf.Len() > 42 {
				break
			}
			sbuf.WriteByte(' ')
			sbuf.WriteString(terms[i])
		}
		pos = pos2 + len(terms[i])
	}
	log := strings.Trim(sbuf.String(), " ")
	if len(log) < 2 || (len(log) == 3 && log[0] > 0x80) {
		return
	}
	searcher.addSearchLog(log)
}

func (searcher *Searcher) addSearchLog(query string) error {
	cmdx := cmd.NewCommand(cmd.XS_CMD_SEARCH_ADD_LOG, 0, query)
	_, err := searcher.conn.ExecOK(cmdx, cmd.XS_CMD_OK_LOGGED)
	return err
}

func (searcher *Searcher) cleanFieldQuery(query string) string {
	rp := strings.NewReplacer(" AND ", " ", " OR ", " ")
	query = rp.Replace(query)
	re := regexp.MustCompile("(^|\\s)([0-9A-Za-z_\\.-]+):([^\\s]+)")
	query = cmd.ReplaceAllStringSubmatchFunc(re, query, func(ms []string) string {
		fn := ms[2]
		f, ok := searcher.schema.FieldMetas[fn]
		if !ok {
			return ms[0]
		}
		if f.IsBoolIndex() {
			return ""
		}
		ml := len(ms[3])
		if ms[3][0] == '(' && ms[3][ml-1] == ')' {
			return fmt.Sprintf("%s%s", ms[1], strings.Trim(ms[3], "()"))
		}
		return ms[1]
	})
	return query
}

func (searcher *Searcher) restoreDb() {
	db := searcher.lastDB
	searcher.SetDB(db)
	for d := range searcher.curDBs {
		searcher.AddDB(d)
	}
}
