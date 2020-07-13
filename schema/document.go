package schema

import "github.com/ninggf/xs4go/cmd"

// DocResSize is the len of the doc buf
const (
	DocResSize   int    = 20
	DocResFormat string = "Idocid/Irank/Iccount/ipercent/fweight"
)

// Document for search result
type Document struct {
	Charset string
	Fields  map[string]string
	Docid   uint32
	Rank    uint32
	Ccount  uint32
	Percent int32
	Weight  float32
	Matched []string
}

// NewDocument creates document instance from meta
func NewDocument(meta string) (*Document, error) {
	doc := &Document{}
	doc.Fields = make(map[string]string)
	if len(meta) != DocResSize {
		doc.Charset = meta
	} else {
		metas, err := cmd.UnPack(DocResFormat, meta)
		if err != nil {
			return nil, err
		}
		doc.Docid = metas["docid"].(uint32)    //4
		doc.Rank = metas["rank"].(uint32)      //4
		doc.Ccount = metas["ccount"].(uint32)  //4
		doc.Percent = metas["percent"].(int32) //4
		doc.Weight = metas["weight"].(float32) //4
	}
	return doc, nil
}
