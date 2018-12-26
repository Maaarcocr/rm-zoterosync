package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Maaarcocr/rmsync"
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
	fmt.Println(BaseZoteroURL + UserId + "/collections/" + directory.Key + "/items")
	res, err := getJson(BaseZoteroURL+UserId+"/collections/"+directory.Key+"/items", &zoteroItems)
	if err != nil {
		return nil, err
	}
	next := ""
	if val, ok := link.ParseResponse(res)["next"]; ok {
		next = val.String()
	}
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

func createRemarkableFileMap(files []rmsync.RemarkableFile) map[string]struct{} {
	fileMap := make(map[string]struct{}, 0)
	for _, file := range files {
		fileMap[file.VisibleName] = struct{}{}
	}
	return fileMap
}

func getSharedDirectories(directories []rmsync.RemarkableFile, zoteroDirectories []ZoteroDirectory) []ZoteroDirectory {
	var sharedDirectories []ZoteroDirectory
	dirMap := createRemarkableFileMap(directories)
	for _, zoteroDirectory := range zoteroDirectories {
		if _, ok := dirMap[zoteroDirectory.Data.Name]; ok {
			sharedDirectories = append(sharedDirectories, zoteroDirectory)
		}
	}
	return sharedDirectories
}

func createRemarkableFilesToSync(pdfs []ZoteroItem) []rmsync.FileToSync {
	files := make([]rmsync.FileToSync, 0)
	for _, pdf := range pdfs {
		files = append(files, rmsync.FileToSync{pdf.Data.Filename, pdf.Data.Url})
	}
	return files
}

func sync() error {
	directories, err := rmsync.GetDirectoriesMetadataFiles()
	if err != nil {
		return err
	}
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
		filesToSync := createRemarkableFilesToSync(pdfs)
		err = rmsync.Sync(filesToSync)
		if err != nil {
			return err
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
