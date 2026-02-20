package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/futureCreator/vcoding/internal/run"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cost and run statistics",
	RunE:  runStats,
}

func runStats(cmd *cobra.Command, args []string) error {
	runsDir := filepath.Join(".vcoding", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No runs found.")
			return nil
		}
		return fmt.Errorf("reading runs dir: %w", err)
	}

	type runStat struct {
		id   string
		meta run.Meta
	}

	var stats []runStat
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "latest" {
			continue
		}
		metaPath := filepath.Join(runsDir, e.Name(), "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta run.Meta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		stats = append(stats, runStat{id: e.Name(), meta: meta})
	}

	if len(stats) == 0 {
		fmt.Println("No runs found.")
		return nil
	}

	// Sort by started_at descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].meta.StartedAt.After(stats[j].meta.StartedAt)
	})

	var totalCost float64
	var completed, failed int
	for _, s := range stats {
		totalCost += s.meta.TotalCost
		switch s.meta.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	fmt.Printf("Runs: %d total, %d completed, %d failed\n", len(stats), completed, failed)
	fmt.Printf("Total cost: $%.4f\n", totalCost)
	if len(stats) > 0 {
		fmt.Printf("Average cost: $%.4f\n", totalCost/float64(len(stats)))
	}
	fmt.Println()
	fmt.Printf("%-40s %-10s %-12s %s\n", "Run ID", "Status", "Cost", "Mode")
	fmt.Println(string(make([]byte, 70)))
	for _, s := range stats {
		fmt.Printf("%-40s %-10s $%-11.4f %s\n",
			s.id, s.meta.Status, s.meta.TotalCost, s.meta.InputMode)
	}
	return nil
}
