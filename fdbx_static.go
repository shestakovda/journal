package journal

import (
	"context"

	"github.com/shestakovda/fdbx/v2/db"
	"github.com/shestakovda/fdbx/v2/orm"
)

//goland:noinspection GoUnusedExportedFunction
func Autovacuum(ctx context.Context, dbc db.Connection, journalID uint16, vcPack int, opts ...orm.Option) {
	go orm.NewTable(journalID, orm.BatchIndex(idxJournal)).Autovacuum(ctx, dbc, vcPack, opts...)
}

//goland:noinspection GoUnusedExportedFunction
func Vacuum(dbc db.Connection, journalID uint16, vcPack int) error {
	return orm.NewTable(journalID, orm.BatchIndex(idxJournal)).Vacuum(dbc, vcPack)
}
