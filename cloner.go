package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tdewolff/parse/css"
	"golang.org/x/net/html"
)

func sendRequest(URL string) ([]byte, error) {
	// Request a resource and return it to the constructor

	resp, err := http.Get(URL)

	if err != nil {
		return []byte("0"), err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	return body, err

}

func check(e error) {
	// Checking for file handling

	if e != nil {
		panic(e)
	}
}

func writeFile(data []byte, fileName string) {
	// Writes a web object file to disk

	err := ioutil.WriteFile(fileName, data, 0644)
	check(err)
}

func getContent(URL string, outFolder string, largePaths []string) ([]string, []string) {
	// Retrieve local site content for use in clone

	var newPaths []string
	var newLargePaths []string
	var oldLink string

	for i, link := range largePaths {

		oldLink = link

		if strings.HasPrefix(link, "/") {
			// Resource is at root path

			link = string([]rune(link)[1:])
			u, err := url.Parse(URL)

			if err != nil {
				fmt.Print(" -", err.Error())
				continue
			}

			URL = u.Scheme + "://" + u.Host
		}

		if strings.HasPrefix(link, "./") {
			link = string([]rune(link)[2:])
		}

		fmt.Print("Getting content: ", URL+"/"+link)

		data, reqErr := sendRequest(URL + "/" + link)

		if reqErr != nil {
			fmt.Print(" - " + reqErr.Error())
		} else {
			fmt.Print(" - Success\n")
			newLargePaths = append(newLargePaths, oldLink)

			_, fileString := path.Split(link)
			newName := "content" + strconv.Itoa(i) + filepath.Ext(fileString)

			if filepath.Ext(fileString) == ".css" {
				// Modify file to contain new CSS elements if required

				data = constructCSS(string(data), largePaths, newPaths)
			}

			writeFile(data, outFolder+string(os.PathSeparator)+"content"+string(os.PathSeparator)+newName)
			newPaths = append(newPaths, "content/"+"content"+strconv.Itoa(i)+filepath.Ext(fileString))
		}

	}

	return newPaths, newLargePaths
}

func constructCSS(data string, largePaths []string, newPaths []string) []byte {
	// Update stylesheet

	fmt.Println("Constructing CSS...")
	newData := data

	for i, content := range largePaths {
		if filepath.Ext(content) == ".css" {
			break
		}
		_, fileString := path.Split(newPaths[i])
		newData = strings.ReplaceAll(newData, content, fileString)
	}

	return []byte(newData)
}

func parseCSS(URL string, stylesheet string, embedded bool) []string {
	// Get links out of stylesheets for processing

	var stylePaths []string
	var l *css.Lexer

	if embedded == false {
		// Recieved a list of css file paths

		if strings.HasPrefix(stylesheet, ".") {
			stylesheet = string([]rune(stylesheet)[1:])
		}

		data, err := http.Get(URL + "/" + stylesheet)
		if err != nil {
			fmt.Println(err.Error())
			data.Body.Close()
			return stylePaths
		}

		defer data.Body.Close()
		l = css.NewLexer(data.Body)

	} else {

		l = css.NewLexer(bytes.NewBufferString(stylesheet))
	}

	for {
		tt, text := l.Next()
		switch tt {
		case css.ErrorToken:
			return stylePaths

		case css.URLToken:
			stylePath := strings.Replace(strings.Replace(strings.Replace(strings.Replace(string(text), "url(", "", -1), "'", "", -1), "\"", "", -1), ")", "", -1)
			stylePaths = append(stylePaths, stylePath)
		}
	}
}

func parser(URL string) ([]string, []string) {
	// Create an array of referenced objects from the page

	data, err := http.Get(URL)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer data.Body.Close()

	var links []string
	var formLinks []string
	var cssLinks []string
	var stylePaths []string

	z := html.NewTokenizer(data.Body)
	for {
		tt := z.Next()

		switch tt {

		case html.ErrorToken:

			for _, l := range cssLinks {
				stylePaths = parseCSS(URL, l, false)
				for _, path := range stylePaths {
					links = append(links, path)
				}

				links = append(links, l)
			}

			return links, formLinks

		case html.StartTagToken, html.EndTagToken:
			token := z.Token()

			if "form" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "action" {
						formLinks = append(formLinks, attr.Val)
					} else if attr.Key == "style" {
						stylePaths = parseCSS(URL, attr.Val, true)
						for _, path := range stylePaths {
							links = append(links, path)
						}
					}
				}
			}

			if "img" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "src" && !(strings.HasPrefix(attr.Val, "http://") || strings.HasPrefix(attr.Val, "https://")) {
						links = append(links, attr.Val)
					} else if attr.Key == "style" {
						stylePaths = parseCSS(URL, attr.Val, true)
						for _, path := range stylePaths {
							links = append(links, path)
						}
					}
				}
			}

			if "script" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "src" && !(strings.HasPrefix(attr.Val, "http://") || strings.HasPrefix(attr.Val, "https://")) {
						links = append(links, attr.Val)
					} else if attr.Key == "style" {
						stylePaths = parseCSS(URL, attr.Val, true)
						for _, path := range stylePaths {
							links = append(links, path)
						}
					}
				}
			}

			if "link" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "href" && !(strings.HasPrefix(attr.Val, "http://") || strings.HasPrefix(attr.Val, "https://")) {
						if filepath.Ext(attr.Val) == ".css" {
							cssLinks = append(cssLinks, attr.Val)
							continue
						} else {
							links = append(links, attr.Val)
							continue
						}
					}
				}
			}

			if "style" == token.Data {
				z.Next()
				embeddedStyles := parseCSS(URL, string(z.Text()), true)
				for _, link := range embeddedStyles {
					links = append(links, link)
				}
			}

			for _, attr := range token.Attr {
				// Catch-all for embedded styles

				if attr.Key == "style" {
					stylePaths = parseCSS(URL, attr.Val, true)
					for _, path := range stylePaths {
						links = append(links, path)
					}
				}
			}
		}
	}
}

func constructor(data string, outFolder string, largePaths []string, shortPaths []string, formLinks []string, formURL string) {
	// Creates main page document
	fmt.Println("Building page...")
	newData := data

	for i, content := range largePaths {
		newData = strings.ReplaceAll(newData, content, shortPaths[i])
	}

	if len(formLinks) > 0 && formURL != "" {
		fmt.Println("Performing form action substitution with: " + formURL)
		for _, link := range formLinks {
			newData = strings.Replace(newData, "action=\""+link+"\"", "action=\""+formURL+"\"", -1)
			newData = strings.Replace(newData, "action="+link, "action=\""+formURL+"\"", -1)
		}
	} else {
		fmt.Println("Skipping form action substitution")
	}

	writeFile([]byte(newData), outFolder+string(os.PathSeparator)+"index.html")
	println("Site cloned to", outFolder)
}

func printBanner() {
	fmt.Println("============================================================")
	fmt.Println("                           cloner")
	fmt.Println("============================================================")
}

func main() {

	var formURL string
	var URL string
	var outFolder string
	flag.StringVar(&URL, "u", "", "The URL of the site to clone")
	flag.StringVar(&formURL, "f", "", "The URL of the site to replace in form actions")
	flag.StringVar(&outFolder, "o", "."+string(os.PathSeparator), "Output location")
	flag.Parse()

	if URL == "" {
		printBanner()
		flag.PrintDefaults()
		os.Exit(1)
	}

	if !(strings.HasSuffix(outFolder, string(os.PathSeparator))) {
		outFolder = outFolder + string(os.PathSeparator)
	}

	currentTime := time.Now()
	outFolder = outFolder + "cloned-" + currentTime.Format("2006-01-02-15-04-05")

	if strings.HasSuffix(URL, "/") {
		URL = URL[:len(URL)-1]
	}

	printBanner()
	fmt.Println("Cloning page:", URL)
	data, reqErr := sendRequest(URL)

	if reqErr != nil {
		fmt.Println(reqErr.Error())
		os.Exit(1)
	}

	err := os.Mkdir(outFolder, 0755)
	check(err)
	err2 := os.Mkdir(outFolder+string(os.PathSeparator)+"content", 0755)
	check(err2)

	contentList, formLinks := parser(URL)

	shortPaths, largePaths := getContent(URL, outFolder, contentList)
	constructor(string(data), outFolder, largePaths, shortPaths, formLinks, formURL)
}
