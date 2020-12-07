package journal

import (
	"strings"

	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/fdbx/v2/orm"
	"github.com/shestakovda/journal/models"
)

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
