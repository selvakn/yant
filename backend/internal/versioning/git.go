package versioning

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Version represents a single point-in-time snapshot derived from a git commit.
type Version struct {
	CommitHash string
	ShortHash  string
	Timestamp  time.Time
	Message    string
	Insertions int
	Deletions  int
}

// DiffLine represents a single line in a parsed unified diff.
type DiffLine struct {
	Type      string // "add", "remove", "context", "header"
	Content   string
	OldLineNo int
	NewLineNo int
}

// DiffResult holds a complete diff between two versions.
type DiffResult struct {
	OldCommit        string
	NewCommit        string
	OldDate          time.Time
	NewDate          time.Time
	Lines            []DiffLine
	HasDrawingChange bool
}

var commitHashRe = regexp.MustCompile(`^[0-9a-f]{4,40}$`)

func ValidCommitHash(hash string) bool {
	return commitHashRe.MatchString(hash)
}

func Init(notesDir string) error {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return fmt.Errorf("versioning: resolve path: %w", err)
	}

	if isGitRepo(absDir) {
		return nil
	}

	if err := gitCmd(absDir, "init"); err != nil {
		return fmt.Errorf("versioning: git init: %w", err)
	}
	if err := gitCmd(absDir, "config", "user.email", "yant@localhost"); err != nil {
		return fmt.Errorf("versioning: git config email: %w", err)
	}
	if err := gitCmd(absDir, "config", "user.name", "yant"); err != nil {
		return fmt.Errorf("versioning: git config name: %w", err)
	}

	out, err := gitOutput(absDir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("versioning: git status: %w", err)
	}
	if strings.TrimSpace(out) == "" {
		return nil
	}

	if err := gitCmd(absDir, "add", "-A"); err != nil {
		return fmt.Errorf("versioning: git add: %w", err)
	}
	if err := gitCmd(absDir, "commit", "-m", "seed: initial version of all notes"); err != nil {
		return fmt.Errorf("versioning: git commit (seed): %w", err)
	}

	return nil
}

func CommitFile(notesDir, relPath, message string) error {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return err
	}
	if err := gitCmd(absDir, "add", relPath); err != nil {
		return fmt.Errorf("versioning: git add %s: %w", relPath, err)
	}

	out, err := gitOutput(absDir, "diff", "--cached", "--name-only")
	if err != nil {
		return fmt.Errorf("versioning: git diff --cached: %w", err)
	}
	if strings.TrimSpace(out) == "" {
		return nil
	}

	if err := gitCmd(absDir, "commit", "-m", message); err != nil {
		return fmt.Errorf("versioning: git commit: %w", err)
	}
	return nil
}

func CommitDelete(notesDir, relPath, message string) error {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return err
	}
	if err := gitCmd(absDir, "rm", "--cached", "--ignore-unmatch", relPath); err != nil {
		return fmt.Errorf("versioning: git rm %s: %w", relPath, err)
	}

	out, err := gitOutput(absDir, "diff", "--cached", "--name-only")
	if err != nil {
		return fmt.Errorf("versioning: git diff --cached: %w", err)
	}
	if strings.TrimSpace(out) == "" {
		return nil
	}

	if err := gitCmd(absDir, "commit", "-m", message); err != nil {
		return fmt.Errorf("versioning: git commit: %w", err)
	}
	return nil
}

func Log(notesDir, relPath string, limit, offset int) ([]Version, error) {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return nil, err
	}

	if !isGitRepo(absDir) {
		return nil, nil
	}

	args := []string{"log", "--follow", "--format=%H|%h|%aI|%s", "--numstat"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--skip=%d", offset), fmt.Sprintf("-n"), fmt.Sprintf("%d", limit))
	}
	args = append(args, "--", relPath)

	out, err := gitOutput(absDir, args...)
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, fmt.Errorf("versioning: git log: %w", err)
	}

	return parseGitLog(out), nil
}

func Show(notesDir, relPath, commitHash string) (string, error) {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return "", err
	}
	ref := commitHash + ":" + relPath
	out, err := gitOutput(absDir, "show", ref)
	if err != nil {
		return "", fmt.Errorf("versioning: git show %s: %w", ref, err)
	}
	return out, nil
}

func Diff(notesDir, relPath, oldCommit, newCommit string) (string, error) {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return "", err
	}
	out, err := gitOutput(absDir, "diff", oldCommit, newCommit, "--", relPath)
	if err != nil {
		return "", fmt.Errorf("versioning: git diff: %w", err)
	}
	return out, nil
}

func ParentCommit(notesDir, commitHash string) (string, error) {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return "", err
	}
	out, err := gitOutput(absDir, "rev-parse", commitHash+"^")
	if err != nil {
		return "", fmt.Errorf("versioning: no parent for %s: %w", commitHash, err)
	}
	return strings.TrimSpace(out), nil
}

func GetVersion(notesDir, commitHash string) (*Version, error) {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return nil, err
	}
	out, err := gitOutput(absDir, "log", "-1", "--format=%H|%h|%aI|%s", commitHash)
	if err != nil {
		return nil, fmt.Errorf("versioning: git log for %s: %w", commitHash, err)
	}
	versions := parseGitLog(out)
	if len(versions) == 0 {
		return nil, fmt.Errorf("versioning: commit %s not found", commitHash)
	}
	return &versions[0], nil
}

func FileExistsAtCommit(notesDir, relPath, commitHash string) bool {
	absDir, _ := filepath.Abs(notesDir)
	_, err := gitOutput(absDir, "cat-file", "-t", commitHash+":"+relPath)
	return err == nil
}

func ParseDiff(raw string) []DiffLine {
	var lines []DiffLine
	scanner := bufio.NewScanner(strings.NewReader(raw))
	oldLine, newLine := 0, 0

	hunkRe := regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "---"), strings.HasPrefix(line, "+++"):
			lines = append(lines, DiffLine{Type: "header", Content: line})
		case hunkRe.MatchString(line):
			m := hunkRe.FindStringSubmatch(line)
			oldLine, _ = strconv.Atoi(m[1])
			newLine, _ = strconv.Atoi(m[2])
			lines = append(lines, DiffLine{Type: "header", Content: line})
		case strings.HasPrefix(line, "+"):
			lines = append(lines, DiffLine{
				Type: "add", Content: line[1:],
				NewLineNo: newLine,
			})
			newLine++
		case strings.HasPrefix(line, "-"):
			lines = append(lines, DiffLine{
				Type: "remove", Content: line[1:],
				OldLineNo: oldLine,
			})
			oldLine++
		default:
			if len(line) > 0 && line[0] == ' ' {
				line = line[1:]
			}
			lines = append(lines, DiffLine{
				Type: "context", Content: line,
				OldLineNo: oldLine, NewLineNo: newLine,
			})
			oldLine++
			newLine++
		}
	}
	return lines
}

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func gitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, string(output))
	}
	return string(output), nil
}

func parseGitLog(raw string) []Version {
	var versions []Version
	scanner := bufio.NewScanner(strings.NewReader(raw))
	var current *Version

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 && len(parts[0]) == 40 {
			if current != nil {
				versions = append(versions, *current)
			}
			ts, _ := time.Parse(time.RFC3339, parts[2])
			current = &Version{
				CommitHash: parts[0],
				ShortHash:  parts[1],
				Timestamp:  ts,
				Message:    parts[3],
			}
			continue
		}

		if current != nil {
			fields := strings.Split(line, "\t")
			if len(fields) >= 3 {
				ins, _ := strconv.Atoi(fields[0])
				del, _ := strconv.Atoi(fields[1])
				current.Insertions += ins
				current.Deletions += del
			}
		}
	}
	if current != nil {
		versions = append(versions, *current)
	}
	return versions
}
