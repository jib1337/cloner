# cloner | Jack Nelson

## Features
- Clone a webpage for whatever purpose you wish
- Substitute in your own form action URLs for POST data
- Sometimes works, sometimes doesn't (just to keep things exciting)

## Usage
```
============================================================
                           cloner
============================================================
  -f string
        The URL of the site to replace in form actions
  -o string
        Output location (default ".\\")
  -u string
        The URL of the site to clone
```

## Todo
- Reduce the number of requests needed (should always be 1 per file)
- Refactor how the parser works
- Figure out a less bad way to replace links