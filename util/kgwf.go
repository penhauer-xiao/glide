package util

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	prefix_ver     = ".v"
	separated_ver  = "."
	separated_path = "/"
)

var (
	createGitParseVersion = regexp.MustCompile(`(?m-s)(?:tags)/(\S+)$`)
)

func splitPkgVer(pkgVer string) (v []string) {
	if strings.Contains(pkgVer, prefix_ver) {
		return strings.Split(pkgVer, separated_ver)
	}
	return v
}

/*
gopkg.in/user/pkg.v3                 → github.com/user/pkg                 (branch/tag v3, v3.N, or v3.N.M)
gopkg.in/user/pkg.v3/sub_pkg         → github.com/user/pkg/sub_pkg         (branch/tag v3, v3.N, or v3.N.M)
gopkg.in/user/pkg.v3/sub_pkg/sub_pkg → github.com/user/pkg/sub_pkg/sub_pkg (branch/tag v3, v3.N, or v3.N.M)
*/
func gopkg_in_user(remote string) (newRepository, branch string) {
	if s := strings.Split(remote, separated_path); len(s) >= 3 {
		pkgVer := splitPkgVer(s[2])
		newRepository = fmt.Sprintf("https://%s/%s/%s", s[0], s[1], pkgVer[0])
		logrus.Warnln("github.com/user/pkg Mode:", remote, newRepository, pkgVer[1])
		return newRepository, pkgVer[1]
	}
	return "", ""
}

/*
gopkg.in/pkg.v3                 → github.com/go-pkg/pkg                  (branch/tag v3, v3.N, or v3.N.M)
gopkg.in/pkg.v3/sub_pkg         → github.com/go-pkg/pkg/sub_pkg          (branch/tag v3, v3.N, or v3.N.M)
gopkg.in/pkg.v3/sub_pkg/sub_pkg → github.com/go-pkg/pkg/sub_pkg/sub_pkg  (branch/tag v3, v3.N, or v3.N.M)
*/
func gopkg_in(url string, maps map[string]interface{}) (newRepository, branch string) {
	Package, ok1 := maps["package"]
	repo, ok2 := maps["repo"]
	rule, ok3 := maps["rule"]

	if !ok1 || !ok2 || !ok3 {
		return "", ""
	}

	if !rule.(bool) {
		return "", ""
	}

	if !strings.Contains(url, Package.(string)) {
		return "", ""
	}

	remote := strings.Replace(url, Package.(string), repo.(string), -1)
	var s []string
	if s = strings.Split(remote, separated_path); len(s) < 2 {
		return "", ""
	}

	pkgVer := splitPkgVer(s[1])
	if len(pkgVer) != 0 {
		newRepository = fmt.Sprintf("https://%s/go-%s/%s", s[0], pkgVer[0], pkgVer[0])
		logrus.Warnln("github.com/go-pkg/pkg Mode:", url, newRepository, pkgVer[1])
		return newRepository, pkgVer[1]
	}

	return gopkg_in_user(remote)
}

func other_in(url string, maps map[string]interface{}) (newRepository, branch string) {
	rawUrl, ok1 := maps["package"]
	newUrl, ok2 := maps["repo"]
	if !ok1 || !ok2 {
		return "", ""
	}

	if !strings.Contains(url, rawUrl.(string)) {
		return "", ""
	}

	newRepository = strings.Replace(url, rawUrl.(string), newUrl.(string), -1)
	s := strings.Split(newRepository, separated_path)
	if len(s) < 3 {
		return "", ""
	}

	// t := separated_path + s[2]
	// baseUrl := strings.Split(newRepository, t)[0]
	newRepository = fmt.Sprintf("https://%s/%s/%s", s[0], s[1], s[2])

	logrus.Warnln("KGWF Mode:", url, newRepository)
	return newRepository, ""
}

func mtoGWF(remote string, maps interface{}) (newRepository, branch string) {
	m := maps.(map[string]interface{})
	old_url, ok1 := m["old_url"]
	old_paths, ok2 := m["old_paths"]
	new_url, ok3 := m["new_url"]

	if !ok1 || !ok2 || !ok3 {
		return remote, ""
	}

	for _, old_path := range old_paths.([]interface{}) {
		repo := old_url.(string) + old_path.(string)
		if strings.Contains(remote, repo) {
			s := strings.Split(repo, separated_path)
			return `https://` + new_url.(string) + s[len(s)-1], ""
		}
	}
	return remote, ""
}

func otomGWF(remote string, maps interface{}) (newRepository, branch string) {
	m := maps.(map[string]interface{})
	old_url, ok1 := m["old_url"]
	new_url, ok2 := m["new_url"]
	new_paths, ok3 := m["new_paths"]

	if !ok1 || !ok2 || !ok3 {
		return remote, ""
	}

	if !strings.Contains(remote, old_url.(string)) {
		return remote, ""
	}

	for _, path := range new_paths.([]interface{}) {
		repo := old_url.(string) + path.(string)
		if strings.Contains(remote, repo) {
			return `https://` + new_url.(string) + path.(string), ""
		}
	}

	rs := strings.Split(remote, separated_path)
	if len(rs) < 2 {
		return remote, ""
	}

	for _, path := range new_paths.([]interface{}) {
		if strings.Contains(path.(string), rs[1]) {
			ps := strings.SplitN(path.(string), separated_path, 2)
			newRepository = strings.Replace(remote, old_url.(string), new_url.(string), -1)
			newRepository = strings.Replace(newRepository, rs[1], ps[0], -1)
			return `https://` + newRepository, ""
		}
	}

	return remote, ""
}

func fixGWF(remote string, maps interface{}) (newRepository, branch string) {
	m := maps.(map[string]interface{})
	old_url, ok1 := m["old_url"]
	new_url, ok2 := m["new_url"]
	fix, ok3 := m["xfix"]

	if !ok1 || !ok2 || !ok3 {
		return remote, ""
	}

	if !strings.Contains(remote, old_url.(string)) {
		return remote, ""
	}

	rs := strings.Split(remote, separated_path)
	if len(rs) < 2 {
		return remote, ""
	}

	if fix.(string)[0:1] == "-" {
		return `https://` + new_url.(string) + rs[1] + fix.(string), ""
	}

	if fix.(string)[len(fix.(string))-1:] == "-" {
		return `https://` + new_url.(string) + fix.(string) + rs[1], ""
	}

	return remote, ""
}

func machSSH(remote string, maps interface{}) (newRepository, branch string) {
	m := maps.(map[string]interface{})
	old_url, ok1 := m["old_url"]

	if !ok1 {
		return remote, ""
	}

	if !strings.Contains(remote, old_url.(string)) {
		return remote, ""
	}

	paths := strings.Split(remote, `/`)
	if len(paths) < 3 {
		return remote, ""
	}
	newRepository = fmt.Sprintf("git@git.%s:%s/%s.git", paths[0], paths[1], paths[2])
	return
}

func KGWF(remote string) (newRepository, branch string) {

	if viper.Get("gwf_SSH") != nil {
		for _, m := range viper.Get("gwf_SSH").([]interface{}) {
			if newRepository, _ := machSSH(remote, m); newRepository != remote {
				logrus.Warnln("gwfSSH Mode:", remote, newRepository)
				return newRepository, ""
			}
		}
	}

	if viper.Get("gwf_fix") != nil {
		for _, m := range viper.Get("gwf_fix").([]interface{}) {
			if newRepository, _ := fixGWF(remote, m); newRepository != remote {
				logrus.Warnln("fixGWF Mode:", remote, newRepository)
				return newRepository, ""
			}
		}
	}

	if viper.Get("gwf_mto") != nil {
		for _, m := range viper.Get("gwf_mto").([]interface{}) {
			if newRepository, _ := mtoGWF(remote, m); newRepository != remote {
				logrus.Warnln("mtoGWF Mode:", remote, newRepository)
				return newRepository, ""
			}
		}
	}

	if viper.Get("gwf_otom") != nil {
		for _, m := range viper.Get("gwf_otom").([]interface{}) {
			if newRepository, _ := otomGWF(remote, m); newRepository != remote {
				logrus.Warnln("otomGWF Mode:", remote, newRepository)
				return newRepository, ""
			}
		}
	}

	if viper.Get("gwf") != nil {
		for _, m := range viper.Get("gwf").([]interface{}) {
			maps := m.(map[string]interface{})
			if newRepository, branch = gopkg_in(remote, maps); newRepository != "" && branch != "" {
				return newRepository, branch
			}

			if newRepository, branch = other_in(remote, maps); newRepository != "" {
				return newRepository, branch
			}
		}
	}

	return remote, ""
}

func isDigit(w string) bool {
	if w == "" {
		return false
	}
	for _, r := range w {
		if !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

func machSubpackagesSSH(remote string, maps interface{}) (newRepository, subPackage string) {
	m := maps.(map[string]interface{})
	old_url, ok1 := m["old_url"]

	if !ok1 {
		return remote, ""
	}

	if !strings.Contains(remote, old_url.(string)) {
		return remote, ""
	}

	paths := strings.Split(remote, `/`)
	if len(paths) < 4 {
		return remote, ""
	}
	return strings.Join(paths[:3], `/`), strings.Join(paths[3:], `/`)
}

func GetSubpackages(repo string) (name, subPackage string) {
	if strings.Contains(repo, `gopkg.in/`) {
		if x := strings.Index(repo, prefix_ver); x != -1 {
			start, end := repo[:x], repo[x+2:]
			if y := strings.Index(end, `/`); y != -1 {
				ver := end[:y]
				if isDigit(ver) {
					name = start
					subPackage = end[y+1:]
					logrus.Warnln("------------>", repo, name, subPackage)
					return
				}
			}
		}
		return
	}

	if viper.Get("gwf_sub") != nil {
		for _, m := range viper.Get("gwf_sub").([]interface{}) {
			mv := m.(map[string]interface{})
			for k, v := range mv {
				index, _ := strconv.Atoi(k)
				for _, vv := range v.([]interface{}) {
					if strings.Contains(repo, vv.(string)) {
						if v1 := strings.Split(repo, `/`); len(v1) > index {
							name = strings.Join(v1[0:index], `/`)
							subPackage = strings.Join(v1[index:], `/`)
							logrus.Warnln("------------>", repo, name, subPackage)
							return
						}
					}
				}
			}
		}
	}

	if viper.Get("gwf_SSH") != nil {
		for _, m := range viper.Get("gwf_SSH").([]interface{}) {
			if name, subPackage = machSubpackagesSSH(repo, m); name != repo {
				logrus.Warnln("------------>", repo, name, subPackage)
				return
			}
		}
	}
	return "", ""
}

func GetTagsVersion(remote string) <-chan string {
	ch := make(chan string)
	go func() {
		out, err := exec.Command("git", "ls-remote", remote).CombinedOutput()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				ti := strings.TrimSpace(line)
				if found := createGitParseVersion.FindString(ti); found != "" {
					tg := strings.TrimPrefix(strings.TrimSuffix(found, "^{}"), "tags/")
					ch <- tg
				}
			}
		}
		close(ch)
	}()
	return ch
}
