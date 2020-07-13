# xs4go
xs golang sdk

## 文档

建立索引: 参考[Xunsearch索引](http://www.xunsearch.com/doc/php/guide/index.overview)部分。

搜索: 参考[Xunsearch搜索](http://www.xunsearch.com/doc/php/guide/search.overview)部分。

## 配置文件

参考`test/demox.toml`和[Xunsearch](http://www.xunsearch.com/doc/php/guide/ini.guide)官方。

## 分词器

请自己实现如下接口：

```go
type Tokenizer interface {
    GetTokens(text string) []string
}
```

然后将其设置为分词器:

```go
index, err := xs.NewIndexer("./demox.toml")

index.SetTokenizer(yourTokenizer)
```