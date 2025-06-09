package cmd

import (
	"fmt"
	"os"

	"samsung-firmware-tool/internal/versionfetch"

	"github.com/spf13/cobra"
)

// CheckCmd represents the check command
var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for the latest firmware version",
	Long:  `This command checks for the latest firmware version for a given device model and region.`,
	Run: func(cmd *cobra.Command, args []string) {
		if model == "" || region == "" {
			fmt.Println("错误: --model 和 --region 是必需的。")
			os.Exit(1)
		}
		checkLatestVersion(model, region)
	},
}

func init() {
	rootCmd.AddCommand(CheckCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// checkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// checkCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func checkLatestVersion(model, region string) string {
	fmt.Printf("Checking latest version for Model: %s, Region: %s\n", model, region)
	result := versionfetch.GetLatestVersion(model, region)

	if result.Error != nil {
		fmt.Printf("Error checking version: %v\n", result.Error)
		fmt.Printf("Raw output: %s\n", result.RawOutput)
		return "" // Return empty string on error
	}

	fmt.Printf("Latest Firmware Version: %s\n", result.VersionCode)
	fmt.Printf("Android Version: %s\n", result.AndroidVersion)
	return result.VersionCode // Return the firmware version
}
