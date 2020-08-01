package main

import (
	"fmt"

	"github.com/ninggf/xs4go/tokenizer"
)

func main() {
	// 当词典文件较大时，请使用scws-gen-dict命令生成xdb文件
	err := tokenizer.InitScws("./dict.xdb", "./rules.utf8.ini")
	if err != nil {
		fmt.Println(err)
	}

	for i := 0; i < 1000; i++ {
		fmt.Println(getTks("我非常喜欢golang，中国人也非常喜欢吃大米哦"))
	}

	tokenizer.CloseScws()
}

func getTks(str string) []string {
	scws, err := tokenizer.GetScwsTokenizer()
	defer scws.Close()
	if err != nil {
		fmt.Println(err)
		return []string{}
	}
	scws.SetMulti(tokenizer.SCWS_MULTI_ZALL)
	return scws.GetTokens(str)
}
