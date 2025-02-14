//go:build windows
// +build windows

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// processJSON reads the metadata JSON file, extracts the photoTakenTime,
// and updates the corresponding image file's modification and access times.
func processJSON(jsonPath string) {
	file, err := os.Open(jsonPath)
	if err != nil {
		log.Printf("Error reading JSON file %s: %v\n", jsonPath, err)
		return
	}

	var meta Takeout
	if err := json.NewDecoder(file).Decode(&meta); err != nil {
		log.Printf("Error parsing JSON file %s: %v\n", jsonPath, err)
		return
	}

	ts, err := strconv.ParseInt(meta.PhotoTakenTime.Timestamp, 10, 64)
	if err != nil {
		log.Printf("Error parsing timestamp in %s: %v\n", jsonPath, err)
		return
	}
	takenTime := time.Unix(ts, 0)

	// Remove the .json extension to determine the image file's path.
	imagePath := meta.Title
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		log.Printf("Image file %s does not exist for metadata %s\n", imagePath, jsonPath)
		return
	}

	// Update the file's modification and access times.
	if err := os.Chtimes(imagePath, takenTime, takenTime); err != nil {
		log.Printf("Error updating file times for %s: %v\n", imagePath, err)
		return
	}

	if err := changeDateCreated(imagePath, takenTime); err != nil {
		log.Printf("Error updating file times for %s: %v\n", imagePath, err)
		return
	}

	log.Printf("Updated file times of %s to %s\n", imagePath, takenTime.Format(time.RFC3339))
}

// timeToFiletime converts a time.Time to a Windows FILETIME structure.
// Windows FILETIME counts 100-nanosecond intervals since January 1, 1601.
func timeToFiletime(t time.Time) syscall.Filetime {
	const ticksPerSecond = 10000000     // 10^7 100-ns intervals per second
	const epochDifference = 11644473600 // seconds between 1601-01-01 and 1970-01-01
	unixTime := t.Unix()
	nano := t.Nanosecond()
	total := uint64(unixTime+epochDifference)*ticksPerSecond + uint64(nano)/100
	return syscall.Filetime{
		LowDateTime:  uint32(total & 0xFFFFFFFF),
		HighDateTime: uint32(total >> 32),
	}
}

// changeDateCreated changes the creation date of the file.
// On Windows it uses syscall.SetFileTime; on other platforms it returns an error.
func changeDateCreated(imagePath string, takenTime time.Time) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("changeDateCreated is only supported on Windows (current OS: %s)", runtime.GOOS)
	}

	// Open the file with read-write access.
	file, err := os.OpenFile(imagePath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Get the underlying Windows handle.
	handle := syscall.Handle(file.Fd())
	// Convert the takenTime to Windows FILETIME.
	ft := timeToFiletime(takenTime)

	// Set the file's creation time.
	if err := syscall.SetFileTime(handle, &ft, &ft, &ft); err != nil {
		return fmt.Errorf("failed to set creation time: %v", err)
	}

	return nil
}

// processDir walks through the directory specified by dirPath.
// For each subdirectory, it spawns a new goroutine.
// For each JSON file, it calls processJSON to update the corresponding image file.
func processDir(dirPath string, wg *sync.WaitGroup) {
	defer wg.Done()

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		log.Printf("Error reading directory %s: %v\n", dirPath, err)
		return
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			wg.Add(1)
			go processDir(fullPath, wg)
		} else {
			// Only process files ending with .json (assumed to be Google Takeout metadata)
			if strings.HasSuffix(entry.Name(), ".json") {
				processJSON(fullPath)
			}
		}
	}
}

func main() {
	// Optionally allow a different starting directory via command-line flag.
	startDir := flag.String("dir", ".", "Directory to start the recursive walk")
	flag.Parse()

	absStartDir, err := filepath.Abs(*startDir)
	if err != nil {
		log.Printf("Error determining absolute path: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go processDir(absStartDir, &wg)
	wg.Wait()
}
