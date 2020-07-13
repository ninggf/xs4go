package test

import (
	"strconv"
	"testing"

	xs "github.com/ninggf/xs4go"
)

func newIndexer(t *testing.T) *xs.Indexer {
	index, err := xs.NewIndexer("./demox.toml")
	if err != nil {
		t.Error(err)
	}
	return index
}

func Test_NewIndexer(t *testing.T) {
	index := newIndexer(t)
	doc := make(map[string]string)
	doc["id"] = strconv.Itoa(1018)
	doc["message"] = "中国 日本 厉害"
	index.Add(doc)

	index.Close()
}

func TestIndexer_Del(t *testing.T) {
	index := newIndexer(t)
	//index.Del("1012", "1015")
	index.DelByField("message", "中国")
	index.Close()
}

func TestIndexer_AddSynonym(t *testing.T) {
	index := newIndexer(t)
	index.OpenBuffer(1)
	err := index.AddSynonym("搜索", "查找", "检索", "Search")
	if err != nil {
		t.Error(err)
	}
	err = index.AddSynonym("Hello", "Hi", "Hay", "你好")
	if err != nil {
		t.Error(err)
	}
	err = index.Submit()
	index.Close()
}

func TestIndexer_DelSynonym(t *testing.T) {
	index := newIndexer(t)
	index.OpenBuffer(1)
	err := index.DelSynonym("你", "伊", "OK", "book")
	if err != nil {
		t.Error(err)
	}
	err = index.Submit()
	index.Close()
}

func TestIndexer_Rebuild(t *testing.T) {
	index := newIndexer(t)

	if err := index.EndRebuild(); err != nil {
		t.Error(err)
	}

	index.Close()
}

func TestIndexer_FlushLoggin(t *testing.T) {
	index := newIndexer(t)

	if err := index.FlushLogging(); err != nil {
		t.Error(err)
	}

	index.Close()
}
