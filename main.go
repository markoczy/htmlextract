package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/net/html"

	"github.com/antchfx/htmlquery"
)

// checks for error and panics
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// discard ignores variable
func discard(interface{}) {}

// task single entity of work
type task struct {
	in, out string
}

// traverse walks html recursively and extracts text
func traverse(n *html.Node, sb *strings.Builder) {
	for next := n.FirstChild; next != nil; next = next.NextSibling {
		atom := next.DataAtom.String()

		switch {
		case atom == "script" || atom == "noscript" || atom == "style":
			// skip
			continue
		case next.Type == html.TextNode:
			sb.WriteString(next.Data)
		default:
			traverse(next, sb)
		}
	}
}

// extractInnerText extracts inner text of a html structure
func extractInnerText(pathToHTMLFile string) string {
	sb := strings.Builder{}
	file, err := os.Open(pathToHTMLFile)
	check(err)

	reader := bufio.NewReader(file)
	doc, err := htmlquery.Parse(reader)
	check(err)

	traverse(doc, &sb)

	text := sb.String()
	rx, err := regexp.Compile("(\\s+)")
	check(err)
	text = rx.ReplaceAllString(text, " ")

	return text
}

// initTasks initialises filesystem and creates the tasks
func initTasks(pathIn, pathOut string, tasks []task) []task {
	dirCreated := false
	createDir := func() {
		if !dirCreated {
			err := os.MkdirAll(pathOut, 0777)
			check(err)
		}
		dirCreated = true
	}

	files, err := ioutil.ReadDir(pathIn)
	check(err)

	for _, file := range files {
		switch {
		case file.IsDir():
			tasks = initTasks(pathIn+"/"+file.Name(), pathOut+"/"+file.Name(), tasks)
		case strings.HasSuffix(strings.ToLower(file.Name()), ".html"):
			createDir()
			tasks = append(tasks, task{pathIn + "/" + file.Name(), pathOut + "/" + file.Name() + ".txt"})
		}
	}
	return tasks
}

func main() {
	in := flag.String("in", "", "path to input directory")
	out := flag.String("out", "", "path to output directory")
	flag.Parse()

	if *in == "" || *out == "" {
		flag.Usage()
		return
	}

	fmt.Println("Initialising...")
	tasks := initTasks(*in, *out, []task{})
	fmt.Printf("Found %d html files to process.\nProcessing...\n", len(tasks))

	var waitGroup sync.WaitGroup
	defer func() {
		waitGroup.Wait()
		fmt.Println("Done.")
	}()

	for _, cur := range tasks {
		waitGroup.Add(1)
		go func(cur task, wg *sync.WaitGroup) {
			// fmt.Println("Current task", cur)
			text := extractInnerText(cur.in)
			ioutil.WriteFile(cur.out, []byte(text), 077)
			wg.Done()
		}(cur, &waitGroup)
	}

}
