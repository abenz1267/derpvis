package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/pborman/getopt/v2"
)

// nolint
var (
	add      string
	remove   int
	current  bool
	list     bool
	push     bool
	repolist string
	repos    []Repo

	PermissionReadWrite = 0o777
)

type Repo struct {
	Folder string `json:"folder"`
	Source string `json:"source"`
}

// nolint
func init() {
	getopt.FlagLong(&add, "add", 'a', "", "folder to add to monitoring")
	getopt.FlagLong(&remove, "remove", 'r', "folder to remove from monitoring")
	getopt.Flag(&current, 'c', "use current folder")
	getopt.Flag(&push, 'p', "push changes")
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

	if push {
		pushAll()

		return
	}

	pullAll()
}

func pushAll() {
	for _, repo := range repos {
		r, err := git.PlainOpen(repo.Folder)
		if err != nil {
			log.Println(err)
			return
		}

		w, err := r.Worktree()
		if err != nil {
			log.Println(err)
			return
		}

		s, err := w.Status()
		if err != nil {
			log.Println(err)
			return
		}

		if !s.IsClean() {
			w.AddGlob(".")

			var message string

			fmt.Printf("Enter commit message for %s: ", repo.Folder)

			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				message = strings.TrimSpace(scanner.Text())

				if message != "" {
					break
				}

				fmt.Printf("Enter commit message for %s: ", repo.Folder)
			}

			_, err := w.Commit(message, &git.CommitOptions{})
			if err != nil {
				log.Println(err)
				return
			}

			err = r.Push(&git.PushOptions{})
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func pullAll() {
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

func clone(repo Repo) {
	_, err := git.PlainClone(repo.Folder, false, &git.CloneOptions{
		URL:      repo.Source,
		Progress: os.Stdout,
	})
	if err != nil {
		panic(err)
	}
}

func updateRepo(repo Repo) {
	if _, err := os.Stat(repo.Folder); os.IsNotExist(err) {
		log.Printf("Folder missing: %s\n", repo.Folder)

		clone(repo)

		return
	}

	r, err := git.PlainOpen(repo.Folder)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			os.RemoveAll(repo.Folder)

			clone(repo)

			return
		}

		panic(err)
	}

	w, err := r.Worktree()
	if err != nil {
		panic(err)
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil {
		log.Printf("%s: %s", repo.Folder, err)
		return
	}

	ref, err := r.Head()
	if err != nil {
		panic(err)
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		panic(err)
	}

	log.Printf("%s: %s", repo.Folder, commit.Message)
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

	r, err := git.PlainOpen(dir)
	if err != nil {
		panic(err)
	}

	remote, err := r.Remote("origin")
	if err != nil {
		panic(err)
	}

	return remote.Config().URLs[0]
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

	err = ioutil.WriteFile(repolist, b, fs.FileMode(PermissionReadWrite))
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

	err = os.Mkdir(filepath.Join(cfgDir, "derpvis"), fs.FileMode(PermissionReadWrite))
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(repolist, []byte("[]"), fs.FileMode(PermissionReadWrite))
	if err != nil {
		panic(err)
	}
}
