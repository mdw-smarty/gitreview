package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	GitFetch          bool
	GitRepositoryRoot string
	GitGUILauncher    string
	OutputFilePath    string
}

func ReadConfig(version string) *Config {
	log.SetFlags(log.Ltime | log.Lshortfile)

	config := new(Config)

	flags := flag.NewFlagSet(fmt.Sprintf("gitreview @ %s", version), flag.ExitOnError)

	flags.Usage = func() {
		_, _ = fmt.Fprintf(flags.Output(), "Usage of %s:\n\n", flags.Name())
		_, _ = fmt.Fprintf(flags.Output(), "%s\n\n```\n", doc)
		flags.PrintDefaults()
		_, _ = fmt.Fprintln(flags.Output(), "```")
	}

	flags.StringVar(&config.GitGUILauncher,
		"gui", "smerge", ""+
			"The external git GUI application to use for visual reviews.\n"+
			"-->",
	)

	flags.StringVar(&config.OutputFilePath,
		"outfile", "SMARTY_REVIEW_LOG", ""+
			"The path or name of the environment variable containing the\n"+
			"path to your pre-existing code review file. If the file exists\n"+
			"the final log entry will be appended to that file instead of stdout.\n"+
			"-->",
	)

	flags.BoolVar(&config.GitFetch,
		"fetch", true, ""+
			"When false, suppress all git fetch operations via --dry-run.\n"+
			"Repositories with updates will still be included in the review.\n"+
			"-->",
	)

	_ = flags.Parse(os.Args[1:])

	config.GitRepositoryRoot = rootDir(flags.Arg(0))

	if !config.GitFetch {
		log.Println("Running git fetch with --dry-run (updated repositories will not be reviewed).")
		gitFetchCommand += " --dry-run"
	}

	return config
}

func rootDir(pathFlag string) string {
	if len(pathFlag) > 0 {
		root, err := filepath.Abs(pathFlag)
		if err != nil {
			log.Fatal(err)
		}
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(home, "src")
}

func (this *Config) OpenOutputWriter() io.WriteCloser {
	this.OutputFilePath = strings.TrimSpace(this.OutputFilePath)
	if this.OutputFilePath == "" {
		log.Println("Final report will be written to stdout.")
		return os.Stdout
	}

	path, found := os.LookupEnv(this.OutputFilePath)
	if found {
		log.Printf("Found output path in environment variable: %s=%s", this.OutputFilePath, path)
	} else {
		path = this.OutputFilePath
	}

	stat, err := os.Stat(path)
	if err == nil && !errors.Is(err, os.ErrNotExist) {
		file, err2 := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, stat.Mode())
		if err2 == nil {
			log.Println("Final report will be appended to", path)
			return file
		} else {
			log.Printf("Could not open file for appending: [%s] Error: %v", this.OutputFilePath, err2)
		}
	}

	log.Println("Final report will be written to stdout.")
	return os.Stdout
}

const rawDoc = `

# gitreview

gitreview facilitates visual inspection (code review) of git
repositories that meet any of the following criteria:

1. New content was fetched
2. Behind origin/<default-branch>
3. Ahead of origin/<default-branch>
4. Messy (have uncommitted state)
5. Throw errors for the required git operations (listed below)

We use variants of the following commands to ascertain the
status of each repository:

- ''git remote''           (shows remote address)
- ''git status''           (shows uncommitted files)
- ''git fetch''            (finds new commits/tags/branches)
- ''git rev-list''         (lists commits behind/ahead-of <default-branch>)
- ''git config --get ...'' (show config parameters of a repo)

...all of which should be safe enough. 

Each repository that meets any criteria above will be
presented for review. After all reviews are complete a
concatenated report of all output from ''git fetch'' for
repositories that were behind their origin is printed to
stdout. Only repositories with "smarty" in their
path are included in this report.

Repositories are scanned recursively from the working directory.

Installation:

    go get -u github.com/smarty/gitreview


Skipping Repositories:

If you have repositories in your list that you would rather not review,
you can mark them to be skipped by adding a config variable to the
repository. The following command will produce this result:

    git config --add review.skip true


Omitting Repositories:

If you have repositories in your list that you would still like to audit
but aren't responsible to sign off (it's code from another team), you can 
mark them to be omitted from the final report by adding a config variable
to the repository. The following command will produce this result:

    git config --add review.omit true


CLI Flags:
`

var doc = strings.ReplaceAll(strings.TrimSpace(rawDoc), "''", "`")
