package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	repos    []Repo

	PERMISSION_READ_WRITE = 0o777
	GIT                   = "/usr/bin/git"
)

type Repo struct {
	Folder string `json:"folder"`
	Source string `json:"source"`
}

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
		addFolder()

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
		go func(repo Repo) {
			defer wg.Done()
			updateRepo(repo)
		}(v)
	}

	wg.Wait()
}

func syncEnvFolders() {
	envVar, exists := os.LookupEnv("DERPVIS_FOLDERS")

	if !exists {
		return
	}

	folders := strings.Split(envVar, ",")

	for _, v := range folders {
		arrStr := strings.Split(v, "(")
		if len(arrStr) != 2 {
			log.Fatalf("Missing source repository for: %s. Cloning...", arrStr[0])
		}

		folder := arrStr[0]
		src := strings.Trim(arrStr[1], ")")

		repo := Repo{
			Folder: folder,
			Source: src,
		}

		if !folderExists(repo, false) {
			repos = append(repos, repo)
		}
	}

	writeFolders()
}

func updateRepo(repo Repo) {
	if _, err := os.Stat(repo.Folder); os.IsNotExist(err) {
		log.Printf("Folder missing: %s\n", repo.Folder)

		cmd := exec.Command(GIT, "clone", repo.Source, repo.Folder)

		err := cmd.Run()
		if err != nil {
			panic(err)
		}

		return
	}

	cmd := exec.Command(GIT, "pull")
	cmd.Dir = repo.Folder

	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	log.Printf("%s: %s", repo.Folder, string(out))
}

func removeFolder() {
	for k := range repos {
		if k+1 == remove {
			repos = removeIndex(repos, k)
		}
	}

	writeFolders()
}

func removeIndex(s []Repo, index int) []Repo {
	return append(s[:index], s[index+1:]...)
}

func listFolders() {
	for k, v := range repos {
		//nolint
		fmt.Printf("%d: %s (%s)\n", k+1, v.Folder, v.Source)
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

func addFolder() {
	if current {
		workingDir, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		repo := Repo{
			Folder: workingDir,
			Source: getRemoteOrigin(workingDir),
		}

		if folderExists(repo, true) {
			return
		}

		repos = append(repos, repo)

		writeFolders()

		return
	}

	if add != "" {
		repo := Repo{
			Folder: add,
			Source: getRemoteOrigin(add),
		}

		repos = append(repos, repo)

		writeFolders()

		return
	}
}

func getRemoteOrigin(dir string) string {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Fatalf("Folder doesn't exist: %s", dir)
	}

	cmd := exec.Command(GIT, "config", "--get", "remote.origin.url")
	cmd.Dir = dir

	res, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	return string(res)
}

func folderExists(repo Repo, printMsg bool) bool {
	for i, v := range repos {
		if v.Folder == repo.Folder {
			if printMsg {
				log.Println("Folder already exists!")
			}

			repos[i].Source = repo.Source
			return true
		}
	}

	return false
}

func writeFolders() {
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
