package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/tidwall/gjson"
)

type Photographer struct {
	name string
	url  string
}

type Camera struct {
	name string
	url  string
}

type Lens struct {
	name string
	url  string
}

type ShutterSpeed struct {
	numerator   int
	denominator int
}

type Exif struct {
	camera               Camera
	lens                 Lens
	displayText          string
	focalLength          int
	shutterSpeed         ShutterSpeed
	aperture             int
	iso                  int
	exposureComp         int
	videoUrl             *string
	videoFps             *int
	videoCodec           *string
	videoRecordingDevice *string
}

type Thumbnail struct {
	url    string
	width  int
	height int
}

type Image struct {
	id           string
	index        int
	hidden       bool
	userLiked    bool
	creator      Photographer
	title        string
	description  string
	exif         Exif
	thumbnails   []Thumbnail
	width        int
	height       string
	size         string
	url          string
	rawUrl       string
	rawSize      string
	likes        int
	hasUserLiked bool
	directUrl    string
	commentId    string
	commentCount int
}

type Gallery struct {
	id              int
	isHidden        bool
	sponsored       bool
	title           string
	totalImages     int
	photographers   []Photographer
	likes           int
	url             string
	commentsEnabled bool
	images          []Image
}

type SampleGallery struct {
	gallery    Gallery
	adSlotHtml string
	images     []Image
}

func createJson(jsonData string, fileName string) bool {
	raw := strings.Replace(jsonData, "\\", "", -1)
	j, err := json.MarshalIndent(raw, "", "\t")
	if err != nil {
		log.Fatalln(err.Error())
	}
	json := strings.Replace(string(j[:]), "\\", "", -1)
	json = json[:len(json)-1][1:]
	if err := ioutil.WriteFile(fileName+".json", []byte(json), 0644); err != nil {
		log.Fatalln(err.Error())
	}
	return true
}

func saveImage(url string, path string) bool {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err.Error())
	}
	req.Header.Set("User-Agent", "dpreview_scraper/1.0")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer res.Body.Close()

	re := regexp.MustCompile(`[^/]+$`)
	m := re.FindStringSubmatch(url)
	file := m[0]
	f, err := os.Create(path + "/" + file)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer f.Close()

	_, err = f.ReadFrom(res.Body)
	if err != nil {
		log.Fatalln(err.Error())
	}
	return true
}

func main() {
	url := flag.String("url", "", "url to access")
	flag.Parse()
	if *url == "" {
		log.Fatalln("-url must be provided!")
	}
	id := ""

	c := colly.NewCollector()

	// https://github.com/gocolly/colly/issues/26
	visited := false
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if visited {
			return
		}
		if strings.HasPrefix(e.Attr("href"), "/sample-galleries") {
			re := regexp.MustCompile(`\d{10}`)
			m := re.FindStringSubmatch(e.Attr("href"))
			id = m[0]
			visited = true
		}
	})

	c.Visit(*url)

	if id == "" {
		log.Fatalln("No gallery ID found!")
	}

	api := "https://www.dpreview.com/sample-galleries/data/get-gallery?galleryId=" + id
	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		log.Fatalln(err.Error())
	}
	req.Header.Set("User-Agent", "dpreview_scraper/1.0")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer res.Body.Close()
	b, rErr := ioutil.ReadAll(res.Body)
	if rErr != nil {
		log.Fatalln(err.Error())
	}
	gallery := gjson.Get(string(b[:]), "gallery")
	images := gjson.Get(string(b[:]), "images")

	re := regexp.MustCompile(`[^/]+$`)
	m := re.FindStringSubmatch(*url)
	folder := m[0]
	path := "galleries/" + folder
	err = os.MkdirAll(path+"/images", os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}

	createJson(gallery.Raw, path+"/gallery")
	createJson(images.Raw, path+"/images")

	imageList := gjson.Get(string(b[:]), "images.#.thumbnails.#.url")
	for _, images := range imageList.Array() {
		for _, image := range images.Array() {
			saveImage(image.String(), path+"/images")
		}
	}
}
