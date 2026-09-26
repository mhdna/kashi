// optimize_images rewrites the receipt photos already on disk (expense
// receipts and close-shift receipts) into the small WebP form that new
// uploads get, and updates their DB rows to match. New uploads are handled by
// api/image_optimize.go; this is the one-off backfill for the older,
// full-size JPEG/PNG originals.
//
// It is a dry run unless -apply is given. Run it from the API's working
// directory (where app.env lives and where the relative file_path values in
// the DB resolve):
//
//	go run ./cmd/optimize_images            # report what would change
//	go run ./cmd/optimize_images -apply     # do it, keeping originals in .orig/
//
// Each rewritten image is written to a new uuid.webp file beside the original
// and the DB row is repointed with a compare-and-swap on the old file_path,
// so the run is safe to repeat, safe to interrupt, and safe against a
// concurrent upload. The original is moved to a .orig/ folder in the same
// directory (-backup=false deletes it instead); once you've eyeballed the
// results, reclaim the space with:
//
//	find <storage-root> -type d -name .orig -prune -exec rm -rf {} +
package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/mhdna/kashi/util"
	"github.com/mhdna/kashi/util/imageopt"
	_ "golang.org/x/image/webp"
)

// Both tables share the columns this tool touches.
var tables = []string{"branch_expense_images", "branch_shift_receipt_images"}

type imageRow struct {
	id          int64
	filePath    string
	contentType string
	sizeBytes   int64
}

type totals struct {
	scanned, converted, skipped, failed int
	before, after                       int64
}

func main() {
	apply := flag.Bool("apply", false, "actually rewrite files and DB rows (default: dry run)")
	backup := flag.Bool("backup", true, "move originals to a .orig/ folder instead of deleting them")
	limit := flag.Int("limit", 0, "stop after converting this many images (0 = no limit)")
	flag.Parse()

	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config (run from the directory containing app.env): ", err)
	}
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	mode := "DRY RUN"
	if *apply {
		mode = "APPLY"
	}
	fmt.Printf("optimize_images [%s]\n", mode)

	var grand totals
	for _, table := range tables {
		t, err := processTable(context.Background(), conn, table, *apply, *backup, *limit-grand.converted, *limit > 0)
		if err != nil {
			log.Fatalf("%s: %v", table, err)
		}
		fmt.Printf("%-30s scanned=%d converted=%d skipped=%d failed=%d  %s -> %s\n",
			table, t.scanned, t.converted, t.skipped, t.failed, human(t.before), human(t.after))
		grand.scanned += t.scanned
		grand.converted += t.converted
		grand.skipped += t.skipped
		grand.failed += t.failed
		grand.before += t.before
		grand.after += t.after
		if *limit > 0 && grand.converted >= *limit {
			break
		}
	}

	saved := grand.before - grand.after
	pct := 0.0
	if grand.before > 0 {
		pct = 100 * float64(saved) / float64(grand.before)
	}
	fmt.Printf("TOTAL converted=%d skipped=%d failed=%d  %s -> %s (saves %s, %.0f%%)\n",
		grand.converted, grand.skipped, grand.failed, human(grand.before), human(grand.after), human(saved), pct)
	if !*apply {
		fmt.Println("dry run: nothing was changed. Re-run with -apply.")
	}
	if grand.failed > 0 {
		os.Exit(1)
	}
}

func processTable(ctx context.Context, conn *sql.DB, table string, apply, backup bool, remaining int, limited bool) (totals, error) {
	var t totals

	rows, err := conn.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, file_path, content_type, size_bytes FROM %s ORDER BY id`, table))
	if err != nil {
		return t, err
	}
	var all []imageRow
	for rows.Next() {
		var r imageRow
		if err := rows.Scan(&r.id, &r.filePath, &r.contentType, &r.sizeBytes); err != nil {
			rows.Close()
			return t, err
		}
		all = append(all, r)
	}
	if err := rows.Close(); err != nil {
		return t, err
	}

	for _, r := range all {
		if limited && t.converted >= remaining {
			break
		}
		t.scanned++

		outcome, before, after, err := convertOne(ctx, conn, table, r, apply, backup)
		switch {
		case err != nil:
			t.failed++
			log.Printf("FAIL %s id=%d %s: %v", table, r.id, r.filePath, err)
		case outcome == "converted":
			t.converted++
			t.before += before
			t.after += after
			fmt.Printf("  %s id=%d %s  %s -> %s\n", table, r.id, filepath.Base(r.filePath), human(before), human(after))
		default:
			t.skipped++
		}
	}
	return t, nil
}

// convertOne returns "converted" or "skipped"; sizes are only meaningful for
// "converted".
func convertOne(ctx context.Context, conn *sql.DB, table string, r imageRow, apply, backup bool) (string, int64, int64, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("skip %s id=%d: file missing on disk: %s", table, r.id, r.filePath)
			return "skipped", 0, 0, nil
		}
		return "", 0, 0, err
	}

	// Already in the target form: re-encoding a WebP we made would only lose
	// quality, so a repeat run must leave these alone.
	if r.contentType == imageopt.ContentType {
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil &&
			cfg.Width <= imageopt.MaxDim && cfg.Height <= imageopt.MaxDim {
			return "skipped", 0, 0, nil
		}
	}

	res := imageopt.Optimize(data, r.contentType, filepath.Ext(r.filePath))
	if !res.Optimized {
		return "skipped", 0, 0, nil
	}
	before, after := int64(len(data)), int64(len(res.Data))

	if !apply {
		return "converted", before, after, nil
	}

	dir := filepath.Dir(r.filePath)
	newPath := filepath.Join(dir, uuid.NewString()+res.Ext)
	if err := writeAtomic(newPath, res.Data); err != nil {
		return "", 0, 0, err
	}

	// Compare-and-swap on the old path: if something else repointed the row
	// in the meantime, don't clobber it and don't leave the new file behind.
	result, err := conn.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s SET file_path = $1, content_type = $2, size_bytes = $3 WHERE id = $4 AND file_path = $5`, table),
		newPath, res.ContentType, after, r.id, r.filePath)
	if err != nil {
		os.Remove(newPath)
		return "", 0, 0, err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		os.Remove(newPath)
		log.Printf("skip %s id=%d: row changed underneath us", table, r.id)
		return "skipped", 0, 0, nil
	}

	// The row now points at the new file; what's left is cleanup, and a
	// failure here costs disk space, not correctness.
	if backup {
		if err := os.MkdirAll(filepath.Join(dir, ".orig"), 0o755); err == nil {
			err = os.Rename(r.filePath, filepath.Join(dir, ".orig", filepath.Base(r.filePath)))
		}
		if err != nil {
			log.Printf("warn: could not back up %s: %v", r.filePath, err)
		}
	} else if err := os.Remove(r.filePath); err != nil {
		log.Printf("warn: could not remove %s: %v", r.filePath, err)
	}
	removeStaleThumbs(dir, filepath.Base(r.filePath))

	return "converted", before, after, nil
}

// removeStaleThumbs drops cached thumbnails of the replaced original (they
// are keyed by its basename, so they'd otherwise be orphaned forever).
func removeStaleThumbs(dir, oldBase string) {
	matches, err := filepath.Glob(filepath.Join(dir, ".thumb", "*_"+oldBase+".*"))
	if err != nil {
		return
	}
	for _, m := range matches {
		os.Remove(m)
	}
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".optimize-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func human(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGT"[exp])
}
