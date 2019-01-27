// +build ignore

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

var errNonGithub = errors.New("can't derive repository URL from a package not hosted on github.com or gopkg.in")
var projectName = regexp.MustCompile(`(?m)^ *name = "(.+?)"$`)

func main() {
	if errGR := genRepos(os.Args[1]); errGR != nil {
		fmt.Println(errGR.Error())
		os.Exit(1)
	}
}

func genRepos(packag string) error {
	deps, errSD := scanDeps(packag)
	if errSD != nil {
		return errSD
	}

	deps[packag] = struct{}{}
	deps["github.com/golang/go"] = struct{}{}

	uniqueUrls, errLCRU := loadCustomRepoUrls(".")
	if errLCRU != nil {
		return errLCRU
	}

	for dep := range deps {
		depParts := strings.SplitN(dep, "/", 4)

		if strings.Contains(depParts[0], ".") {
			switch depParts[0] {
			case "golang.org":
				break
			case "github.com", "gopkg.in":
				uniqueUrls[strings.TrimRight("https://"+strings.Join(depParts[:3], "/"), "/")] = struct{}{}
			default:
				return errNonGithub
			}
		}
	}

	urls := make([]string, len(uniqueUrls))
	urlIdx := 0

	for url := range uniqueUrls {
		urls[urlIdx] = url
		urlIdx++
	}

	sort.Strings(urls)

	return ioutil.WriteFile(
		"GithubcomAl2klimovGo_gen_source_repos.go",
		[]byte(fmt.Sprintf("package main\nvar GithubcomAl2klimovGo_gen_source_repos = %#v", urls)),
		0666,
	)
}

func scanDeps(packag string) (map[string]struct{}, error) {
	gopkgLock, errRGL := ioutil.ReadFile("Gopkg.lock")
	if errRGL != nil {
		return nil, errRGL
	}

	pkgs := make(map[string]struct{})

	for _, name := range projectName.FindAllSubmatch(gopkgLock, -1) {
		pkgs[string(name[1])] = struct{}{}
	}

	return pkgs, nil
}

func loadCustomRepoUrls(rootDir string) (map[string]struct{}, error) {
	allUrls := make(map[string]struct{})

	{
		rawUrls, errRF := ioutil.ReadFile(path.Join(rootDir, "GithubcomAl2klimovGo_gen_source_repos.txt"))
		if errRF == nil {
			for _, line := range bytes.Split(rawUrls, []byte{'\n'}) {
				if len(line) > 0 {
					allUrls[string(line)] = struct{}{}
				}
			}
		} else if !os.IsNotExist(errRF) {
			return nil, errRF
		}
	}

	entries, errRD := ioutil.ReadDir(rootDir)
	if errRD != nil {
		return nil, errRD
	}

	for _, entry := range entries {
		if entry.IsDir() {
			urls, errLCRU := loadCustomRepoUrls(path.Join(rootDir, entry.Name()))
			if errLCRU != nil {
				return nil, errLCRU
			}

			for url := range urls {
				allUrls[url] = struct{}{}
			}
		}
	}

	return allUrls, nil
}
