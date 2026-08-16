package law

import "github.com/Kuniwak/lifeplan/tsv"

const (
	LawStartYearColumn tsv.ColumnName = "適用開始年"
	LawEndYearColumn   tsv.ColumnName = "適用終了年"
	LawSourceColumn    tsv.ColumnName = "出典"
)

type reader struct {
	tsv.Reader
}

func newReader(table *tsv.Table, name string, columns ...tsv.ColumnName) (reader, error) {
	r, err := tsv.NewReader(table, tsv.Slot(name), columns...)
	if err != nil {
		return reader{}, err
	}
	return reader{Reader: r}, nil
}
