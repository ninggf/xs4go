package tokenizer

/*
#cgo CFLAGS: -g -Wall
#cgo LDFLAGS: -lscws
#include <stdlib.h>
#include <scws/scws.h>
int am_i_null(scws_t pointer) {
  if (NULL == pointer) {
    return 0;
  }
  return 1;
}
*/
import "C"

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"unsafe"
)

const (
	SCWS_MULTI_NONE    int = 0x00000 // 无
	SCWS_MULTI_SHORT   int = 0x01000 // 短词
	SCWS_MULTI_DUALITY int = 0x02000 // 二元（将相邻的2个单字组合成一个词）
	SCWS_MULTI_ZMAIN   int = 0x04000 // 重要单字
	SCWS_MULTI_ZALL    int = 0x08000 // 全部单字
	SCWS_MULTI_MASK    int = 0xff000
	SCWS_XDICT_XDB     int = 1
	SCWS_XDICT_MEM     int = 2
	SCWS_XDICT_TXT     int = 4
)

// ScwsTokenizer scws tokenizer depends on scws
type ScwsTokenizer struct {
	scws  C.scws_t
	inUse bool
}

type kv struct {
	Key   string
	Value int
}

var (
	gscws   *ScwsTokenizer
	mutex   sync.Mutex
	fmutex  sync.Mutex
	closeCh chan C.scws_t
	quitCh  chan int
)

// InitScws 初始化全局的scws实例.
func InitScws(dict string, rule ...string) error {
	mutex.Lock()
	defer mutex.Unlock()
	if gscws == nil {
		var err error
		gscws, err = newScwsTokenizer(dict)
		if err != nil {
			return err
		}

		if len(rule) > 0 {
			gscws.SetRule(rule[0])
		}

		closeCh = make(chan C.scws_t, 4096)
		quitCh = make(chan int)

		go func() {
			for {
				select {
				case scwst := <-closeCh:
					fmutex.Lock()
					C.scws_free(scwst)
					fmutex.Unlock()
				case <-quitCh:
					gscws = nil
					return
				}
			}
		}()
	}
	return nil
}

// CloseScws close scws
func CloseScws() {
	if gscws != nil && gscws.inUse {
		gscws.inUse = false
		closeCh <- gscws.scws
		quitCh <- 1
	}
}

// GetScwsTokenizer 获取一个scws实例，该方法必须在InitScws之后调用。
func GetScwsTokenizer() (*ScwsTokenizer, error) {
	if gscws == nil || C.am_i_null(gscws.scws) == C.int(0) {
		return nil, fmt.Errorf("please call tokenizer.InitScws first")
	}
	return gscws.fork()
}

// AddXdbDict load xdb dict file
func (scws *ScwsTokenizer) AddXdbDict(dict string) error {
	return scws.addDict(dict, 1^2)
}

// AddTxtDict load text dict file
func (scws *ScwsTokenizer) AddTxtDict(dict string) error {
	return scws.addDict(dict, 4^2)
}

// SetXdbDict load xdb dict file
func (scws *ScwsTokenizer) SetXdbDict(dict string) error {
	return scws.setDict(dict, 1^2)
}

// SetTxtDict load text dict file
func (scws *ScwsTokenizer) SetTxtDict(dict string) error {
	return scws.setDict(dict, 4^2)
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
	if scws.scws == nil || text == "" || strings.Trim(text, "\n\r ") == "" {
		return []string{}
	}
	content := C.CString(text)
	defer C.free(unsafe.Pointer(content))

	buffer := []byte(text)
	resMap := map[string]int{}
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
				termx := string(term)
				if _, ok := resMap[termx]; ok {
					resMap[termx]++
				} else {
					resMap[termx] = 1
				}
			}
			cur = cur.next
			if cur == nil {
				break
			}
		}
		C.scws_free_result(res)
	}
	//以下对词进行排序
	ss := []kv{}
	for k, v := range resMap {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	results := make([]string, len(resMap))
	for i, v := range ss {
		results[i] = v.Key
	}
	return results
}

// Close tokenizer to free it's memory
func (scws *ScwsTokenizer) Close() {
	if scws != nil && scws.inUse {
		scws.inUse = false
		closeCh <- scws.scws
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

// newScwsTokenizer create a scws tokenizer
func newScwsTokenizer(dict string) (*ScwsTokenizer, error) {
	scws := C.scws_new()

	if scws == nil || C.am_i_null(scws) == C.int(0) {
		return nil, fmt.Errorf("cannot initialize scws tokenizer")
	}
	charset := C.CString("utf8")
	defer C.free(unsafe.Pointer(charset))
	C.scws_set_charset(scws, charset)
	tokenizer := ScwsTokenizer{scws: scws, inUse: true}
	var err error
	if strings.HasSuffix(dict, ".xdb") {
		err = tokenizer.SetXdbDict(dict)
	} else if strings.HasSuffix(dict, ".txt") {
		err = tokenizer.SetTxtDict(dict)
	}
	if err != nil {
		tokenizer.Close()
		return nil, err
	}
	C.scws_set_ignore(scws, C.int(1))
	return &tokenizer, nil
}

// fork a scws
func (scws *ScwsTokenizer) fork() (*ScwsTokenizer, error) {
	fmutex.Lock()
	scwsForked := C.scws_fork(scws.scws)
	fmutex.Unlock()
	if scwsForked == nil || C.am_i_null(scwsForked) == C.int(0) {
		return nil, fmt.Errorf("cannot fork a new scws instance for OOM")
	}

	charset := C.CString("utf8")
	defer C.free(unsafe.Pointer(charset))
	C.scws_set_charset(scwsForked, charset)
	C.scws_set_ignore(scwsForked, C.int(1))
	sc := &ScwsTokenizer{scwsForked, true}

	return sc, nil
}
