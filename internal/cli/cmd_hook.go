package cli

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/divijg19/sage/internal/event"
	"github.com/spf13/cobra"
)

var hookRepo string

var hookCmd = &cobra.Command{
	Use:    "hook",
	Hidden: true,
	Short:  "Internal hook entrypoints",
}

var hookPostCommitCmd = &cobra.Command{
	Use:    "post-commit",
	Hidden: true,
	Short:  "Record a git post-commit event",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Hooks must never block git workflows.
		_ = runHookPostCommit(hookRepo)
		return nil
	},
}

func init() {
	hookCmd.PersistentFlags().StringVar(&hookRepo, "repo", "", "path to repo (defaults to current directory)")
	_ = hookCmd.PersistentFlags().MarkHidden("repo")

	hookCmd.AddCommand(hookPostCommitCmd)
	rootCmd.AddCommand(hookCmd)
}

func runHookPostCommit(repo string) error {
	root, err := gitRepoRoot(repo)
	if err != nil {
		return nil
	}
	project := normalizeProjectName(strings.TrimSpace(filepath.Base(root)))
	if project == "" {
		project = "global"
	}

	sha, err := gitOutput(repo, "rev-parse", "HEAD")
	if err != nil {
		return nil
	}
	subject, _ := gitOutput(repo, "show", "-s", "--format=%s", "HEAD")
	body, _ := gitOutput(repo, "show", "-s", "--format=%b", "HEAD")
	authorName, _ := gitOutput(repo, "show", "-s", "--format=%an", "HEAD")
	authorEmail, _ := gitOutput(repo, "show", "-s", "--format=%ae", "HEAD")
	commitTimeRaw, _ := gitOutput(repo, "show", "-s", "--format=%aI", "HEAD")
	branch, _ := gitOutput(repo, "rev-parse", "--abbrev-ref", "HEAD")

	timestamp := time.Now()
	if strings.TrimSpace(commitTimeRaw) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(commitTimeRaw)); err == nil {
			timestamp = t
		}
	}

	repoID := repoHash(root)
	eventID := fmt.Sprintf("git:%s:%s", repoID, sha)

	content := buildCommitContent(sha, branch, strings.TrimSpace(body))
	title := strings.TrimSpace(subject)
	if title == "" {
		title = "(no subject)"
	}

	tags := []string{"git", "commit"}
	_ = ensureTagsConfigured(tags)

	s, err := openGlobalStore()
	if err != nil {
		return nil
	}

	e := event.Event{
		ID:        eventID,
		Timestamp: timestamp,
		Project:   project,
		Kind:      event.CommitKind,
		Title:     title,
		Content:   content,
		Tags:      tags,
		Metadata: map[string]string{
			"repo_root":    root,
			"repo_id":      repoID,
			"sha":          sha,
			"branch":       branch,
			"author_name":  authorName,
			"author_email": authorEmail,
			"commit_time":  commitTimeRaw,
		},
	}

	if err := s.Append(e); err != nil {
		// Ignore duplicates or any failures: never block git.
		return nil
	}

	return nil
}

func repoHash(s string) string {
	h := sha1.Sum([]byte(s))
	// 10 bytes (20 hex chars) is enough to avoid collisions in practice.
	return hex.EncodeToString(h[:10])
}

func buildCommitContent(sha, branch, body string) string {
	lines := []string{}
	if strings.TrimSpace(sha) != "" {
		lines = append(lines, "sha: "+strings.TrimSpace(sha))
	}
	if strings.TrimSpace(branch) != "" {
		lines = append(lines, "branch: "+strings.TrimSpace(branch))
	}

	body = strings.TrimSpace(body)
	if body != "" {
		lines = append(lines, "", body)
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
