package journal

import (
	"github.com/golang/glog"
	"github.com/shestakovda/fdbx"
	"github.com/shestakovda/typex"
)

func LoadExistingEntries(cn fdbx.Conn, db fdbx.DB, ids []string) (res []*Entry, err error) {
	mods := make([]Model, 0, len(ids))
	recs := make([]fdbx.Record, 0, len(ids))
	fac := NewFactoryFDB(cn, db).(*fdbFactory)

	for i := range ids {
		m := &fdbModel{fac: fac}

		if m.id, err = typex.ParseUUID(ids[i]); err != nil {
			return nil, ErrValidate.WithReason(err).WithDetail("Некорректный формат идентификатора")
		}

		mods = append(mods, m)
		recs = append(recs, m)
	}

	if err = db.Load(nil, recs...); err != nil {
		// фолбэк - выбираем по одному
		var mod Model

		mods = mods[:0]

		for i := range ids {
			if mod, err = fac.ByID(ids[i]); err != nil {
				glog.Infof("Ошибка загрузки Entry %s: %s", ids[i], err)
				continue
			}

			mods = append(mods, mod)
		}
	}

	res = make([]*Entry, len(mods))
	for i := range mods {
		if res[i], err = mods[i].Export(true); err != nil {
			return
		}
	}

	return res, nil
}
