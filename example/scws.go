package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ninggf/xs4go/tokenizer"
)

func init() {
	// 当词典文件较大时，请使用scws-gen-dict命令生成xdb文件
	err := tokenizer.InitScws("./dict.xdb", "./rules.utf8.ini")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	for i := 0; i < 20; i++ {
		fmt.Println(getTks("我非常喜欢golang，中国人也非常喜欢吃大米哦"))
	}
	http.HandleFunc("/", indexHandler)
	http.ListenAndServe(":30809", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, getTks("我非常喜欢golang，中国人也非常喜欢吃大米哦"))
}

func getTks(str string) []string {
	scws, err := tokenizer.GetScwsTokenizer()
	defer scws.Close()
	if err != nil {
		//panic(err)
		return []string{}
	}
	scws.SetMulti(tokenizer.SCWS_MULTI_ZALL)
	return scws.GetTokens(str)
}
