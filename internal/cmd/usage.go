package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/liasica/cpm/internal/claude"
	"github.com/liasica/cpm/internal/cookie"
	"github.com/liasica/cpm/internal/i18n"
	"github.com/liasica/cpm/internal/profile"
	"github.com/spf13/cobra"
)

// Claude API 响应结构
type organization struct {
	UUID          string   `json:"uuid"`
	Name          string   `json:"name"`
	RateLimitTier string   `json:"rate_limit_tier"`
	BillingType   string   `json:"billing_type"`
	Capabilities  []string `json:"capabilities"`
	RavenType     string   `json:"raven_type"`
}

type usageResponse struct {
	FiveHour       *usageWindow `json:"five_hour"`
	SevenDay       *usageWindow `json:"seven_day"`
	SevenDayOpus   *usageWindow `json:"seven_day_opus"`
	SevenDaySonnet *usageWindow `json:"seven_day_sonnet"`
	SevenDayCowork *usageWindow `json:"seven_day_cowork"`
	ExtraUsage     *extraUsage  `json:"extra_usage"`
}

type usageWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    *string `json:"resets_at"`
}

type extraUsage struct {
	IsEnabled    bool     `json:"is_enabled"`
	MonthlyLimit *float64 `json:"monthly_limit"`
	UsedCredits  *float64 `json:"used_credits"`
	Utilization  *float64 `json:"utilization"`
}

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: i18n.T("Show rate-limit usage and reset times for all accounts", "查询所有账户的用量和重置时间"),
	Run: func(cmd *cobra.Command, args []string) {
		pm, err := profile.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		names, err := pm.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// 如果没有 profile，尝试读取当前 Claude 数据目录
		if len(names) == 0 {
			fmt.Println("No profiles found, checking current Claude session...")
			var claudeDir string
			claudeDir, err = claude.DataDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			printUsageForDir("(current)", claudeDir)
			return
		}

		current := pm.Current()
		for _, name := range names {
			label := name
			if name == current {
				label = name + " *"
			}

			profDir := pm.ProfileDir(name)

			// 当前 profile 使用 Claude 数据目录中的最新 cookies
			if name == current {
				var claudeDir string
				claudeDir, err = claude.DataDir()
				if err == nil {
					printUsageForDir(label, claudeDir)
					continue
				}
			}

			printUsageForDir(label, profDir)
		}
	},
}

func printUsageForDir(label, dir string) {
	fmt.Printf("\n── %s ──\n", label)

	cookies, err := cookie.ReadFromProfile(dir)
	if err != nil {
		fmt.Printf("  Failed to read cookies: %v\n", err)
		return
	}

	if len(cookies) == 0 {
		fmt.Println("  No cookies found")
		return
	}

	// 获取组织信息
	orgs, err := fetchOrganizations(cookies)
	if err != nil {
		fmt.Printf("  Failed to fetch organizations: %v\n", err)
		return
	}

	if len(orgs) == 0 {
		fmt.Println("  No organizations found")
		return
	}

	for _, org := range orgs {
		fmt.Printf("  Organization: %s", org.Name)
		if org.RateLimitTier != "" {
			fmt.Printf(" (%s)", org.RateLimitTier)
		}
		fmt.Println()

		// 获取用量信息
		usage, err := fetchUsage(cookies, org.UUID)
		if err != nil {
			fmt.Printf("    Usage: failed to fetch (%v)\n", err)
			continue
		}

		printUsage(usage)
	}
}

func printUsage(u *usageResponse) {
	if u == nil {
		fmt.Println("    No usage data")
		return
	}

	printWindow("5-hour", u.FiveHour)
	printWindow("7-day", u.SevenDay)
	printWindow("7-day Opus", u.SevenDayOpus)
	printWindow("7-day Sonnet", u.SevenDaySonnet)
	printWindow("7-day Cowork", u.SevenDayCowork)

	if u.ExtraUsage != nil && u.ExtraUsage.IsEnabled {
		fmt.Print("    Extra usage: enabled")
		if u.ExtraUsage.UsedCredits != nil && u.ExtraUsage.MonthlyLimit != nil {
			fmt.Printf(" ($%.2f / $%.2f)", *u.ExtraUsage.UsedCredits, *u.ExtraUsage.MonthlyLimit)
		}
		fmt.Println()
	}
}

func printWindow(name string, w *usageWindow) {
	if w == nil {
		return
	}

	pct := w.Utilization * 100
	fmt.Printf("    %-14s %6.1f%%", name+":", pct)

	if w.ResetsAt != nil {
		t, err := time.Parse(time.RFC3339, *w.ResetsAt)
		if err == nil {
			remaining := time.Until(t)
			if remaining > 0 {
				fmt.Printf("  resets %s (%s)", t.Local().Format("01-02 15:04"), formatDuration(remaining))
			} else {
				fmt.Print("  (reset)")
			}
		}
	}

	fmt.Println()
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}

func fetchOrganizations(cookies []*http.Cookie) ([]organization, error) {
	body, err := apiGet("https://claude.ai/api/organizations", cookies)
	if err != nil {
		return nil, err
	}

	var orgs []organization
	if err = json.Unmarshal(body, &orgs); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return orgs, nil
}

func fetchUsage(cookies []*http.Cookie, orgUUID string) (*usageResponse, error) {
	body, err := apiGet(fmt.Sprintf("https://claude.ai/api/organizations/%s/usage", orgUUID), cookies)
	if err != nil {
		return nil, err
	}

	var usage usageResponse
	if err = json.Unmarshal(body, &usage); err != nil {
		return nil, fmt.Errorf("failed to parse usage response: %w", err)
	}

	return &usage, nil
}

func apiGet(url string, cookies []*http.Cookie) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 手动构建 Cookie header 避免 Go 对特殊字符的校验
	var parts []string
	for _, c := range cookies {
		parts = append(parts, c.Name+"="+c.Value)
	}
	req.Header.Set("Cookie", strings.Join(parts, "; "))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://claude.ai/")
	req.Header.Set("Origin", "https://claude.ai")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func init() {
	rootCmd.AddCommand(usageCmd)
}
