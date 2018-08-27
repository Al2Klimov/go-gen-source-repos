// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"
)

var errGoListBadOut = errors.New("got bad output from 'go list -json'")
var errNonGithub = errors.New("can't derive repository URL from a package not hosted on github.com")

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

	uniqueUrls := map[string]struct{}{}

	for dep := range deps {
		depParts := strings.SplitN(dep, "/", 4)

		if strings.Contains(depParts[0], ".") {
			switch depParts[0] {
			case "golang.org":
				break
			case "github.com":
				uniqueUrls["https://"+strings.Join(depParts[:3], "/")] = struct{}{}
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
	cmd := exec.Command("go", "list", "-json", packag)
	outBuf := bytes.Buffer{}

	cmd.Stdout = &outBuf
	cmd.Stderr = os.Stderr

	if errCR := cmd.Run(); errCR != nil {
		return nil, errCR
	}

	var goList interface{}

	if errJUM := json.Unmarshal(outBuf.Bytes(), &goList); errJUM != nil {
		return nil, errJUM
	}

	var pkgs map[string]struct{}

	if goListMap, goListIsMap := goList.(map[string]interface{}); goListIsMap {
		if deps, hasDeps := goListMap["Deps"]; hasDeps {
			if depsArray, depsIsArray := deps.([]interface{}); depsIsArray {
				pkgs = make(map[string]struct{}, len(depsArray))

				for _, pkg := range depsArray {
					if pkgString, pkgIsString := pkg.(string); pkgIsString {
						pkgs[pkgString] = struct{}{}
					} else {
						return nil, errGoListBadOut
					}
				}
			} else {
				return nil, errGoListBadOut
			}
		} else {
			pkgs = map[string]struct{}{}
		}
	} else {
		return nil, errGoListBadOut
	}

	return pkgs, nil
}
