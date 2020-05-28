package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckEncoding confirms that the encoding of a file matches *.dca. If it doesn't, it will
// convert it using ffmpeg and dca.exe
func CheckEncoding(pathToFile string, dcaFolder string) string {
	splitStr := SplitByNonWord(pathToFile)
	ext := splitStr[len(splitStr) - 1]
	filename := splitStr[len(splitStr) - 2]

	if ext != "dca" {
		fmt.Println("Converting to DCA")
		ConvertToDCA(pathToFile, dcaFolder)
		return fmt.Sprintf("./sounds/%s.dca", filename)
	} else {
		return pathToFile
	}
}

// ConvertToDCA will convert a file to DCA format for consumption by discord
func ConvertToDCA(pathToFile string, dcaFolder string) string {
	splitStr := SplitByNonWord(pathToFile)
	filename := splitStr[len(splitStr) - 2]

	cmd := fmt.Sprintf("ffmpeg -i %s -f s16le -ar 48000 -ac 2 pipe:1 | ./dca > %s/%s.dca",
		pathToFile,
		dcaFolder,
		filename)
	cmd = strings.Replace(cmd, "\\", "/", -1)
	_, err  := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		panic(err)
		return "ERROR"
	}
	return fmt.Sprintf("%s/%s.dca", dcaFolder, filename)
}

// SplitByNonWord is a simple helper to split a string by any non-word character
func SplitByNonWord(toSplit string) []string {
	pattern := regexp.MustCompile(`\W`)
	return pattern.Split(toSplit, -1)
}

// GetAllFilesInDir will simply return a string array of all the files in a directory, excluding
// directories, but will return files in subdirectories
func GetAllFilesInDir(pathToDir string) []string {
	var files []string
	err := filepath.Walk(pathToDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return files
}

// DownloadFile simply downloads from a url and saves locally
// credit: https://golangcode.com/download-a-file-from-a-url/
func DownloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
