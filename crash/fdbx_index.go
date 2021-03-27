package crash

import (
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/shestakovda/fdbx/v2"

	"github.com/shestakovda/journal/models"
)

func idxCrash(buf []byte) (map[uint16][]fdb.Key, error) {
	mod := models.GetRootAsFdbxCrash(buf, 0)
	start := fdbx.Time2Byte(time.Unix(0, mod.Created()))
	return map[uint16][]fdb.Key{
		IndexDate: {start},
		IndexCode: {fdbx.AppendRight(mod.Code(), start...)},
	}, nil
}
