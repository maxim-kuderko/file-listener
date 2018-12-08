package main

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var uploading = sync.Map{}

func main() {
	// load settings
	settings := readSettings()
	// per folder listen to files
	for _, setting := range settings {
		// listen to files
		rawFiles, errors := listenToFiles(setting.SourcePath, setting.SourceRegex, 1)
		go printErrors(errors)
		// listen to locked files
		filesToDelete, errors := uploadFiles(setting, rawFiles)
		go printErrors(errors)
		go deleteFiles(filesToDelete)
	}
	done := make(chan bool, 1)
	<-done
}

type setting struct {
	SourcePath        string `json:"source_path"`
	SourceRegex       string `json:"source_regex"`
	Destination       string `json:"destination"`
	DestinationRegion string `json:"destination_region"`
}

type File struct {
	Name        string
	Path        string
	Size        int64
	LastUpdated time.Time
}

func printErrors(errors <-chan error) {
	for e := range errors {
		log.Println(e)
	}
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

func listenToFiles(path, rx string, listenInterval int) (<-chan File, <-chan error) {
	output := make(chan File)
	errors := make(chan error)
	rgx, err := regexp.Compile(rx)
	if err != nil {
		panic(err)
	}
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(listenInterval))
		for range ticker.C {

			files, err := ioutil.ReadDir(path)
			if err != nil {
				errors <- err
			}
			for _, file := range files {
				if file.IsDir() || !rgx.MatchString(file.Name()) || file.Size() < 2 {
					continue
				}
				output <- File{
					Name:        file.Name(),
					Path:        path,
					Size:        file.Size(),
					LastUpdated: file.ModTime(),
				}
			}
		}
	}()
	return output, errors
}

func uploadFiles(st setting, files <-chan File) (<-chan File, <-chan error) {
	output := make(chan File)
	errors := make(chan error)
	go func() {
		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(st.DestinationRegion),
		}))
		upd := s3manager.NewUploader(sess)
		pool, err := strconv.Atoi(os.Getenv(`UPLOAD_CONCURRENCY`))
		if err != nil {
			pool = 50
		}
		sem := make(chan bool, pool)
		wg := sync.WaitGroup{}
		dest, err := url.Parse(st.Destination)
		if err != nil {
			panic(err)
		}
		for f := range files {
			_, loaded := uploading.LoadOrStore(f.Path+f.Name, 1)
			if !loaded {
				continue
			}
			sem <- true
			wg.Add(1)
			go func(f File) {
				defer func() {
					wg.Done()
					<-sem
					uploading.Delete(f.Path + f.Name)
				}()
				key := dest.Path + f.LastUpdated.Format(`year=2006/month=01/day=02/hour=15`) + `/` + f.Name
				r, err := os.Open(f.Path + f.Name)
				if err != nil {
					errors <- err
					return
				}
				log.Println(`uploading to: `, key)
				_, err = upd.Upload(&s3manager.UploadInput{
					Bucket: aws.String(dest.Host),
					Key:    aws.String(key),
					Body:   r,
				})
				if err != nil {
					errors <- err
					return
				}
				output <- f
			}(f)
		}
	}()
	return output, errors
}

func deleteFiles(files <-chan File) {
	for f := range files {
		log.Println(`removing file`, f)
		if err := os.Remove(f.Path + f.Name); err != nil {
			log.Println(err)
		}
	}
}
