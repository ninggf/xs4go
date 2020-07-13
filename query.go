package xs

import (
	"bytes"
	"math"
	"regexp"
	"strings"
	"sync"

	"github.com/ninggf/xs4go/cmd"
	"github.com/ninggf/xs4go/schema"
)

var mux sync.Mutex

// preQueryString
//
// 搜索语句的准备工作,登记相关的字段前缀并给非布尔字段补上括号
func (searcher *Searcher) preQueryString(query string) string {
	mux.Lock()
	defer mux.Unlock()

	query = strings.Trim(query, " \n\t\r")

	replacer := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")
	query = replacer.Replace(query)

	newQuery := bytes.NewBufferString("")
	parts := strings.Split(query, " ")
	reg := regexp.MustCompile("[\\\\x81-\\\\xfe]")

	for _, part := range parts {
		if part == "" {
			continue
		}
		newQuery.WriteByte(' ')
		if pos := strings.Index(part, ":"); pos > 0 {
			var i int
			for i = 0; i < pos; i++ {
				if ppos := strings.Index("+-~(", part[i:i+1]); ppos == -1 {
					break
				}
			}
			name := part[i:pos] // 字段名
			if field, ok := searcher.schema.FieldMetas[name]; ok && field.Vno != schema.MIXED_VNO {
				searcher.regQueryPrefix(name)
				prefix := ""
				suffix := ""
				value := part[pos+1:]
				if i > 0 {
					prefix = part[0:i]
				}
				if strings.HasSuffix(part, ")") {
					suffix = ")"
					value = part[pos+1 : len(part)-1]
				}

				terms := searcher.tokenizer.GetTokens(value)
				for i, term := range terms {
					terms[i] = strings.ToLower(term)
				}
				value = strings.Join(terms, " "+name+":")
				newQuery.WriteString(prefix)
				newQuery.WriteString(name)
				newQuery.WriteByte(':')
				newQuery.WriteString(value)
				newQuery.WriteString(suffix)
				continue
			}
		}
		if len(part) > 1 && (part[0:1] == "+" || part[0:1] == "-") && part[1:2] != "(" && reg.Match([]byte(part)) {
			newQuery.WriteString(part[0:1])
			newQuery.WriteByte('(')
			newQuery.WriteString(part[1:])
			newQuery.WriteByte(')')
			//newQuery .= substr($part, 0, 1) . '(' . substr($part, 1) . ')';
			continue
		}
		newQuery.WriteString(part)
	}
	return strings.TrimLeft(newQuery.String(), " ")
}

func (searcher *Searcher) regQueryPrefix(name string) {
	_, ok := searcher.queryPrefix[name]
	if !ok {
		if field, ok := searcher.schema.FieldMetas[name]; ok && field.Vno != schema.MIXED_VNO {
			typeb := cmd.XS_CMD_PREFIX_NORMAL
			if field.IsBoolIndex() {
				typeb = cmd.XS_CMD_PREFIX_BOOLEAN
			}
			cmdx := cmd.NewCommand2(cmd.XS_CMD_QUERY_PREFIX, uint8(typeb), field.Vno, name)
			if _, err := searcher.conn.ExecOK(cmdx, 0); err != nil {
				searcher.queryPrefix[name] = false
				return
			}
			searcher.queryPrefix[name] = true
		}
	}
}

// initSpecialField 初始化一些特殊的字段
func (searcher *Searcher) initSpecialField() {
	for _, field := range searcher.schema.FieldMetas {
		if field.Cutlen > 0 {
			cln := uint8(math.Ceil(float64(field.Cutlen) / 10.0))
			if cln > 127 {
				cln = 127
			}
			searcher.conn.ExecOK(cmd.NewCommand2(cmd.XS_CMD_SEARCH_SET_CUT, cln, field.Vno), 0)
		}
		if field.IsNumeric() {
			searcher.conn.ExecOK(cmd.NewCommand2(cmd.XS_CMD_SEARCH_SET_NUMERIC, 0, field.Vno), 0)
		}
	}
}
