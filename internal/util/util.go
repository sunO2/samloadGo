package util

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"
)

const DEFAULT_CHUNK_SIZE = 4096 // 4KB, Kotlin code uses 256KB for download, but 4KB for CRC32 check. Let's start with 4KB.

// GlobalHttpClient is a shared HTTP client with infinite timeout.
var GlobalHttpClient = &http.Client{
	Timeout: 0, // No timeout
}

// XMLNode represents a generic XML element for parsing.
type XMLNode struct {
	XMLName  xml.Name
	Attr     []xml.Attr `xml:",attr"` // Add this line to parse attributes
	Content  []byte     `xml:",innerxml"`
	Children []XMLNode  `xml:",any"`
}

// FirstElementByTagName finds the first child element with the given tag name.
func FirstElementByTagName(node *XMLNode, tagName string) *XMLNode {
	for i := range node.Children {
		if strings.EqualFold(node.Children[i].XMLName.Local, tagName) {
			return &node.Children[i]
		}
	}
	return nil
}

// Text returns the inner text of the node.
func (n *XMLNode) Text() string {
	return strings.TrimSpace(string(n.Content))
}

// TrackOperationProgress is a placeholder for progress tracking.
// In a real application, this would update a UI or log progress.
func TrackOperationProgress(
	size int64,
	progressCallback func(current, max, bps int64),
	operation func() (int64, error),
	progressOffset int64,
	condition func() bool,
	throttle bool,
) error {
	var current int64 = progressOffset
	startTime := time.Now()

	for condition() {
		n, err := operation()
		if err != nil {
			return err
		}
		current += n

		// Calculate bytes per second (bps)
		elapsed := time.Since(startTime).Seconds()
		var bps int64
		if elapsed > 0 {
			bps = int64(float64(current) / elapsed)
		}

		progressCallback(current, size, bps)

		if n == 0 { // No more data read, break to prevent infinite loop
			break
		}
	}
	return nil
}
