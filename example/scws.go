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

	scws, _ := tokenizer.GetScwsTokenizer()
	scws.SetMulti(tokenizer.SCWS_MULTI_SHORT)
	terms := scws.GetTokens("我非常喜欢golang，中国人也非常喜欢吃大米哦")
	fmt.Println(terms)
	scws.Close()

	scws, _ = tokenizer.GetScwsTokenizer()
	scws.SetMulti(tokenizer.SCWS_MULTI_ZMAIN)
	terms = scws.GetTokens("我非常喜欢golang，中国人也非常喜欢吃大米哦")
	fmt.Println(terms)
	scws.Close()

	scws, _ = tokenizer.GetScwsTokenizer()
	scws.SetMulti(tokenizer.SCWS_MULTI_ZALL)
	terms = scws.GetTokens("我非常喜欢golang，中国人也非常喜欢吃大米哦")
	fmt.Println(terms)
	scws.Close()

	tokenizer.CloseScws()
}
