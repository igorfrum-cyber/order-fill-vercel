package xlsx

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"order-fill/services/document-service/internal/domain/orderfill"
)

func TestManualEditsSurviveSaveOnTyumenSource(t *testing.T) {
	sourceBytes, err := os.ReadFile(privateTestdata(t, "Ангио Тюмень .xlsx"))
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	blankBytes, err := os.ReadFile(privateTestdata(t, "2026 08 25 Бланк заказа ANGIOPHARM.xlsx"))
	if err != nil {
		t.Fatalf("read blank: %v", err)
	}

	codec := NewCodec()
	source, err := codec.Load(sourceBytes)
	if err != nil {
		t.Fatalf("load source: %v", err)
	}
	blank, err := codec.Load(blankBytes)
	if err != nil {
		t.Fatalf("load blank: %v", err)
	}
	result, err := orderfill.Fill(orderfill.FillCommand{
		Source:     source,
		Blank:      blank,
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
	})
	if err != nil {
		t.Fatalf("fill: %v", err)
	}

	var target orderfill.ReportRow
	for _, row := range result.Rows {
		if row.Editable && row.SourceRow != nil && row.Inserted != nil {
			target = row
			break
		}
	}
	if target.Key == "" || target.SourceRow == nil {
		t.Fatal("need a matched editable row with a source line")
	}

	if err := orderfill.ApplyFinalEdits(orderfill.FinalizeCommand{
		Source: result.Source,
		Blank:  result.Blank,
		Rows:   result.Rows,
		Brand:  "angiopharm",
		Edits: []orderfill.ManualEdit{{
			Key:     target.Key,
			Value:   "18",
			Comment: "договорились с поставщиком",
		}},
	}); err != nil {
		t.Fatalf("apply edits: %v", err)
	}

	savedSource, err := result.Source.Save()
	if err != nil {
		t.Fatalf("save source: %v", err)
	}
	reloaded, err := codec.Load(savedSource)
	if err != nil {
		t.Fatalf("reload source: %v", err)
	}
	detection, err := orderfill.DetectSourceColumns(reloaded)
	if err != nil {
		t.Fatalf("detect columns: %v", err)
	}
	row := *target.SourceRow
	gotFact := detection.Sheet.Value(row, detection.Columns[orderfill.ColumnOrderedFact])
	gotComment := detection.Sheet.Value(row, detection.Columns[orderfill.ColumnComment])
	t.Logf("source row %d article=%q fact=%q comment=%q columns fact=%d comment=%d",
		row, target.SourceArticle, gotFact, gotComment,
		detection.Columns[orderfill.ColumnOrderedFact], detection.Columns[orderfill.ColumnComment])
	if gotFact != "18" {
		t.Fatalf("1C «Заказано по факту» = %q, want 18", gotFact)
	}
	if gotComment != "договорились с поставщиком" {
		t.Fatalf("1C «Комментарий» = %q, want the reviewer comment", gotComment)
	}
}

func TestBenchPrivate100kPipeline(t *testing.T) {
	if os.Getenv("ORDERFILL_BENCH") == "" {
		t.Skip("set ORDERFILL_BENCH=1 to time load/fill/save on testdata/private")
	}

	sourcePath := privateTestdata(t, "source_100000.xlsx")
	blankPath := privateTestdata(t, "2026 08 25 Бланк заказа ANGIOPHARM.xlsx")

	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	blankBytes, err := os.ReadFile(blankPath)
	if err != nil {
		t.Fatalf("read blank: %v", err)
	}

	codec := NewCodec()
	started := time.Now()
	source, err := codec.Load(sourceBytes)
	if err != nil {
		t.Fatalf("load source: %v", err)
	}
	t.Logf("load source: %s (%d bytes)", time.Since(started).Round(time.Millisecond), len(sourceBytes))

	started = time.Now()
	blank, err := codec.Load(blankBytes)
	if err != nil {
		t.Fatalf("load blank: %v", err)
	}
	t.Logf("load blank: %s (%d bytes)", time.Since(started).Round(time.Millisecond), len(blankBytes))

	started = time.Now()
	result, err := orderfill.Fill(orderfill.FillCommand{
		Source:     source,
		Blank:      blank,
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
	})
	if err != nil {
		t.Fatalf("fill: %v", err)
	}
	t.Logf("fill: %s (rows=%d)", time.Since(started).Round(time.Millisecond), len(result.Rows))

	started = time.Now()
	if _, err := result.Source.Save(); err != nil {
		t.Fatalf("save source: %v", err)
	}
	t.Logf("save source: %s", time.Since(started).Round(time.Millisecond))

	started = time.Now()
	if _, err := result.Blank.Save(); err != nil {
		t.Fatalf("save blank: %v", err)
	}
	t.Logf("save blank: %s", time.Since(started).Round(time.Millisecond))
}

func privateTestdata(t *testing.T, name string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "testdata", "private", name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skipf("missing testdata/private/%s", name)
	return ""
}
