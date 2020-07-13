package test

import (
	"fmt"
	"testing"

	xs "github.com/ninggf/xs4go"
)

func newSearcher(t *testing.T) *xs.Searcher {
	search, err := xs.NewSearcher("./demox.toml")
	if err != nil {
		t.Error(err)
	}
	return search
}

func Test_NewSearcher(t *testing.T) {
	searcher := newSearcher(t)
	if err := searcher.SetCutOff(50, 2.5); err != nil {
		t.Error(err)
	}
	searcher.Close()
}

func TestSearcher_GetAllSynonyms(t *testing.T) {
	searcher := newSearcher(t)
	synonyms := searcher.GetAllSynonyms(0, 0, false)
	fmt.Errorf("%v", synonyms)
	searcher.Close()
}

func TestSearcher_GetSynonyms(t *testing.T) {
	searcher := newSearcher(t)
	synonyms := searcher.GetSynonyms("搜索")
	fmt.Errorf("%v", synonyms)
	searcher.Close()
}

func TestSearcher_SetDB(t *testing.T) {
	searcher := newSearcher(t)
	err := searcher.AddDB("db")
	if err != nil {
		t.Errorf("%v", err)
	}
	searcher.Close()
}

func TestSearcher_GetQuery(t *testing.T) {
	query := " \t-id:上海  人民\t公园  \n"
	searcher := newSearcher(t)
	q, _ := searcher.GetQuery(query)
	if q != "Query(((人民@1 AND 公园@2) AND_NOT A上海))" {
		t.Errorf("%v != Query(((人民@1 AND 公园@2) AND_NOT A上海))", q)
	}
	query = "杭州 ADJ/3 西湖"
	q, _ = searcher.GetQuery(query)
	if q != "Query((杭州@1 AND (adj@2 PHRASE 2 3@3) AND 西湖@4))" {
		t.Errorf("%v != Query((杭州@1 AND (adj@2 PHRASE 2 3@3) AND 西湖@4))", q)
	}
	query = " message:上海 人民公园"
	q, _ = searcher.GetQuery(query)
	if q != "Query((Zmessag@1 AND 上海@2 AND 人民@3 AND 公园@4))" {
		t.Errorf("%v != Query((Zmessag@1 AND 上海@2 AND 人民@3 AND 公园@4))", q)
	}
	query = " 神雕侠侣 -电视剧"
	q, _ = searcher.GetQuery(query)
	if q != "Query(((神雕侠侣@1 SYNONYM (神雕@78 AND 侠侣@79)) AND_NOT (电视剧@2 SYNONYM (电视@79 AND 视剧@80))))" {
		t.Errorf("%v != Query(((神雕侠侣@1 SYNONYM (神雕@78 AND 侠侣@79)) AND_NOT (电视剧@2 SYNONYM (电视@79 AND 视剧@80))))", q)
	}
	query = "((杭州 AND 西湖) OR (杭州 AND 西溪湿地)) NOT (汽车 火车)"
	q, _ = searcher.GetQuery(query)
	if q != "Query((((杭州@1 AND 西湖@2) OR (杭州@3 AND (西溪@4 AND 湿地@5))) AND_NOT (汽车@6 AND 火车@7)))" {
		t.Errorf("%v != Query((((杭州@1 AND 西湖@2) OR (杭州@3 AND (西溪@4 AND 湿地@5))) AND_NOT (汽车@6 AND 火车@7)))", q)
	}
	searcher.Close()
}

func TestSearcher_GetDbTotal(t *testing.T) {
	searcher := newSearcher(t)
	total := searcher.GetDbTotal()
	if total != 1 {
		t.Errorf("db total %v != 1", total)
	}
	searcher.Close()
}
func TestSearcher_Count(t *testing.T) {
	searcher := newSearcher(t)
	total := searcher.Count("日本")
	if total != 1 {
		t.Errorf("db total %v != 1", total)
	}
	searcher.Close()
}

func TestSearcher_Search(t *testing.T) {
	searcher := newSearcher(t)
	searcher.SetQuery("日本")
	docs, err := searcher.Search()
	if err != nil {
		t.Errorf("db total %v != 1", err)
	}
	t.Error(docs[0])
	searcher.Close()
}

func TestSearcher_GetCorrectedQuery(t *testing.T) {
	searcher := newSearcher(t)
	searcher.SetQuery("message:日本")
	searcher.Search()
	docs := searcher.GetCorrectedQuery()

	t.Error(docs)
	searcher.Close()
}

func TestSearcher_GetHotQuery(t *testing.T) {
	searcher := newSearcher(t)
	searcher.GetHotQuery("total")
	searcher.Close()
}

func TestSearcher_GetRelatedQuery(t *testing.T) {
	searcher := newSearcher(t)
	searcher.GetRelatedQuery("日本")
	searcher.Close()
}
