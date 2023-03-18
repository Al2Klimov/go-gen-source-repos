// SPDX-License-Identifier: GPL-3.0-or-later

package source_repos

import (
	"runtime/debug"
	"sort"
	"strings"
)

func GetLinks() []string {
	unique := map[string]struct{}{"https://github.com/golang/go": {}}

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		for _, modules := range [][]*debug.Module{{&buildInfo.Main}, buildInfo.Deps} {
			for _, module := range modules {
				if steps := strings.SplitN(module.Path, "/", 4); strings.Contains(steps[0], ".") {
					if steps[0] == "github.com" {
						unique[strings.TrimRight("https://"+strings.Join(steps[:3], "/"), "/")] = struct{}{}
					} else {
						unique["https://pkg.go.dev/"+module.Path] = struct{}{}
					}
				}
			}
		}
	}

	links := make([]string, 0, len(unique))
	for ul := range unique {
		links = append(links, ul)
	}

	sort.Strings(links)
	return links
}
