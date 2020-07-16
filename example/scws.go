package main

import (
	"fmt"

	"github.com/ninggf/xs4go/tokenizer"
)

func main() {
	// 当词典文件较大时，请使用scws-gen-dict命令生成xdb文件
	scws, err := tokenizer.NewScwsTokenizer("./dict.txt")
	if err != nil {
		fmt.Println(err)
	}
	scws.SetRule("./rules.utf8.ini")
	scws.SetMulti(tokenizer.SCWS_MULTI_SHORT)
	terms := scws.GetTokens("我非常喜欢golang")
	fmt.Println(terms)
	scws.Close()
}
