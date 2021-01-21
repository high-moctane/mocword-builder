package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

var validLanguages = []string{
	"eng",
	"eng-us",
	"eng-gb",
	"eng-fiction",
	"chi_sim",
	"fre",
	"ger",
	"heb",
	"ita",
	"rus",
	"spa",
}

var validNgrams = []string{"1", "2", "3", "4", "5"}

var flagLanguage = flag.String(
	"language", strings.Join(validLanguages, ","),
	"comma separated language names\n("+strings.Join(validLanguages, ",")+")\n",
)

var flagNgram = flag.String(
	"ngram", strings.Join(validNgrams, ","),
	"comma separated ngram number ("+strings.Join(validNgrams, ",")+")",
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := parseFlags(); err != nil {
		return err
	}

	url := downloadIndexURL("eng", "2")
	body, _ := getHTML(url)
	list, _ := dataURLList(body)
	fmt.Println(strings.Join(list, "\n"))

	return nil
}

func parseFlags() error {
	flag.Parse()
	if err := verifyFlags(); err != nil {
		return fmt.Errorf("cannot parse flags: %w", err)
	}
	return nil
}

func verifyFlags() error {
	if err := verifyFlagLanguage(*flagLanguage); err != nil {
		return fmt.Errorf("invalid flag: %w", err)
	}

	if err := verifyFlagNgram(*flagNgram); err != nil {
		return fmt.Errorf("invalid flag: %w", err)
	}

	return nil
}

func verifyFlagLanguage(flg string) error {
	if invalid := findInvalidFlagElement(flg, validLanguages); invalid != "" {
		return fmt.Errorf("invalid language flag: %q", invalid)
	}
	return nil
}

func verifyFlagNgram(flg string) error {
	if invalid := findInvalidFlagElement(flg, validNgrams); invalid != "" {
		return fmt.Errorf("invalid ngram flag: %q", invalid)
	}
	return nil
}

func findInvalidFlagElement(rawFlag string, validFlags []string) string {
	flags := strings.Split(rawFlag, ",")

	for _, flg := range flags {
		found := false
		for _, validFlg := range validFlags {
			found = found || flg == validFlg
		}

		if !found {
			return flg
		}
	}

	return ""
}

func totalCountsURL(lang, ngram string) string {
	return fmt.Sprintf("http://storage.googleapis.com/books/ngrams/books/20200217/%s/totalcounts-%s", lang, ngram)
}

func downloadIndexURL(lang, ngram string) string {
	return fmt.Sprintf("http://storage.googleapis.com/books/ngrams/books/20200217/%s/%s-%s-ngrams_exports.html", lang, lang, ngram)
}

func getHTML(url string) (body string, err error) {
	res, err := http.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = fmt.Errorf("cannot get %s", url)
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	if !utf8.Valid(buf) {
		err = fmt.Errorf("non-unicode HTML: %s", url)
	}
	body = string(buf)

	return
}

func dataURLList(body string) (urls []string, err error) {
	r := strings.NewReader(body)
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		err = fmt.Errorf("cannot get data urls: %w", err)
		return
	}

	doc.Find("li").Each(func(_ int, s *goquery.Selection) {
		url, ok := s.Find("a").Attr("href")
		if !ok {
			err = fmt.Errorf("cannot get data urls: invalid attr: %s", s.Find("a").Text())
			return
		}

		urls = append(urls, url)
	})

	return
}

func do(ctx context.Context, url, dir string) error {
	done := false
	fname := path.Base(url)
	absFname := filepath.Join(dir, fname)

	if _, err := os.Stat(absFname); os.IsExist(err) {
		return nil
	}

	tmpfile, err := ioutil.TempFile(dir, fname)
	if err != nil {
		return fmt.Errorf("do error: %w")
	}
	defer func() {
		if !done {
			os.Remove(tmpfile.Name())
		}
		os.Rename(tmpfile.Name(), absFname)
	}()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("do error: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do error: %w", err)
	}
	defer resp.Body.Close()

	r := bufio.NewReader(resp.Body)
	gr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gr.Close()

}
