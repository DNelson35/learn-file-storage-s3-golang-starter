package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"os"
)

func processVideoForFastStart(filePath string) (string, error){
	processedFilePath := fmt.Sprintf("%s.processing", filePath)
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", processedFilePath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error processing video: %s, %v", stderr.String(), err)
	}

	fileInfo, err := os.Stat(processedFilePath)
	if err != nil {
		return "", err
	}
	if fileInfo.Size() == 0 {
		return "", fmt.Errorf("processed file is empty")
	}
	return processedFilePath, nil
}