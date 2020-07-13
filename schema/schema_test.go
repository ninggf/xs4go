package schema

import (
	"testing"
)

func TestSchema_AddTerm(t *testing.T) {
	type fields struct {
		Id         *FieldMeta
		Title      *FieldMeta
		Body       *FieldMeta
		FieldMetas map[string]*FieldMeta
		vnoMap     map[uint8]string
		terms      map[string]schemaTerm
		indexes    map[string]string
	}
	type args struct {
		field  string
		term   string
		weight uint8
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"AddTerm1",
			fields{
				nil, nil, nil, nil, nil, make(map[string]schemaTerm), nil,
			},
			args{
				"test", "hello", 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &Schema{
				Id:         tt.fields.Id,
				Title:      tt.fields.Title,
				Body:       tt.fields.Body,
				FieldMetas: tt.fields.FieldMetas,
				vnoMap:     tt.fields.vnoMap,
				terms:      tt.fields.terms,
				indexes:    tt.fields.indexes,
			}
			sc.AddTerm(tt.args.field, tt.args.term, tt.args.weight)
			v, _ := sc.terms[tt.args.field]
			te, _ := v[tt.args.term]
			if te != tt.args.weight {
				t.Errorf("weight = %v, want = %v", te, tt.args.weight)
			}
			sc.AddTerm(tt.args.field, tt.args.term, tt.args.weight)
			v, _ = sc.terms[tt.args.field]
			te, _ = v[tt.args.term]
			if te != (tt.args.weight * 2) {
				t.Errorf("weight = %v, want = %v", te, (tt.args.weight * 2))
			}
		})
	}
}

func TestSchema_AddIndex(t *testing.T) {
	type fields struct {
		Id         *FieldMeta
		Title      *FieldMeta
		Body       *FieldMeta
		FieldMetas map[string]*FieldMeta
		vnoMap     map[uint8]string
		terms      map[string]schemaTerm
		indexes    map[string]string
	}
	type args struct {
		field string
		index string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"AddIndex",
			fields{
				nil, nil, nil, nil, nil, nil, make(map[string]string),
			},
			args{
				"test", "hello",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &Schema{
				Id:         tt.fields.Id,
				Title:      tt.fields.Title,
				Body:       tt.fields.Body,
				FieldMetas: tt.fields.FieldMetas,
				vnoMap:     tt.fields.vnoMap,
				terms:      tt.fields.terms,
				indexes:    tt.fields.indexes,
			}
			sc.AddIndex(tt.args.field, tt.args.index)
			v, _ := sc.indexes[tt.args.field]
			if v != tt.args.index {
				t.Errorf("index = %v, want = %v", v, tt.args.index)
			}
			sc.AddIndex(tt.args.field, "world")
			v, _ = sc.indexes[tt.args.field]
			if v != (tt.args.index + "\nworld") {
				t.Errorf("index = %v, want = %v", v, (tt.args.index + "\nworld"))
			}
		})
	}
}
