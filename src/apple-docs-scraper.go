package main

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const BaseUrl = "https://developer.apple.com/tutorials/data"
const StartUrl = "/documentation/devicemanagement.json"
const MustContain = "devicemanagement"

type Doc struct {
	Abstract      []map[string]string  `json:"abstract"`
	Identifier    map[string]string    `json:"identifier"`
	References    map[string]Reference `json:"references"`
	TopicSections []TopicSection       `json:"topicSections"`
}

type Reference struct {
	Title      string `json:"title"`
	Identifier string `json:"identifier"`
	Url        string `json:"url"`
}

type TopicSection struct {
	Title       string   `json:"title"`
	Identifiers []string `json:"identifiers"`
}

func main() {

	queue := list.New()
	queue.PushBack(BaseUrl + StartUrl)

	for queue.Len() > 0 {
		url := queue.Front()
		queue.Remove(url)

		visited := make(map[string]bool)
		writen := make(map[string]bool)

		crawl(url.Value.(string), queue, visited, writen)
	}
}

func crawl(url string, queue *list.List, visited map[string]bool, writen map[string]bool) {
	isDeviceManagementDoc := strings.Contains(url, MustContain)
	_, alreadyVisited := visited[url]

	if alreadyVisited || !isDeviceManagementDoc {
		return
	}

	visited[url] = true

	fmt.Printf("GETting %v ...\n", url)
	resp, err := http.Get(url)

	if resp.StatusCode != 200 {
		fmt.Printf("got status %v for url %v", resp.StatusCode, url)
		return
	}

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(body)))

	var data Doc
	err = decoder.Decode(&data)

	if err != nil {
		log.Fatal(err)
	}

	identifier := data.Identifier["url"]
	_, alreadyWriten := writen[identifier]

	if !alreadyWriten {
		fmt.Printf("writing %v to disk...\n", identifier)

		writeToFile(string(body), identifier)

		writen[identifier] = true
	} else {
		log.Printf("%v has been writen before, skipping.", identifier)
	}

	// now for all references, recurse...
	for _, ref := range data.References {
		formattedUrl := BaseUrl + ref.Url + ".json"
		crawl(formattedUrl, queue, visited, writen)
	}
}

func getPathAndFileFromDocId(docUrl string) (string, string, bool) {
	re := regexp.MustCompile(`^doc://(.*)/(.*)/?$`) // (`^doc:(.*/)(\w*)/?$`)
	matches := re.FindAllStringSubmatch(docUrl, -1)

	if matches == nil {
		return "", "", false
	}

	return matches[0][1], matches[0][2], true
}

func writeToFile(str string, docId string) {
	currentDir, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}

	path, fileName, ok := getPathAndFileFromDocId(docId)

	if !ok {
		panic(fmt.Sprintf("unable to break down docId %v", docId))
	}

	fullPath := fmt.Sprintf(`%v/%v`, currentDir, path)
	filePath := fmt.Sprintf(`%v/%v.json`, fullPath, fileName)

	err = os.MkdirAll(fullPath, os.ModePerm)

	if err != nil {
		log.Println(err)
	}

	f, err := os.Create(filePath)

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	prettyJson, err := prettify(str)

	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteString(string(prettyJson))

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("wrote %v to disk\n\n", docId)
}

func prettify(str string) (string, error) {
	var prettyJSON bytes.Buffer

	if err := json.Indent(&prettyJSON, []byte(str), "", "\t"); err != nil {
		return "", err
	}

	return prettyJSON.String(), nil
}
