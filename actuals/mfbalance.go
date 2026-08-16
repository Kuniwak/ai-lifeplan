package actuals

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

const MoneyForwardBalanceDateColumn tsv.ColumnName = "日付"

const MoneyForwardBalanceTotalColumn tsv.ColumnName = "合計（円）"

const yearEnd = "12/31"

func ImportMoneyForwardBalance(r io.Reader, buckets Buckets, known Known) (*tsv.Table, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: %w", err)
	}
	content, err = decodeExport(content)
	if err != nil {
		return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: %w", err)
	}

	records, err := csv.NewReader(bytes.NewReader(tsv.StripByteOrderMark(content))).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: no header row")
	}

	header := records[0]
	dateAt := slices.Index(header, string(MoneyForwardBalanceDateColumn))
	if dateAt < 0 {
		return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: no %q column", MoneyForwardBalanceDateColumn)
	}
	totalAt := slices.Index(header, string(MoneyForwardBalanceTotalColumn))
	if totalAt < 0 {
		return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: no %q column", MoneyForwardBalanceTotalColumn)
	}

	byYear := make(map[date.Year]Balance)
	seen := make(map[date.Year]string)
	for i, record := range records[1:] {
		written := record[dateAt]
		year, day, ok := strings.Cut(written, "/")
		if !ok || day != yearEnd {
			continue
		}
		y, err := date.ParseYear(year)
		if err != nil {
			return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: row %d: %w", i+1, err)
		}
		if before, twice := seen[y]; twice {
			return nil, wording.DuplicateKeyError("actuals.ImportMoneyForwardBalance", "year", wording.Number(y),
				wording.Undecided(fmt.Sprintf("which of %s and %s is the year end balance", before, written)))
		}
		seen[y] = written

		var balance Balance
		var parts money.Yen
		for at, column := range header {
			if at == dateAt || at == totalAt {
				continue
			}

			kind := strings.TrimSuffix(column, "（円）")
			bucket, err := buckets.Of(kind)
			if err != nil {
				return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: %w", err)
			}
			amount, err := money.ParseYen(record[at])
			if err != nil {
				return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: row %d, %q: %w", i+1, column, err)
			}

			parts += amount

			if !buckets.KnownAt(kind, written) {
				elsewhere, ok := known[y][kind]
				if !ok {
					balance.Partial = true
					continue
				}
				amount = elsewhere
			}

			switch bucket {
			case Cash:
				balance.Cash += amount
			case Invested:
				balance.Invested += amount
			default:
				balance.Locked += amount
			}
		}

		total, err := money.ParseYen(record[totalAt])
		if err != nil {
			return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: row %d, %q: %w", i+1, MoneyForwardBalanceTotalColumn, err)
		}
		if parts != total {
			return nil, fmt.Errorf(
				"actuals.ImportMoneyForwardBalance: %s: the parts come to %d but the export says %d", written, parts, total)
		}

		byYear[y] = balance
	}

	if len(byYear) == 0 {
		return nil, fmt.Errorf("actuals.ImportMoneyForwardBalance: no row closes a year on %s", yearEnd)
	}

	years := make([]date.Year, 0, len(byYear))
	for y := range byYear {
		years = append(years, y)
	}
	slices.Sort(years)

	out := &tsv.Table{Header: []tsv.ColumnName{
		BalanceYearColumn, BalanceCashColumn, BalanceInvestedColumn,
		BalanceLockedColumn, BalanceTotalColumn, BalancePartialColumn,
	}}
	for _, y := range years {
		b := byYear[y]
		partial := ""
		if b.Partial {
			partial = string(PartialYes)
		}
		out.Rows = append(out.Rows, []string{
			fmt.Sprint(y), fmt.Sprint(int64(b.Cash)), fmt.Sprint(int64(b.Invested)),
			fmt.Sprint(int64(b.Locked)), fmt.Sprint(int64(b.Total())), partial,
		})
	}
	return out, nil
}

func assertUTF8(content []byte) error {
	if err := tsv.AssertUTF8(content); err != nil {
		return fmt.Errorf("%w (the 資産推移 export never is, as it comes out of MoneyForward)", err)
	}
	return nil
}

func decodeExport(content []byte) ([]byte, error) {
	if utf8.Valid(content) {
		return content, nil
	}
	decoded, err := japanese.ShiftJIS.NewDecoder().Bytes(content)
	if err != nil {
		return nil, fmt.Errorf(
			"the file is neither UTF-8 nor CP932, so nothing here can read it: %w", err)
	}
	if bytes.ContainsRune(decoded, utf8.RuneError) {
		return nil, fmt.Errorf(
			"the file is neither UTF-8 nor CP932: CP932 として読むと置換文字が出る。" +
				"iconv -f CP932 -t UTF-8 <file> で何が起きるか確かめること")
	}
	return decoded, nil
}
