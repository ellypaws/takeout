//go:build windows
// +build windows

package main

import (
	"encoding/json"
	"errors"
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

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/sqweek/dialog"
)

// processJSON reads the metadata JSON file, extracts the photoTakenTime,
// and updates the corresponding image file's modification, access, and creation times.
func processJSON(jsonPath string) {
	file, err := os.Open(jsonPath)
	if err != nil {
		color.Red("Error reading JSON file %s: %v\n", jsonPath, err)
		return
	}
	defer file.Close()

	var meta Takeout
	if err := json.NewDecoder(file).Decode(&meta); err != nil {
		color.Red("Error parsing JSON file %s: %v\n", jsonPath, err)
		return
	}

	ts, err := strconv.ParseInt(meta.PhotoTakenTime.Timestamp, 10, 64)
	if err != nil {
		color.Red("Error parsing timestamp in %s: %v\n", jsonPath, err)
		return
	}
	takenTime := time.Unix(ts, 0)

	// Determine the image file by using the Title field (assumed to be the image filename)
	imagePath := filepath.Join(filepath.Dir(jsonPath), meta.Title)

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		color.Red("Image file %s does not exist for metadata %s\n", imagePath, jsonPath)
		return
	}

	// Update modification and access times.
	if err := os.Chtimes(imagePath, takenTime, takenTime); err != nil {
		color.Red("Error updating file times for %s: %v\n", imagePath, err)
		return
	}

	// Update creation time (Windows only).
	if err := changeDateCreated(imagePath, takenTime); err != nil {
		color.Red("Error updating creation time for %s: %v\n", imagePath, err)
		return
	}

	color.Green("✓ Updated file times of %s to %s\n", imagePath, takenTime.Format(time.RFC3339))
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
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get the underlying Windows handle.
	handle := syscall.Handle(file.Fd())
	// Convert takenTime to Windows FILETIME.
	ft := timeToFiletime(takenTime)

	// Set the file's creation, last access, and last write times.
	if err := syscall.SetFileTime(handle, &ft, &ft, &ft); err != nil {
		return fmt.Errorf("failed to set creation time: %w", err)
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
			if entry.Name() == "metadata.json" {
				continue
			}
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

	var absStartDir string
	if *startDir == "." {
		startDir, err := filepath.Abs(".")
		if err != nil {
			log.Fatalf("Error determining absolute path: %v\n", err)
		}
		absStartDir, err = dialog.Directory().Title(`Select the root "Google Photos" folder.`).SetStartDir(startDir).Browse()
		if err != nil {
			if errors.Is(err, dialog.ErrCancelled) {
				absStartDir, err = filepath.Abs(".")
				if err != nil {
					log.Fatalf("Error determining absolute path: %v\n", err)
				}
				color.Yellow(`Using current "%s" directory\n`, absStartDir)
			} else {
				log.Fatalf("Error selecting directory: %v\n", err)
			}
		}
	}

	// List folders in absStartDir.
	var folders []string
	entries, err := os.ReadDir(absStartDir)
	if err != nil {
		log.Fatalf("Error reading directory %s: %v\n", absStartDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			folderPath := filepath.Join(absStartDir, entry.Name())
			folders = append(folders, folderPath)
		}
	}

	// Prepare a slice to hold the user's selected folders.
	var selectedFolders []string

	// Build MultiSelect options with all folders checked by default.
	options := make([]huh.Option[string], len(folders))
	for i, folder := range folders {
		options[i] = huh.NewOption(folder, folder).Selected(true)
	}

	// Build the form.
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select folders to process").
				Options(options...).
				Value(&selectedFolders),
		),
	)

	// Run the form to let the user choose which folders to process.
	if err := form.Run(); err != nil {
		log.Fatalf("Error running form: %v", err)
	}

	// Process each selected folder concurrently.
	var wg sync.WaitGroup
	for _, folder := range selectedFolders {
		wg.Add(1)
		go processDir(folder, &wg)
	}
	wg.Wait()

	color.Green("Processing complete!")
}
