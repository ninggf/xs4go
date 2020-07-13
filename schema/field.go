package schema

// Field of document to be indexed
type Field struct {
	Type   string
	Index  string
	Cutlen uint32
	Weight uint16
	Phrase string
	NoBool string `toml:"no_bool"`
	Fid    uint8
}

// FieldMeta of a Field
type FieldMeta struct {
	Field
	Name string
	Flag int
	Vno  uint8
}

// NewField creates meta data of a field
func newField(name string, def Field) *FieldMeta {
	fm := &FieldMeta{def, name, 0, 0}
	fm.preare()
	return fm
}

func (meta *FieldMeta) WithPos() bool {
	return (meta.Flag & FLAG_WITH_POSITION) > 0
}

func (meta *FieldMeta) IsBoolIndex() bool {
	if (meta.Flag & FLAG_NON_BOOL) > 0 {
		return false
	}

	return meta.Type == "id"
}

func (meta *FieldMeta) IsNumeric() bool {
	return meta.Type == "numeric"
}

func (meta *FieldMeta) IsSpecial() bool {
	return meta.Type == "id" || meta.Type == "title" || meta.Type == "body"
}

func (meta *FieldMeta) HasIndexSelf() bool {
	return (meta.Flag & FLAG_INDEX_SELF) > 0
}

func (meta *FieldMeta) HasIndexMixed() bool {
	return (meta.Flag & FLAG_INDEX_MIXED) > 0
}

func (meta *FieldMeta) HasIndex() bool {
	return (meta.Flag & FLAG_INDEX_BOTH) > 0
}

func (field *FieldMeta) String() string {
	return field.Name
}

func (meta *FieldMeta) preare() {
	// type & default setting
	switch meta.Type {
	case "id":
		meta.Flag = FLAG_INDEX_SELF
		break
	case "title":
		meta.Flag = FLAG_INDEX_BOTH | FLAG_WITH_POSITION
		meta.Weight = 5
		break
	case "body":
		meta.Vno = MIXED_VNO
		meta.Flag = FLAG_INDEX_SELF | FLAG_WITH_POSITION
		meta.Cutlen = 300
		break
	default:
	}
	// index flag
	if meta.Index != "" && meta.Type != "body" {
		idx := meta.Index
		flg, ok := INDEX_TYPES[idx]
		if ok {
			meta.Flag = meta.Flag & ^FLAG_INDEX_BOTH
			meta.Flag = meta.Flag | flg
		}

		if meta.Type == "id" {
			meta.Flag = meta.Flag | FLAG_INDEX_SELF
		}
	}

	if meta.Weight <= 0 {
		meta.Weight = 1
	}

	if meta.Weight > 0 && meta.Type == "body" {
		meta.Weight = meta.Weight & MAX_WDF
	}

	switch meta.Phrase {
	case "yes":
	case "YES":
	case "y":
	case "Y":
		meta.Flag = meta.Flag | FLAG_WITH_POSITION
		break
	case "no":
	case "NO":
	case "n":
	case "N":
		meta.Flag &= ^FLAG_WITH_POSITION
		break
	default:
	}

	switch meta.NoBool {
	case "yes":
	case "YES":
	case "y":
	case "Y":
		meta.Flag = meta.Flag | FLAG_NON_BOOL
		break
	case "no":
	case "NO":
	case "n":
	case "N":
		meta.Flag = meta.Flag & ^FLAG_NON_BOOL
		break
	default:
	}
}
