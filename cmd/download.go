package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"samsung-firmware-tool/internal/fusclient"
	"samsung-firmware-tool/internal/request"

	"github.com/spf13/cobra"
)

// DownloadStatus defines the current status of a download task.
type DownloadStatus int

const (
	StatusIdle DownloadStatus = iota
	StatusInitializing
	StatusDownloading
	StatusPaused
	StatusCompleted
	StatusFailed
)

func (s DownloadStatus) String() string {
	switch s {
	case StatusIdle:
		return "Idle"
	case StatusInitializing:
		return "Initializing"
	case StatusDownloading:
		return "Downloading"
	case StatusPaused:
		return "Paused"
	case StatusCompleted:
		return "Completed"
	case StatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// ProgressCallback defines the function signature for progress updates.
type ProgressCallback = func(current, max, bps int64)

// DownloadTask represents a single firmware download operation.
type DownloadTask struct {
	Model      string
	Region     string
	FwVersion  string
	ImeiSerial string
	OutputPath string
	FileName   string

	Status         DownloadStatus
	Progress       float64 // Percentage
	CurrentSize    int64   // Bytes downloaded so far
	TotalSize      int64   // Total bytes to download
	BytesPerSecond int64   // Download speed

	client     *fusclient.FusClient
	binaryInfo *request.BinaryFileInfo
	outputFile *os.File
	progressMu sync.Mutex // Mutex to protect progress updates
	pauseMu    sync.Mutex // Mutex to protect pause/resume state
	paused     bool
	cond       *sync.Cond // Condition variable for pausing/resuming

	cancelCtx  context.Context    // Context for cancelling the download
	cancelFunc context.CancelFunc // Function to cancel the context

	// Callbacks
	OnProgress ProgressCallback
	OnFinish   func(msg string)
	OnError    func(err error)
}

// NewDownloadTask creates and initializes a new DownloadTask.
func NewDownloadTask(model, region, fwVersion, imeiSerial, outputPath string, onProgress ProgressCallback) *DownloadTask {
	dt := &DownloadTask{
		Model:      model,
		Region:     region,
		FwVersion:  fwVersion,
		ImeiSerial: imeiSerial,
		OutputPath: outputPath,
		Status:     StatusIdle,
		OnProgress: onProgress,
		client:     fusclient.NewFusClient(),
		OnFinish: func(msg string) {
			fmt.Println(msg)
		},
		OnError: func(err error) {
			fmt.Printf("Error: %v\n", err)
		},
	}
	dt.cond = sync.NewCond(&dt.pauseMu)
	dt.cancelCtx, dt.cancelFunc = context.WithCancel(context.Background()) // Initialize context
	return dt
}

// Start initiates the download process, supporting resume.
func (dt *DownloadTask) Start() error {
	dt.Status = StatusInitializing
	dt.updateProgress()

	fmt.Printf("Initializing download for firmware %s for Model: %s, Region: %s to %s\n", dt.FwVersion, dt.Model, dt.Region, dt.OutputPath)

	onVersionException := func(err error, info *request.BinaryFileInfo) {
		fmt.Printf("Version exception: %v\n", err)
		if info != nil {
			fmt.Println("Attempting to proceed with download despite version exception...")
			dt.binaryInfo = info
			dt.performDownload(dt.cancelCtx) // Proceed with download, pass context
		}
	}
	shouldReportError := func(err error) bool {
		return true // For now, always report
	}

	binaryInfo := request.RetrieveBinaryFileInfo(dt.FwVersion, dt.Model, dt.Region, dt.ImeiSerial, dt.client, dt.OnFinish, onVersionException, shouldReportError)
	if binaryInfo == nil {
		dt.Status = StatusFailed
		dt.OnError(errors.New("failed to retrieve binary file information"))
		return errors.New("failed to retrieve binary file information")
	}
	dt.binaryInfo = binaryInfo
	dt.FileName = binaryInfo.FileName
	dt.TotalSize = binaryInfo.Size

	return dt.performDownload(dt.cancelCtx) // Pass the context
}

// performDownload handles the actual file download logic, including resume.
func (dt *DownloadTask) performDownload(ctx context.Context) error {
	fullPath := filepath.Join(dt.OutputPath, dt.FileName)

	// Check for existing file to resume download
	fileInfo, err := os.Stat(fullPath)
	if err == nil {
		// File exists, check if it's a partial download
		dt.CurrentSize = fileInfo.Size()
		if dt.CurrentSize >= dt.TotalSize {
			dt.Status = StatusCompleted
			dt.OnFinish(fmt.Sprintf("File already downloaded: %s", fullPath))
			return nil
		}
		fmt.Printf("Resuming download from %d bytes.\n", dt.CurrentSize)
		dt.outputFile, err = os.OpenFile(fullPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			dt.Status = StatusFailed
			dt.OnError(fmt.Errorf("error opening file for resume: %w", err))
			return err
		}
	} else if os.IsNotExist(err) {
		// File does not exist, start new download
		dt.CurrentSize = 0
		dt.outputFile, err = os.Create(fullPath)
		if err != nil {
			dt.Status = StatusFailed
			dt.OnError(fmt.Errorf("error creating output file: %w", err))
			return err
		}
	} else {
		dt.Status = StatusFailed
		dt.OnError(fmt.Errorf("error checking file status: %w", err))
		return err
	}
	defer dt.outputFile.Close()

	// Perform BINARY_INIT request
	nonce, err := dt.client.GetNonce()
	if err != nil {
		dt.Status = StatusFailed
		dt.OnError(fmt.Errorf("error getting nonce for BINARY_INIT: %w", err))
		return err
	}
	binaryInitRequest := request.CreateBinaryInit(dt.binaryInfo.FileName, nonce)
	_, err = dt.client.MakeReq(fusclient.BinaryInit, binaryInitRequest, true)
	if err != nil {
		dt.Status = StatusFailed
		dt.OnError(fmt.Errorf("error performing BINARY_INIT request: %w", err))
		return err
	}

	dt.Status = StatusDownloading
	dt.updateProgress()

	// Wrap the original progress callback to include pause/resume logic
	wrappedProgressCallback := func(current, max, bps int64) {
		dt.pauseMu.Lock()
		for dt.paused {
			dt.Status = StatusPaused
			dt.updateProgress() // Update status to paused
			dt.cond.Wait()
		}
		dt.Status = StatusDownloading // Resume status
		dt.pauseMu.Unlock()
		dt.updateProgressCallback()(current, max, bps) // Call original progress update
	}

	md5Sum, err := dt.client.DownloadFile(
		ctx, // Pass the context here
		dt.binaryInfo.Path+dt.binaryInfo.FileName,
		dt.CurrentSize, // Pass current size for resume
		dt.binaryInfo.Size,
		dt.outputFile,
		dt.CurrentSize, // outputSize for progress tracking
		wrappedProgressCallback,
	)
	if err != nil {
		if err == context.Canceled {
			fmt.Println("\nDownload cancelled by user (paused).")
			return nil // Return nil error as it's a controlled pause
		}
		if err == io.EOF {
			// io.EOF means download finished successfully
			dt.Status = StatusCompleted
			dt.OnFinish(fmt.Sprintf("\nDownload complete. MD5: %s", md5Sum))
			return nil
		}
		dt.Status = StatusFailed
		dt.OnError(fmt.Errorf("\nError downloading file: %w", err))
		return err
	}

	dt.Status = StatusCompleted
	dt.OnFinish(fmt.Sprintf("\nDownload complete. MD5: %s", md5Sum))
	return nil
}

// updateProgressCallback returns a progress callback function that updates the task's state.
func (dt *DownloadTask) updateProgressCallback() ProgressCallback {
	return func(current, max, bps int64) {
		dt.progressMu.Lock()
		defer dt.progressMu.Unlock()
		dt.CurrentSize = current
		dt.TotalSize = max
		dt.BytesPerSecond = bps
		if max > 0 {
			dt.Progress = float64(current) / float64(max) * 100
		} else {
			dt.Progress = 0
		}
		if dt.OnProgress != nil {
			dt.OnProgress(dt.CurrentSize, dt.TotalSize, dt.BytesPerSecond)
		}
	}
}

// DownloadCmd represents the download command
var DownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download firmware",
	Long:  `This command downloads firmware for a given device model, region, firmware version, and IMEI/Serial number.`,
	Run: func(cmd *cobra.Command, args []string) {
		if model == "" || region == "" || fwVersion == "" || imeiSerial == "" || outputFile == "" {
			fmt.Println("错误: --model, --region, --fw, --imei, 和 --output 是下载固件所必需的。")
			os.Exit(1)
		}
		var progressCall = func(current, max, bps int64) {
			fmt.Printf("\rDownloading: %d/%d bytes (%.2f%%) @ %d B/s", current, max, float64(current)/float64(max)*100, bps)
		}

		task := NewDownloadTask(model, region, fwVersion, imeiSerial, outputFile, progressCall)
		err := task.Start()
		if err != nil {
			fmt.Printf("Download task failed: %v\n", err)
			os.Exit(1)
		}
	},
}

// Pause pauses the download task.
func (dt *DownloadTask) Pause() {
	dt.pauseMu.Lock()
	defer dt.pauseMu.Unlock()
	if dt.Status == StatusDownloading {
		dt.paused = true
		dt.Status = StatusPaused
		dt.updateProgress()
		dt.cancelFunc() // Cancel the context to stop the HTTP request
		fmt.Println("Download paused.")
	}
}

// Resume resumes a paused download task.
func (dt *DownloadTask) Resume() {
	dt.pauseMu.Lock()
	defer dt.pauseMu.Unlock()
	if dt.Status == StatusPaused {
		dt.paused = false
		dt.Status = StatusDownloading
		dt.updateProgress()
		// Recreate context for new request
		dt.cancelCtx, dt.cancelFunc = context.WithCancel(context.Background())
		go func() {
			// Re-initiate download in a new goroutine
			err := dt.performDownload(dt.cancelCtx)
			if err != nil {
				dt.OnError(err)
			}
		}()
		dt.cond.Signal() // Signal the waiting goroutine to resume (though it might not be waiting anymore if context cancelled)
		fmt.Println("Download resumed.")
	}
}

func init() {
	rootCmd.AddCommand(DownloadCmd)
}

// updateProgress is a helper to call the OnProgress callback.
func (dt *DownloadTask) updateProgress() {
	dt.progressMu.Lock()
	defer dt.progressMu.Unlock()
	if dt.OnProgress != nil {
		dt.OnProgress(dt.CurrentSize, dt.TotalSize, dt.BytesPerSecond)
	}
}
