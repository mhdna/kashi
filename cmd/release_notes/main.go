// release_notes turns the commits that shipped since the last posted update
// into rows for the home page's IT Management feed (admin_activities, see
// db/migrations/000121_system_update_notes), posted as Alex.
//
// It doesn't touch a database itself: it reads the routes already posted
// ("/system_updates/<hash>", one per line) on stdin and prints the INSERT on
// stdout, so `make deploy-notes` can pipe both ends through ssh to prod:
//
//	echo /system_updates/d9ff21a | go run ./cmd/release_notes   # what would post after d9ff21a
//
// Every commit is posted, whatever its type prefix ("feat(products): add X"
// and "chore: tidy" alike, or no prefix at all). "feat(products): add X"
// becomes "added X" with the detail "<hash> · products". It starts after the newest commit already
// posted that's in HEAD's history, skips any commit posted before, and with
// nothing posted at all takes the last -limit commits.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const routePrefix = "/system_updates/"

// Commit subjects are imperative ("add", "make"); the feed reads
// "Alex added ...", so the first word goes to the past tense.
var pastTense = map[string]string{
	"add": "added", "allow": "allowed", "cache": "cached", "debounce": "debounced",
	"disable": "disabled", "drop": "dropped", "enable": "enabled", "fix": "fixed",
	"group": "grouped", "hide": "hid", "import": "imported", "improve": "improved",
	"index": "indexed", "keep": "kept", "let": "let", "limit": "limited",
	"make": "made", "match": "matched", "move": "moved", "open": "opened",
	"outline": "outlined", "record": "recorded", "remove": "removed", "rename": "renamed",
	"seed": "seeded", "send": "sent", "set": "set", "show": "showed", "size": "sized",
	"sort": "sorted", "speed": "sped", "split": "split", "stop": "stopped",
	"support": "supported", "update": "updated", "use": "used",
}

// A subject that doesn't start with a known verb ("smart auto-fill on ...",
// "attendance shows all branches") reads as "Alex shipped <subject>".
const fallbackVerb = "shipped"

// Optional "type(scope): " prefix; the type itself is ignored.
var subjectRe = regexp.MustCompile(`^(?:[a-z]+(?:\(([^)]*)\))?!?:\s*)?(.+)$`)

type commit struct {
	hash, date, scope, summary string
}

func main() {
	limit := flag.Int("limit", 20, "commits to consider when nothing has been posted yet")
	author := flag.String("author", "Alex", "user (display name or username) the updates are posted as")
	flag.Parse()

	posted := map[string]bool{}
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		if h, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), routePrefix); ok && h != "" {
			posted[h] = true
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}

	commits, err := newCommits(posted, *limit)
	if err != nil {
		log.Fatal(err)
	}
	if len(commits) == 0 {
		fmt.Fprintln(os.Stderr, "release_notes: nothing new to post")
		return
	}
	for _, c := range commits {
		fmt.Fprintf(os.Stderr, "release_notes: %s %s\n", c.hash, c.summary)
	}
	fmt.Print(insertSQL(commits, *author))
}

// newCommits walks HEAD's history newest-first until it meets a posted
// commit, then returns the unposted ones oldest-first.
func newCommits(posted map[string]bool, limit int) ([]commit, error) {
	args := []string{"log", "--no-merges", "--format=%H%x1f%cI%x1f%s"}
	if len(posted) == 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	var found []commit
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\x1f", 3)
		if len(parts) != 3 {
			continue
		}
		hash := parts[0][:7]
		if posted[hash] {
			break
		}
		m := subjectRe.FindStringSubmatch(parts[2])
		found = append(found, commit{hash: hash, date: parts[1], scope: m[1], summary: describe(m[2])})
	}
	for i, j := 0, len(found)-1; i < j; i, j = i+1, j-1 {
		found[i], found[j] = found[j], found[i]
	}
	return found, nil
}

// Housekeeping tacked onto a subject ("...; fix lint") isn't news.
var choreClauseRe = regexp.MustCompile(`(?i)^(fix(es|ed)? |run |ran )?(the )?(lint|linting|tests?|typos?|formatting|gofmt|cleanup)$`)

// describe words a commit subject for "Alex <summary>".
func describe(subject string) string {
	var kept []string
	for clause := range strings.SplitSeq(subject, ";") {
		if c := strings.TrimSpace(clause); c != "" && !choreClauseRe.MatchString(c) {
			kept = append(kept, c)
		}
	}
	subject = strings.Join(kept, "; ")

	first, rest, _ := strings.Cut(subject, " ")
	if past, ok := pastTense[strings.ToLower(first)]; ok {
		if rest == "" {
			return past
		}
		return past + " " + rest
	}
	return fallbackVerb + " " + subject
}

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// insertSQL posts the commits as author, matched by display name or
// username. It refuses to run (the DO block raises, and deploy-notes runs
// psql with ON_ERROR_STOP) when no user matches, rather than posting rows
// with no user that the home feed can only show as "Someone".
func insertSQL(commits []commit, author string) string {
	match := fmt.Sprintf("name ILIKE %[1]s OR username ILIKE %[1]s", quote(author))
	var b strings.Builder
	fmt.Fprintf(&b, "DO $$ BEGIN\n  IF NOT EXISTS (SELECT 1 FROM users WHERE %s) THEN\n    RAISE EXCEPTION 'release_notes: no user named %%', %s;\n  END IF;\nEND $$;\n", match, quote(author))
	b.WriteString("INSERT INTO admin_activities (user_id, method, route, summary, detail, created_at)\nVALUES\n")
	for i, c := range commits {
		detail := c.hash
		if c.scope != "" {
			detail += " · " + c.scope
		}
		fmt.Fprintf(&b, "  ((SELECT id FROM users WHERE %s ORDER BY id LIMIT 1), 'RELEASE', %s, %s, %s, %s)",
			match, quote(routePrefix+c.hash), quote(c.summary), quote(detail), quote(c.date))
		if i < len(commits)-1 {
			b.WriteString(",\n")
		}
	}
	b.WriteString(";\n")
	return b.String()
}
