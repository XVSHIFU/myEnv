package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"myenv/internal/core"
)

// JSON is streamed as one document, retaining bounded memory even for a large
// history. A core failure closes the document with partial progress and exit 1.
func cleanCommand(directory *string, jsonOutput *bool, userConfigDirectory string, languages ...locale) *cobra.Command {
	var lang locale
	if len(languages) > 0 {
		lang = languages[0]
	}
	var dryRun bool
	var cache string
	cmd := &cobra.Command{Use: "clean", Short: "Remove unused managed project generations", Long: "Remove unused managed project generations, preserving current, previous, leased and preparing generations.\n--dry-run lists candidates and logical file bytes without changing managed state.\nFailed preparation records and their generation/download directories are also cleaned. Windows clean can recover identified leases after supervisor and Job exit; unknown and legacy leases remain protected. Use clean --cache node to remove shared Node transport archives; Use clean --cache uv to delegate cache cleanup to the installed pinned uv, without --force.", Args: cobra.NoArgs}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "List candidates without deleting")
	cmd.Long += "\nInterrupted preparations with an explicit completed-child-phase record can also be recovered. Preview is advisory; actual cleanup rechecks under the workspace lock."
	cmd.Flags().StringVar(&cache, "cache", "", "Clean a shared cache instead of project generations (node or uv)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if cache != "" && cache != "node" && cache != "uv" {
			return fmt.Errorf("supported cache selections: node, uv")
		}
		out := cmd.OutOrStdout()
		first := true
		if *jsonOutput {
			if _, err := io.WriteString(out, `{"schema":1,"data":{"items":[`); err != nil {
				return &childExit{code: 1}
			}
		}
		report := func(item core.CleanItem) error {
			if !*jsonOutput {
				_, err := fmt.Fprintf(out, lang.text("%s\t%d bytes\t%s\n"), item.ID, item.Bytes, item.Directory)
				return err
			}
			if !first {
				if _, err := io.WriteString(out, ","); err != nil {
					return err
				}
			}
			first = false
			return json.NewEncoder(out).Encode(item)
		}
		var summary core.CleanResult
		var err error
		if cache != "" {
			var service *core.Service
			service, err = runtimeService(false, userConfigDirectory)
			if err == nil {
				defer closeRuntimeService(service)
				if cache == "node" {
					summary, err = service.CleanNodeCache(cmd.Context(), dryRun, report)
				} else {
					summary, err = service.CleanUVCache(cmd.Context(), dryRun, report)
				}
			}
		} else {
			summary, err = (&core.Service{}).Clean(cmd.Context(), *directory, dryRun, report)
		}
		var failureInfo *failure
		exitCode := 1
		if err != nil {
			failureInfo = &failure{Code: "CLEAN_FAILED", Message: err.Error(), NextAction: "Check the reported cause and retry myenv clean --dry-run."}
			if cache != "" {
				failureInfo.NextAction = "Check the reported cause and retry myenv clean --cache " + cache + " --dry-run."
			}
			var environmentErr *environmentConfigError
			if errors.As(err, &environmentErr) {
				failureInfo.NextAction = environmentErr.nextAction()
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				failureInfo.Code = "CANCELED"
				exitCode = 130
			}
		}
		if *jsonOutput {
			if _, e := io.WriteString(out, `],"summary":`); e != nil {
				return &childExit{code: 1}
			}
			if e := json.NewEncoder(out).Encode(summary); e != nil {
				return &childExit{code: 1}
			}
			tail, _ := json.Marshal(struct {
				OK      bool     `json:"ok"`
				Changed bool     `json:"changed"`
				Error   *failure `json:"error"`
			}{err == nil, summary.Changed, failureInfo})
			if _, e := fmt.Fprintf(out, "},%s\n", tail[1:]); e != nil {
				return &childExit{code: 1}
			}
		} else {
			if _, e := fmt.Fprintf(out, lang.text("Candidates: %d; removed: %d; logical bytes: %d\nRecoverable leases: %d; recovered: %d; unknown: %d\n"), summary.Candidates, summary.Removed, summary.Bytes, summary.RecoverableLeases, summary.RecoveredLeases, summary.UnknownLeases); e != nil {
				return &childExit{code: 1}
			}
			if failureInfo != nil {
				lang.failure(cmd.ErrOrStderr(), failureInfo)
			}
		}
		if err != nil {
			return &childExit{code: exitCode}
		}
		return nil
	}
	return cmd
}
