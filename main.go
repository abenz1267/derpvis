package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"

	"github.com/pborman/getopt/v2"
)

var (
	add     string
	remove  int
	current bool
	list    bool
)

var (
	repolist string
	repos    []string
)

func init() {
	getopt.FlagLong(&add, "add", 'a', "", "folder to add to monitoring")
	getopt.FlagLong(&remove, "remove", 'r', "folder to remove from monitoring")
	getopt.Flag(&current, 'c', "use current folder")
	getopt.Flag(&list, 'l', "list folders")
}

func main() {
	createDatabase()
	parseFolders()
	getopt.Parse()

	if add != "" || current {
		addFolder(current)
		return
	}

	if list {
		listFolders()
		return
	}

	if remove != 0 {
		removeFolder()
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(repos))

	for _, v := range repos {
		go func(repo string) {
			defer wg.Done()
			updateRepo(repo)
		}(v)
	}

	wg.Wait()
}

func updateRepo(r string) {
	cmd := exec.Command("git", "pull")
	cmd.Dir = r
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %s", r, string(out))
}

func removeFolder() {
	for k := range repos {
		if k+1 == remove {
			repos = removeIndex(repos, k)
		}
	}

	writeFolders()
}

func removeIndex(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}

func listFolders() {
	for k, v := range repos {
		fmt.Printf("%d: %s\n", k+1, v)
	}
}

func parseFolders() {
	b, err := ioutil.ReadFile(repolist)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &repos)
	if err != nil {
		panic(err)
	}
}

func addFolder(c bool) {
	if c {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		if folderExists(wd) {
			return
		}

		repos = append(repos, wd)

		writeFolders()
	}

	if add != "" {
		repos = append(repos, add)
		writeFolders()
	}
}

func folderExists(f string) bool {
	for _, v := range repos {
		if v == f {
			println("Folder already exists!")
			return true
		}
	}

	return false
}

func writeFolders() {
	sort.Strings(repos)

	b, err := json.Marshal(repos)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(repolist, b, 0700)
	if err != nil {
		panic(err)
	}
}

func createDatabase() {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	repolist = filepath.Join(cfgDir, "derpvis", "folders.json")
	if _, err = os.Stat(repolist); !os.IsNotExist(err) {
		return
	}

	err = os.Mkdir(filepath.Join(cfgDir, "derpvis"), 0700)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(repolist, []byte("[]"), 0700)
	if err != nil {
		panic(err)
	}
}
