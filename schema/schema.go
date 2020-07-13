package schema

import (
	"errors"
	"fmt"
	"sort"
)

type schemaTerm map[string]uint8

// Schema of Document to be indexed
type Schema struct {
	Id         *FieldMeta
	Title      *FieldMeta
	Body       *FieldMeta
	StrId      string
	FieldMetas map[string]*FieldMeta
	vnoMap     map[uint8]string
	terms      map[string]schemaTerm
	indexes    map[string]string
}

func newSchema(fields map[string]Field) (*Schema, error) {
	sc := &Schema{}
	sc.FieldMetas = make(map[string]*FieldMeta)
	sc.vnoMap = make(map[uint8]string)
	sc.terms = make(map[string]schemaTerm)
	sc.indexes = make(map[string]string)

	nvoMap := make(map[string]uint8)

	fs := make([]string, 0, len(fields))

	for k := range fields {
		fs = append(fs, k)
	}
	sort.Strings(fs)

	for i, fd := range fs {
		nvoMap[fd] = uint8(i)
	}

	for f, v := range fields {
		fd := newField(f, v)
		if v.Type == "id" {
			if sc.Id == nil {
				sc.FieldMetas[f] = fd
				sc.Id = fd
				sc.StrId = f
			} else {
				return nil, fmt.Errorf("Dumplicated Id field: %s and %s", f, sc.Id.Name)
			}
		} else if v.Type == "title" {
			if sc.Title == nil {
				sc.FieldMetas[f] = fd
				sc.Title = fd
			} else {
				return nil, fmt.Errorf("Dumplicated Title field: %s and %s", f, sc.Title.Name)
			}
		} else if v.Type == "body" {
			if sc.Body == nil {
				sc.FieldMetas[f] = fd
				sc.Body = fd
			} else {
				return nil, fmt.Errorf("Dumplicated Body field: %s and %s", f, sc.Body.Name)
			}
		} else {
			sc.FieldMetas[f] = fd
		}
		if v.Type == "body" {
			fd.Vno = MIXED_VNO
		} else if v.Fid > 0 {
			fd.Vno = v.Fid - 1
		} else {
			fd.Vno = nvoMap[f]
		}
		sc.vnoMap[fd.Vno] = f
	}

	if sc.Id == nil {
		return nil, errors.New("Missing field of type ID")
	}
	return sc, nil
}

// AddTerm to a field of the Schema
func (sc *Schema) AddTerm(field, term string, weight uint8) {
	if weight == 0 {
		weight = 1
	}
	m, ok := sc.terms[field]
	if !ok {
		m = make(schemaTerm)
		sc.terms[field] = m
	}
	t, ok := m[term]
	if ok {
		m[term] = t + weight
	} else {
		m[term] = weight
	}
}

// AddIndex to a field of the Schema
func (sc *Schema) AddIndex(field, index string) {
	if index == "" {
		return
	}

	m, ok := sc.indexes[field]
	if !ok {
		sc.indexes[field] = index
	} else {
		sc.indexes[field] = m + "\n" + index
	}
}

// GetTerms of the field of this schema
func (sc *Schema) GetTerms(field string) (map[string]uint8, bool) {
	term, ok := sc.terms[field]
	if ok {
		return term, ok
	}
	return nil, ok
}

// GetIndex the field of this schema
func (sc *Schema) GetIndex(field string) (string, bool) {
	idx, ok := sc.indexes[field]
	return idx, ok
}

// VnoMap return then value of number of the field
func (sc *Schema) VnoMap() map[uint8]string {
	return sc.vnoMap
}
