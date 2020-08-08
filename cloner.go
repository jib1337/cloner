package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/html"
)

func sendRequest(url string, index bool) ([]byte, error) {
	// Request a resource and return it to the constructor

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println(err.Error())
	}
	defer resp.Body.Close()

	if err != nil {
		// Print an error for an item if it cannot be retrieved.
		fmt.Printf("Error encountered whilst retrieving %s:\n\t%s\n", url, err)
	}

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

func parser(url string) []string {
	// Create an array of referenced objects from the page

	data, err := http.Get(url)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer data.Body.Close()

	var links []string
	z := html.NewTokenizer(data.Body)
	for {
		tt := z.Next()

		switch tt {

		case html.ErrorToken:
			return links

		case html.StartTagToken, html.EndTagToken:
			token := z.Token()
			if "img" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						links = append(links, attr.Val)
					}
				}
			}

			if "script" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						links = append(links, attr.Val)
					}
				}
			}

			if "link" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			}
		}
	}
}

func main() {

	//TODO: Make these arguments
	url := "https://vorozhko.net/get-all-links-from-html-page-with-go-lang"
	currentTime := time.Now()
	outFolder := "C:\\Users\\Jack\\Desktop\\" + "cloned-" + currentTime.Format("2006-01-02 15-04-05")

	fmt.Println("Cloner v0.1\nCloning page:", url)
	data, reqErr := sendRequest(url, false)

	if reqErr != nil {
		fmt.Println(reqErr.Error())
		os.Exit(1)
	}

	// Set up content directories
	err := os.Mkdir(outFolder, 0755)
	check(err)
	err2 := os.Mkdir(outFolder+string(os.PathSeparator)+"content", 0755)
	check(err2)
	writeFile(data, outFolder+string(os.PathSeparator)+"index.html")

	objects := parser(url)
	for _, v := range objects {
		fmt.Println(v)
	}
}
