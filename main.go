package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterhellberg/link"
)

var UserId string = os.Getenv("ZOTERO_USERID")
var ApiKey string = os.Getenv("ZOTERO_APIKEY")
var myClient = &http.Client{Timeout: 10 * time.Second}

const BaseZoteroURL string = "https://api.zotero.org/users/"

const BaseDir string = "/home/root/.local/share/remarkable/xochitl/"

type Metadata struct {
	Deleted          bool   `json:"deleted"`
	DastModified     string `json:"lastModified"`
	Metadatamodified bool   `json:"metadatamodified"`
	Modified         bool   `json:"modified"`
	Parent           string `json:"parent"`
	Pinned           bool   `json:"pinned"`
	Synced           bool   `json:"synced"`
	Type             string `json:"type"`
	Version          int    `json:"version"`
	VisibleName      string `json:"visibleName"`
}

type ZoteroItem struct {
	Key  string         `json:"key"`
	Data ZoteroItemData `json:"data"`
}

type ZoteroItemData struct {
	ContentType string `json:"contentType"`
	Filename    string `json:"filename"`
	Url         string `json:"url"`
}

type ZoteroDirectory struct {
	Key  string        `json:"key"`
	Data ZoteroDirData `json:"data"`
}

type ZoteroDirData struct {
	Key              string      `json:"key"`
	Version          int         `json:"version"`
	Name             string      `json:"name"`
	ParentCollection bool        `json:"parentCollection"`
	Relations        interface{} `json:"relations"`
}

type RemarkableFile struct {
	Filename    string
	VisibleName string
}

func getMetadataFilenames() ([]string, error) {
	var filenames []string
	err := filepath.Walk(BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".metadata" {
			return nil
		}
		filenames = append(filenames, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filenames, nil
}

func getDirectories(filenames []string) ([]RemarkableFile, error) {
	var directories []RemarkableFile
	for _, filename := range filenames {
		filecontent, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		var metadata Metadata
		err = json.Unmarshal(filecontent, &metadata)
		if err != nil {
			return nil, err
		}
		if metadata.Type == "CollectionType" {
			directories = append(directories, RemarkableFile{filename, metadata.VisibleName})
		}
	}
	return directories, nil
}

func getPdfFiles(filenames []string) ([]RemarkableFile, error) {
	var pdfFiles []RemarkableFile
	for _, filename := range filenames {
		filecontent, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		var metadata Metadata
		err = json.Unmarshal(filecontent, &metadata)
		if err != nil {
			return nil, err
		}
		if metadata.Type == "DocumentType" {
			pdfFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".pdf"
			if _, err := os.Stat(pdfFilename); !os.IsNotExist(err) {
				pdfFiles = append(pdfFiles, RemarkableFile{pdfFilename, metadata.VisibleName})
			}
		}
	}
	return pdfFiles, nil
}

func getJson(url string, target interface{}) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Zotero-API-Key", ApiKey)
	res, err := myClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return res, json.NewDecoder(res.Body).Decode(target)
}

func getZoteroDirectories() ([]ZoteroDirectory, error) {
	var directoriesJson []ZoteroDirectory
	_, err := getJson(BaseZoteroURL+UserId+"/collections", &directoriesJson)
	if err != nil {
		return nil, err
	}
	return directoriesJson, nil
}

func getZoteroItemsForDirectory(directory ZoteroDirectory) ([]ZoteroItem, error) {
	var zoteroItems []ZoteroItem
	res, err := getJson(BaseZoteroURL+UserId+"/collections/"+directory.Key+"/items", &zoteroItems)
	if err != nil {
		return nil, err
	}
	next := link.ParseResponse(res)["next"].String()
	for next != "" {
		var tempZoteroItems []ZoteroItem
		res, err := getJson(next, &tempZoteroItems)
		if err != nil {
			return nil, err
		}
		zoteroItems = append(zoteroItems, tempZoteroItems...)
		next = ""
		if val, ok := link.ParseResponse(res)["next"]; ok {
			next = val.String()
		}
	}

	return zoteroItems, nil
}

func getZoteroPdfsFromItems(items []ZoteroItem) []ZoteroItem {
	var zoteroPdfs []ZoteroItem
	for _, item := range items {
		if item.Data.ContentType == "application/pdf" {
			zoteroPdfs = append(zoteroPdfs, item)
		}
	}
	return zoteroPdfs
}

func createRemarkableFileMap(files []RemarkableFile) map[string]struct{} {
	fileMap := make(map[string]struct{}, 0)
	for _, file := range files {
		fileMap[file.VisibleName] = struct{}{}
	}
	return fileMap
}

func getSharedDirectories(directories []RemarkableFile, zoteroDirectories []ZoteroDirectory) []ZoteroDirectory {
	var sharedDirectories []ZoteroDirectory
	dirMap := createRemarkableFileMap(directories)
	for _, zoteroDirectory := range zoteroDirectories {
		if _, ok := dirMap[zoteroDirectory.Data.Name]; ok {
			sharedDirectories = append(sharedDirectories, zoteroDirectory)
		}
	}
	return sharedDirectories
}

func downloadPdfFile(url string) ([]byte, error) {
	r, err := myClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	fileContents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return fileContents, nil
}

func uploadPdfToTablet(fileContents []byte, filename string) error {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	part.Write(fileContents)

	err = writer.Close()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", "http://10.11.99.1/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")
	if err != nil {
		return err
	}
	_, err = myClient.Do(req)
	if err != nil {
		return err
	}
	return nil
}

func sync() error {
	filenames, err := getMetadataFilenames()
	if err != nil {
		return err
	}
	directories, err := getDirectories(filenames)
	if err != nil {
		return err
	}
	pdfFiles, err := getPdfFiles(filenames)
	if err != nil {
		return err
	}
	pdfFileMap := createRemarkableFileMap(pdfFiles)
	zoteroDirectories, err := getZoteroDirectories()
	if err != nil {
		return err
	}
	sharedDirectories := getSharedDirectories(directories, zoteroDirectories)
	for _, zoteroDirectory := range sharedDirectories {
		items, err := getZoteroItemsForDirectory(zoteroDirectory)
		if err != nil {
			return err
		}
		pdfs := getZoteroPdfsFromItems(items)
		for _, item := range pdfs {
			if _, ok := pdfFileMap[item.Data.Filename]; !ok {
				fmt.Println(item.Data.Url)
				fileContents, err := downloadPdfFile(item.Data.Url)
				if err != nil {
					return err
				}
				err = uploadPdfToTablet(fileContents, item.Data.Filename)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

var prevSyncedTime time.Time

func main() {
	for {
		currTime := time.Now()
		if prevSyncedTime.IsZero() {
			err := sync()
			if err != nil {
				fmt.Println(err)
				continue
			}
			prevSyncedTime = currTime
			fmt.Println("SUCCESSFUL SYNC")
			continue
		}
		diff := currTime.Sub(prevSyncedTime)

		if diff > time.Minute*10 {
			err := sync()
			if err != nil {
				fmt.Println(err)
				continue
			}
			prevSyncedTime = currTime
			fmt.Println("SUCCESSFUL SYNC")
		}
		time.Sleep(time.Minute)
	}
}
