package schema

var logger string = `
[fields]

[fields.id]
	type = "id"
	fid  = 1
[fields.pinyin]
	nid  = 2
[fields.partial]
	fid  = 3
[fields.total]
	type="numeric"
	index="self"
	fid  = 4
[fields.lastnum]
	type="numeric"
	index="self"
	fid  = 5
[fields.currnum]
	type="numeric"
	index="self"
	fid  = 6
[fields.currtag]
	fid = 7
[fields.body]
	type="body"
	fid = 8
`
