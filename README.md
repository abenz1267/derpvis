# Derpvis - not smart, still helpful

Derpvis is a little tool that literally just runs "git pull" on a bunch of folders you specified. Also initially clones the repositories, if not present.

You can setup folders and their respective git source with a `DERPVIS_FOLDERS` envvar as well. Derpvis will add those folders to the list located in `~/.config/derpvis`

Example environment variable: `export DERPVIS_FOLDERS=$HOME/.config/nvim(git@github.com:abenz1267/nvim.git),$HOME/.config/kitty(git@github.com:abenz1267/kitty.git)`

Installation:
```
go install github.com/abenz1267/derpvis@latest
```

Commands:

```
-c adds the current folder you are in to the derpvis folder list
-a <folder> adds a given folder
-l lists all folders
-r <index> removes a given folder
```

Folders will be kept in sync with the possible environment variable as the source of truth.
