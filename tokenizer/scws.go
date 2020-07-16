package tokenizer

// #cgo CFLAGS: -g -Wall
// #cgo LDFLAGS: -lscws
// #include <stdlib.h>
// #include <scws/scws.h>
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

const (
	SCWS_MULTI_NONE    int = 0x00000 // 无
	SCWS_MULTI_SHORT   int = 0x01000 // 短词
	SCWS_MULTI_DUALITY int = 0x02000 // 二元（将相邻的2个单字组合成一个词）
	SCWS_MULTI_ZMAIN   int = 0x04000 // 重要单字
	SCWS_MULTI_ZALL    int = 0x08000 // 全部单字
	SCWS_MULTI_MASK    int = 0xff000
)

// ScwsTokenizer scws tokenizer depends on scws
type ScwsTokenizer struct {
	scws C.scws_t
}

// NewScwsTokenizer create a scws tokenizer
func NewScwsTokenizer(dict string) (*ScwsTokenizer, error) {
	scws := C.scws_new()

	if scws == nil {
		return nil, fmt.Errorf("cannot initialize scws tokenizer")
	}
	charset := C.CString("utf8")
	defer C.free(unsafe.Pointer(charset))
	C.scws_set_charset(scws, charset)
	tokenizer := ScwsTokenizer{scws: scws}
	var err error
	if strings.HasSuffix(dict, ".xdb") {
		err = tokenizer.SetXdbDict(dict)
	} else if strings.HasSuffix(dict, ".txt") {
		err = tokenizer.SetTxtDict(dict)
	}
	C.scws_set_ignore(scws, C.int(1))
	if err != nil {
		tokenizer.Close()
		return nil, err
	}
	return &tokenizer, nil
}

// AddXdbDict load xdb dict file
func (scws *ScwsTokenizer) AddXdbDict(dict string) error {
	return scws.addDict(dict, 1)
}

// AddTxtDict load text dict file
func (scws *ScwsTokenizer) AddTxtDict(dict string) error {
	return scws.addDict(dict, 4)
}

// SetXdbDict load xdb dict file
func (scws *ScwsTokenizer) SetXdbDict(dict string) error {
	return scws.setDict(dict, 1)
}

// SetTxtDict load text dict file
func (scws *ScwsTokenizer) SetTxtDict(dict string) error {
	return scws.setDict(dict, 4)
}

// SetRule 设定规则集文件。
func (scws *ScwsTokenizer) SetRule(ruleFile string) {
	if scws.scws != nil {
		rule := C.CString(ruleFile)
		defer C.free(unsafe.Pointer(rule))
		C.scws_set_rule(scws.scws, rule)
	}
}

// SetIgnore 设定分词结果是否忽略所有的标点等特殊符号（不会忽略\r和\n
func (scws *ScwsTokenizer) SetIgnore(yes int) {
	if scws.scws != nil {
		C.scws_set_ignore(scws.scws, C.int(yes))
	}
}

// SetMulti 设定分词执行时是否执行针对长词复合切分。（例：“中国人”分为“中国”、“人”、“中国人”）
func (scws *ScwsTokenizer) SetMulti(mode int) {
	if scws.scws != nil {
		C.scws_set_multi(scws.scws, C.int(mode))
	}
}

// SetDuality 设定是否将闲散文字自动以二字分词法聚合
func (scws *ScwsTokenizer) SetDuality(yes int) {
	if scws.scws != nil {
		C.scws_set_duality(scws.scws, C.int(yes))
	}
}

// GetTokens return terms split by scws
func (scws ScwsTokenizer) GetTokens(text string) []string {
	results := []string{}
	if scws.scws == nil || text == "" || strings.Trim(text, "\n\r ") == "" {
		return results
	}
	content := C.CString(text)
	defer C.free(unsafe.Pointer(content))

	buffer := []byte(text)

	C.scws_send_text(scws.scws, content, C.int(len(buffer)))
	for {
		res := C.scws_get_result(scws.scws)
		if res == nil {
			break
		}

		cur := res
		for {
			ln := int(cur.off) + int(cur.len)
			term := buffer[cur.off:ln]
			idf := int(cur.idf)
			if idf > 0 {
				results = append(results, string(term))
			}
			cur = cur.next
			if cur == nil {
				break
			}
		}
		C.scws_free_result(res)
	}
	return results
}

// Close tokenizer to free it's memory
func (scws *ScwsTokenizer) Close() {
	if scws.scws != nil {
		C.scws_free(scws.scws)
		scws.scws = nil
	}
}

// addDict 添加词典文件。
func (scws *ScwsTokenizer) addDict(dict string, mode int) error {
	if scws.scws != nil {
		dic := C.CString(dict)
		defer C.free(unsafe.Pointer(dic))
		rtn := C.scws_add_dict(scws.scws, dic, C.int(mode))
		if rtn == 0 {
			return nil
		}
		return fmt.Errorf("cannot add dict to scws")
	}
	return fmt.Errorf("scws is closed")
}

// setDict 清除并设定当前 scws 操作所有的词典文件。
func (scws *ScwsTokenizer) setDict(dict string, mode int) error {
	if scws.scws != nil {
		dic := C.CString(dict)
		defer C.free(unsafe.Pointer(dic))
		rtn := C.scws_set_dict(scws.scws, dic, C.int(mode))
		if rtn == 0 {
			return nil
		}
		return fmt.Errorf("cannot set dict to scws")
	}
	return fmt.Errorf("scws is closed")
}