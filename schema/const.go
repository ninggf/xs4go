package schema

const (
	MAX_WDF            = 0x3f
	MIXED_VNO          = 255
	TYPE_STRING        = 0
	TYPE_NUMERIC       = 1
	TYPE_DATE          = 2
	TYPE_ID            = 10
	TYPE_TITLE         = 11
	TYPE_BODY          = 12
	FLAG_INDEX_SELF    = 0x01
	FLAG_INDEX_MIXED   = 0x02
	FLAG_INDEX_BOTH    = 0x03
	FLAG_WITH_POSITION = 0x10
	FLAG_NON_BOOL      = 0x80
)

var INDEX_TYPES = map[string]int{
	"self":  0x01,
	"mixed": 0x02,
	"both":  0x03,
}
