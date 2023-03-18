// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
)

var projectName = regexp.MustCompile(`(?m)^ *name = "(.+?)"$`)

func main() {
	packag := ""
	if len(os.Args) > 1 {
		packag = os.Args[1]
	}

	if errGR := genRepos(packag); errGR != nil {
		fmt.Println(errGR.Error())
		os.Exit(1)
	}
}

func genRepos(packag string) error {
	deps, errSD := scanDeps()
	if errSD != nil {
		return errSD
	}

	if packag != "" {
		deps[packag] = struct{}{}
	}

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
			case "github.com", "gopkg.in", "moul.io":
				uniqueUrls[strings.TrimRight("https://"+strings.Join(depParts[:3], "/"), "/")] = struct{}{}
			case "google.golang.org":
				uniqueUrls["https://github.com/golang/"+depParts[1]] = struct{}{}
			default:
				uniqueUrls["https://pkg.go.dev/"+dep] = struct{}{}
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

var goListJson = regexp.MustCompile(`(?ms)^{.*?^}`)

func scanDeps() (map[string]struct{}, error) {
	pkgs := map[string]struct{}{}

	{
		gopkgLock, errRGL := ioutil.ReadFile("Gopkg.lock")
		if errRGL != nil {
			if os.IsNotExist(errRGL) {
				gopkgLock = nil
			} else {
				return nil, errRGL
			}
		}

		for _, name := range projectName.FindAllSubmatch(gopkgLock, -1) {
			pkgs[string(name[1])] = struct{}{}
		}
	}

	if _, errStat := os.Stat("go.mod"); errStat == nil || !os.IsNotExist(errStat) {
		cmd := exec.Command("go", "list", "-json", "-m", "all")
		var outBuf bytes.Buffer

		cmd.Stdin = nil
		cmd.Stdout = &outBuf
		cmd.Stderr = os.Stderr

		if errRun := cmd.Run(); errRun != nil {
			return nil, errRun
		}

		for _, jsn := range goListJson.FindAll(outBuf.Bytes(), -1) {
			var data struct{ Path string }
			if errJU := json.Unmarshal(jsn, &data); errJU != nil {
				return nil, errJU
			}

			if data.Path != "" {
				pkgs[data.Path] = struct{}{}
			}
		}
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
