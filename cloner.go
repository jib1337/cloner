package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/tdewolff/parse"
	"github.com/tdewolff/parse/css"
	"golang.org/x/net/html"
)

func sendRequest(URL string) ([]byte, error) {
	// Request a resource, handle errors and return it to the constructor

	resp, err := http.Get(URL)

	if resp.StatusCode <= 300 && resp.StatusCode > 400 {
		// If the request returns a 300 status, try not following redirects

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err = client.Get(URL)
	}

	if err != nil {
		return []byte("0"), err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return body, err

}

func checkError(err error, fileName string) bool {
	if err != nil {
		fmt.Printf("Error writing %s: %s\n", fileName, err.Error())
		return false
	}

	return true
}

func writeFile(data []byte, fileName string) bool {
	// Writes a web object file to disk

	err := ioutil.WriteFile(fileName, data, 0644)
	if checkError(err, fileName) == true {
		return true
	}

	return false
}

func getContent(URL string, outFolder string, largePaths []string) ([]string, []string) {
	// Retrieve local site content for use in clone

	var newPaths []string
	var newLargePaths []string
	var oldLink string

	for i, link := range largePaths {

		oldLink = link

		for strings.HasPrefix(link, "/") || strings.HasPrefix(link, "../") {
			// Resource is likely at root path

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

		fmt.Print("Getting content: ", link)

		if !strings.HasPrefix(link, URL) {
			link = URL + "/" + link
		}

		data, reqErr := sendRequest(link)

		if reqErr != nil {
			fmt.Print(" - " + reqErr.Error() + "\n")
		} else {
			fmt.Print(" - Success\n")
			newLargePaths = append(newLargePaths, oldLink)

			_, fileString := path.Split(link)
			newName := fmt.Sprintf("content%d%s", i, filepath.Ext(fileString))

			if filepath.Ext(fileString) == ".css" {
				// Modify file to contain new CSS elements if required

				data = constructCSS(string(data), largePaths, newPaths)
			}

			writeFile(data, outFolder+string(os.PathSeparator)+"content"+string(os.PathSeparator)+newName)
			newPaths = append(newPaths, fmt.Sprintf("content/%s", newName))
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
		bodyBytes, err := ioutil.ReadAll(data.Body)
		if err != nil {
			fmt.Println(err.Error())
			return stylePaths
		}

		l = css.NewLexer(parse.NewInputString(string(bodyBytes)))

	} else {

		l = css.NewLexer(parse.NewInputString(stylesheet))
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
						break
					}
				}
			}

			if "img" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "src" && !(strings.HasPrefix(attr.Val, "http://") || strings.HasPrefix(attr.Val, "https://")) {
						links = append(links, attr.Val)
						break
					}
				}
			}

			if "script" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "src" && !(strings.HasPrefix(attr.Val, "http://") || strings.HasPrefix(attr.Val, "https://")) {
						links = append(links, attr.Val)
						break
					}
				}
			}

			if "link" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "href" && !(strings.HasPrefix(attr.Val, "http://") || strings.HasPrefix(attr.Val, "https://")) {
						if filepath.Ext(attr.Val) == ".css" {
							cssLinks = append(cssLinks, attr.Val)

						} else {
							links = append(links, attr.Val)
						}
						break
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
				// Catch-all for embedded styles in any tag

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

func clean(URL string) string {
	// Clean up the main URL a bit

	u, err := url.Parse(URL)

	if err != nil {
		fmt.Println(err.Error())
		return URL
	}

	urlSplit := strings.Split(u.Path, "/")
	urlSuffix := urlSplit[len(urlSplit)-1]

	if path.Ext(urlSuffix) != "" {
		// If the final part of the URL has an extension, get rid of it (we can't request content links using it)
		URL = u.Scheme + "://" + u.Host + strings.Join(urlSplit[:len(urlSplit)-1], "/")
	}

	return URL
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
			newData = strings.Replace(newData, "action=\"", "action=\""+formURL+"\"", -1)
			newData = strings.Replace(newData, "action="+link, "action=\""+formURL+"\"", -1)
		}
	} else {
		fmt.Println("Skipping form action substitution")
	}

	writeFile([]byte(newData), outFolder+string(os.PathSeparator)+"index.html")
	fmt.Println("Site cloned to", outFolder)
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
	var err error

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

	printBanner()

	if strings.HasSuffix(URL, "/") {
		// Get rid of any single instance of a forward slash on the end (allows "escaping")
		URL = URL[:len(URL)-1]
	}

	if !strings.HasPrefix(URL, "http://") && !strings.HasPrefix(URL, "https://") {
		// Error out if no http/s prefix
		fmt.Println("Error: Please specify complete URL with http/https prefix")
		os.Exit(1)
	}

	err = os.Mkdir(outFolder, 0755)
	if checkError(err, outFolder) == false {
		os.Exit(1)
	}

	fmt.Println("Cloning page:", URL)

	data, reqErr := sendRequest(URL)

	if reqErr != nil {
		fmt.Println(reqErr.Error())
		os.Exit(1)
	}

	URL = clean(URL)

	err = os.Mkdir(outFolder+string(os.PathSeparator)+"content", 0755)
	if checkError(err, outFolder+string(os.PathSeparator)+"content") == false {
		os.Exit(1)
	}

	contentList, formLinks := parser(URL)

	shortPaths, largePaths := getContent(URL, outFolder, contentList)
	constructor(string(data), outFolder, largePaths, shortPaths, formLinks, formURL)
}
