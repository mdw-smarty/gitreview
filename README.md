Usage of gitreview @ dev:

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

- `git remote`           (shows remote address)
- `git status`           (shows uncommitted files)
- `git fetch`            (finds new commits/tags/branches)
- `git rev-list`         (lists commits behind/ahead-of <default-branch>)
- `git config --get ...` (show config parameters of a repo)

...all of which should be safe enough. 

Each repository that meets any criteria above will be
presented for review. After all reviews are complete a
concatenated report of all output from `git fetch` for
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

```
  -fetch
    	When false, suppress all git fetch operations via --dry-run.
    	Repositories with updates will still be included in the review.
    	--> (default true)
  -gui string
    	The external git GUI application to use for visual reviews.
    	--> (default "smerge")
  -outfile string
    	The path or name of the environment variable containing the
    	path to your pre-existing code review file. If the file exists
    	the final log entry will be appended to that file instead of stdout.
    	--> (default "SMARTY_REVIEW_LOG")
```
