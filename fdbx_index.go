package journal

import (
	"encoding/binary"
	"strings"
	"time"

	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/fdbx/v2/orm"
	"github.com/shestakovda/journal/models"
)

func idxJournal(buf []byte) (map[uint16][]fdbx.Key, error) {
	var mid []byte

	stg := new(models.FdbxStage)
	mod := models.GetRootAsFdbxJournal(buf, 0)

	entp := make([]byte, 4)
	clen := mod.ChainLength()
	keys := make([]fdbx.Key, 0, clen)
	start := fdbx.Time2Byte(time.Unix(0, mod.Start()))

	for i := 0; i < clen; i++ {
		if !mod.Chain(stg, i) {
			return nil, ErrInsert.WithStack()
		}

		if mid = stg.Mid(); len(mid) == 0 {
			continue
		}

		binary.BigEndian.PutUint32(entp, uint32(stg.Mtp()))
		keys = append(keys, fdbx.Bytes2Key(mid).LPart(entp...).RPart(start...))
	}

	return map[uint16][]fdbx.Key{
		IndexStart: []fdbx.Key{fdbx.Bytes2Key(start)},
		IndexModel: keys,
	}, nil
}

func filterByService(services []string) orm.Filter {
	exist := make(map[string]struct{}, len(services))
	for i := range services {
		exist[strings.ToLower(services[i])] = struct{}{}
	}

	return func(row fdbx.Pair) (ok bool, err error) {
		srv := models.GetRootAsFdbxJournal(row.Value(), 0).Service()
		_, ok = exist[strings.ToLower(string(srv))]
		return ok, nil
	}
}
