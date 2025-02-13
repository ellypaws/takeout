package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

	fmt.Printf("Updated file times of %s to %s\n", imagePath, takenTime.Format(time.RFC3339))
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
