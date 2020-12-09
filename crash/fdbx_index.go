package crash

import (
	"time"

	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/journal/models"
)

func idxCrash(buf []byte) (map[uint16][]fdbx.Key, error) {
	mod := models.GetRootAsFdbxCrash(buf, 0)
	start := fdbx.Bytes2Key(fdbx.Time2Byte(time.Unix(0, mod.Created())))
	return map[uint16][]fdbx.Key{
		IndexDate: []fdbx.Key{start},
		IndexCode: []fdbx.Key{start.LPart(mod.Code()...)},
	}, nil
}
