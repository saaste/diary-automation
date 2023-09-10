package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

type appSettings struct {
	OriginalPhotoPath string        `yaml:"original_photo_path"`
	TargetPhotoPath   string        `yaml:"target_photo_path"`
	ObsidianFilePath  string        `yaml:"obsidian_file_path"`
	CheckInterval     time.Duration `yaml:"check_interval"`
	ImagePrefix       string        `yaml:"image_prefix"`
}

func readSettings() (*appSettings, error) {
	data, err := os.ReadFile("settings.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read settings.yaml: %v", err)
	}

	var appSettings appSettings
	if err := yaml.Unmarshal(data, &appSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings.yaml: %v", err)
	}

	return &appSettings, nil
}

func checkPhotos(photoPath string) map[string][]string {
	result := make(map[string][]string)

	files, err := ioutil.ReadDir(photoPath)
	if err != nil {
		log.Fatalf("unable to read path %s, %s", photoPath, err)
	}

	r, err := regexp.Compile(`^\d{4}-\d{2}-\d{2}(-\d{2})?.(jpg|png)$`)
	if err != nil {
		log.Fatalf("unable to compile regular expression: %s", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			matched := r.MatchString(file.Name())

			if matched {
				date := getDateFromFile(file.Name())
				if _, ok := result[date]; !ok {
					result[date] = make([]string, 0)
				}

				result[date] = append(result[date], path.Join(photoPath, file.Name()))
			}

		}
	}

	return result
}

func getDateFromFile(filePath string) string {
	filename := path.Base(filePath)
	return filename[0:10]
}

func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func updateDiaryDocument(date string, photoPaths []string, settings *appSettings) {
	diaryFile := fmt.Sprintf("%s.md", date)
	diaryFilePath := path.Join(settings.ObsidianFilePath, diaryFile)
	content := ""
	photoLinks := ""

	for _, photoPath := range photoPaths {
		filename := path.Base(photoPath)
		photoLinks = photoLinks + fmt.Sprintf("![[%s]]\n", settings.ImagePrefix+filename)
	}

	if fileExists(diaryFilePath) {
		content = fmt.Sprintf("\n\n### Iltakirjoitus\n%s", photoLinks)
	} else {
		content = fmt.Sprintf("# %s\n\n### Iltakirjoitus\n%s", date, photoLinks)
	}

	f, err := os.OpenFile(diaryFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("unable to open file %s: %s", diaryFile, err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		log.Fatalf("unable to append text to file: %s", err)
	}
}

func moveImages(photos []string, settings *appSettings) {
	for _, photo := range photos {
		filename := path.Base(photo)
		target := path.Join(settings.TargetPhotoPath, settings.ImagePrefix+filename)
		log.Printf("moving %s to %s\n", photo, target)

		inputFile, err := os.Open(photo)
		if err != nil {
			log.Fatalf("unable to read the input file %s: %s", photo, err)
		}

		outputFile, err := os.Create(target)
		if err != nil {
			log.Fatalf("unable to create the destination file %s: %s", target, err)
		}
		defer outputFile.Close()

		_, err = io.Copy(outputFile, inputFile)
		inputFile.Close()
		if err != nil {
			log.Fatalf("unable to copy image %s to %s: %s", photo, target, err)
		}

		err = os.Remove(photo)
		if err != nil {
			log.Fatalf("unable to delete the input file %s: %s", photo, err)
		}
	}
}

func main() {
	settings, err := readSettings()
	if err != nil {
		log.Fatalf("unable to read setting: %s", err)
	}
	tick := time.Tick(settings.CheckInterval)
	for range tick {
		log.Printf("checking photos from %s\n", settings.OriginalPhotoPath)
		photos := checkPhotos(settings.OriginalPhotoPath)
		for date, photos := range photos {
			log.Printf("updating diary for %s with %d photos\n", date, len(photos))
			updateDiaryDocument(date, photos, settings)
			moveImages(photos, settings)
		}
		time.Sleep(time.Second * time.Duration(settings.CheckInterval))
	}
}
