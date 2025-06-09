package cmd

import (
	"fmt"
	"strings" // Added for strings.TrimSpace and strings.ToLower

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Interactive mode for selecting operations",
	Long:  `This command provides an interactive interface to select and execute various operations.`,
	Run: func(cmd *cobra.Command, args []string) {
		for {
			fmt.Println("\n请选择一个操作:")
			fmt.Println("A: 版本检查")
			fmt.Println("B: 固件下载")
			fmt.Println("C: 固件解码")
			fmt.Println("!q: 退出")
			fmt.Print("请输入您的选择: ")

			var choice string
			fmt.Scanln(&choice)
			choice = strings.TrimSpace(strings.ToLower(choice))

			switch choice {
			case "a":
				runCheckCommand()
			case "b":
				runDownloadCommand()
			case "c":
				runDecryptCommand()
			case "!q":
				fmt.Println("退出程序。")
				return
			default:
				fmt.Println("无效的选择，请重新输入。")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// The flags are defined in rootCmd and are persistent, so they are available here.
}

func runCheckCommand() {
	if model == "" {
		fmt.Print("请输入设备型号 (例如: SM-G998U): ")
		fmt.Scanln(&model)
	} else {
		fmt.Printf("设备型号 (已提供): %s\n", model)
	}
	if region == "" {
		fmt.Print("请输入设备地区 (例如: XAA): ")
		fmt.Scanln(&region)
	} else {
		fmt.Printf("设备地区 (已提供): %s\n", region)
	}

	// Get the firmware version from checkLatestVersion and store it globally
	fwVersion = checkLatestVersion(model, region)
	if fwVersion != "" {
		fmt.Printf("已获取固件版本: %s，可用于后续操作。\n", fwVersion)
	}
}

func runDownloadCommand() {
	if model == "" {
		fmt.Print("请输入设备型号 (例如: SM-G998U): ")
		fmt.Scanln(&model)
	} else {
		fmt.Printf("设备型号 (已提供): %s\n", model)
	}
	if region == "" {
		fmt.Print("请输入设备地区 (例如: XAA): ")
		fmt.Scanln(&region)
	} else {
		fmt.Printf("设备地区 (已提供): %s\n", region)
	}
	if fwVersion == "" {
		fmt.Print("请输入固件版本 (例如: G998USQU4AUF5/G998UOYN4AUF5/G998USQU4AUF5/G998USQU4AUF5): ")
		fmt.Scanln(&fwVersion)
	} else {
		fmt.Printf("固件版本 (已提供): %s\n", fwVersion)
	}
	if imeiSerial == "" {
		fmt.Print("请输入设备IMEI或序列号 (多个请用分号或换行符分隔): ")
		fmt.Scanln(&imeiSerial)
	} else {
		fmt.Printf("设备IMEI或序列号 (已提供): %s\n", imeiSerial)
	}
	var input string
	if outputFile == "" {
		fmt.Print("请输入输出文件路径 (例如: firmware.zip): ")
	} else {
		fmt.Printf("请输入输出文件路径 (当前: %s，回车保留): ", outputFile)
	}
	fmt.Scanln(&input)
	if input != "" {
		outputFile = input
	}

	// Simulate running the download command
	DownloadCmd.Run(DownloadCmd, []string{})
}

func runDecryptCommand() {
	var input string
	if inputFile == "" {
		fmt.Print("请输入输入文件路径 (例如: encrypted.zip.enc4): ")
	} else {
		fmt.Printf("请输入输入文件路径 (当前: %s，回车保留): ", inputFile)
	}
	fmt.Scanln(&input)
	if input != "" {
		inputFile = input
	}

	if outputFile == "" {
		fmt.Print("请输入输出文件路径 (例如: decrypted.zip): ")
	} else {
		fmt.Printf("请输入输出文件路径 (当前: %s，回车保留): ", outputFile)
	}
	fmt.Scanln(&input)
	if input != "" {
		outputFile = input
	}
	if fwVersion == "" {
		fmt.Print("请输入固件版本 (例如: G998USQU4AUF5/G998UOYN4AUF5/G998USQU4AUF5/G998USQU4AUF5): ")
		fmt.Scanln(&fwVersion)
	} else {
		fmt.Printf("固件版本 (已提供): %s\n", fwVersion)
	}
	if model == "" {
		fmt.Print("请输入设备型号 (例如: SM-G998U): ")
		fmt.Scanln(&model)
	} else {
		fmt.Printf("设备型号 (已提供): %s\n", model)
	}
	if region == "" {
		fmt.Print("请输入设备地区 (例如: XAA): ")
		fmt.Scanln(&region)
	} else {
		fmt.Printf("设备地区 (已提供): %s\n", region)
	}
	if imeiSerial == "" {
		fmt.Print("请输入设备IMEI或序列号 (多个请用分号或换行符分隔): ")
		fmt.Scanln(&imeiSerial)
	} else {
		fmt.Printf("设备IMEI或序列号 (已提供): %s\n", imeiSerial)
	}

	// Simulate running the decrypt command
	DecryptCmd.Run(DecryptCmd, []string{})
}
