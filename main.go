package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"time"
)

func main() {
	// load settings
	settings := readSettings()
	// per folder listen to files
	for _, setting := range settings {
		// listen to files
		rawFiles, errors := listenToFiles(setting.SourcePath, setting.SourceRegex)
		go func(errs <-chan error) {
			for e := range errs {
				log.Println(e)
			}
		}(errors)
		// lock files
		lockedPath := setting.SourcePath + setting.Name + `/`
		os.Mkdir(lockedPath, os.ModePerm)
		errors = lockFiles(lockedPath, rawFiles)
		go func(errs <-chan error) {
			for e := range errs {
				log.Println(e)
			}
		}(errors)
		// listen to locked files
		lockedFiles, errors := listenToFiles(lockedPath, `.*`)
		// upload files
		for f := range lockedFiles {
			log.Println(f)
		}
	}

}

type setting struct {
	Name        string `json:"name"`
	SourcePath  string `json:"source_path"`
	SourceRegex string `json:"source_regex"`
	Destination string `json:"destination"`
}

type File struct {
	Name string
	Path string
	Size int64
}

func readSettings() []setting {
	b, err := ioutil.ReadFile(`settings.json`)
	if err != nil {
		panic(err)
	}
	var output []setting
	err = json.Unmarshal(b, &output)
	if err != nil {
		panic(err)
	}
	return output
}

func listenToFiles(path, rx string) (<-chan File, <-chan error) {
	output := make(chan File)
	errors := make(chan error)
	rgx, err := regexp.Compile(rx)
	if err != nil {
		panic(err)
	}
	go func() {
		ticker := time.NewTicker(time.Millisecond * 100)
		for range ticker.C {

			files, err := ioutil.ReadDir(path)
			if err != nil {
				errors <- err
			}
			for _, file := range files {
				if file.IsDir() || !rgx.MatchString(file.Name()) {
					continue
				}
				output <- File{
					Name: file.Name(),
					Path: path,
					Size: file.Size(),
				}
			}
		}
	}()
	return output, errors
}

func lockFiles(lockedPath string, files <-chan File) <-chan error {
	errors := make(chan error)
	go func() {
		for f := range files {
			if err := os.Rename(f.Path+f.Name, lockedPath+f.Name); err != nil {
				errors <- err
			}
		}
	}()
	return errors
}
