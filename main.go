package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/pborman/getopt/v2"
)

var (
	add     string
	remove  int
	current bool
	list    bool
)

var (
	folderFile string
	folders    []string
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

	updateRepos()
}

func updateRepos() {
	cmd := exec.Command("git", "pull")

	for _, v := range folders {
		cmd.Dir = v
		out, err := cmd.Output()
		if err != nil {
			panic(err)
		}

		fmt.Printf("%s: %s", v, string(out))
	}
}

func removeFolder() {
	for k := range folders {
		if k+1 == remove {
			folders = removeIndex(folders, k)
		}
	}

	writeFolders()
}

func removeIndex(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}

func listFolders() {
	for k, v := range folders {
		fmt.Printf("%d: %s\n", k+1, v)
	}
}

func parseFolders() {
	b, err := ioutil.ReadFile(folderFile)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &folders)
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

		folders = append(folders, wd)

		writeFolders()
	}

	if add != "" {
		folders = append(folders, add)
		writeFolders()
	}
}

func folderExists(f string) bool {
	for _, v := range folders {
		if v == f {
			println("Folder already exists!")
			return true
		}
	}

	return false
}

func writeFolders() {
	sort.Strings(folders)

	b, err := json.Marshal(folders)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(folderFile, b, 0700)
	if err != nil {
		panic(err)
	}
}

func createDatabase() {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	folderFile = filepath.Join(cfgDir, "derpvis", "folders.json")
	if _, err = os.Stat(folderFile); !os.IsNotExist(err) {
		return
	}

	err = os.Mkdir(filepath.Join(cfgDir, "derpvis"), 0700)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(folderFile, []byte("[]"), 0700)
	if err != nil {
		panic(err)
	}
}
