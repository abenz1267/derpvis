package main

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/pborman/getopt/v2"
)

//nolint
var (
	add      string
	remove   int
	current  bool
	list     bool
	repolist string
	repos    []string

	PERMISSION_READ_WRITE = 600
)

//nolint
func init() {
	getopt.FlagLong(&add, "add", 'a', "", "folder to add to monitoring")
	getopt.FlagLong(&remove, "remove", 'r', "folder to remove from monitoring")
	getopt.Flag(&current, 'c', "use current folder")
	getopt.Flag(&list, 'l', "list folders")
}

func main() {
	createDatabase()
	parseFolders()
	syncEnvFolders()
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

func syncEnvFolders() {
	envVar := os.Getenv("DERPVIS_FOLDERS")
	folders := strings.Split(envVar, ":")

	for _, v := range folders {
		if !folderExists(v, false) {
			repos = append(repos, v)
		}
	}

	writeFolders()
}

func updateRepo(repo string) {
	if _, err := os.Stat(repo); os.IsNotExist(err) {
		log.Printf("Folder missing: %s\n", repo)

		return
	}

	cmd := exec.Command("git", "pull")
	cmd.Dir = repo

	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	log.Printf("%s: %s", repo, string(out))
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
		log.Printf("%d: %s\n", k+1, v)
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
		workingDir, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		if folderExists(workingDir, true) {
			return
		}

		repos = append(repos, workingDir)

		writeFolders()
	}

	if add != "" {
		repos = append(repos, add)

		writeFolders()
	}
}

func folderExists(f string, printMsg bool) bool {
	for _, v := range repos {
		if v == f {
			if printMsg {
				log.Println("Folder already exists!")
			}

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

	err = ioutil.WriteFile(repolist, b, fs.FileMode(PERMISSION_READ_WRITE))
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

	err = os.Mkdir(filepath.Join(cfgDir, "derpvis"), fs.FileMode(PERMISSION_READ_WRITE))
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(repolist, []byte("[]"), fs.FileMode(PERMISSION_READ_WRITE))
	if err != nil {
		panic(err)
	}
}
