# fzfweb

fzfweb is a program that impersonates the `fzf` binary.
Non-interactive calls (anything passing `-f`/`--filter`, or `--version`/`--help`) are forwarded to the real fzf unchanged.
Interactive calls are instead served as a local web page of checkboxes.
The real fzf still does all the actual query matching, so fzf's extended search syntax works exactly as it does normally.

## Why

fzf's terminal UI doesn't move the caret when you move to a different item, so a screen reader has nothing to track and you can't quickly tell what's currently highlighted.
This makes it unusable with a screen reader.
Pasting into fzf also tends to drop a lot of the characters you paste, making it hard to paste queries.

fzf is used for interactive task selection by Firefox's `mach try fuzzy` and `mach try perf`, and by other tools such as shell history search and various editor plugins.
fzfweb exists so any of those can be used with a screen reader, without losing fzf's query syntax or needing a bespoke web UI per caller.

fzfweb knows nothing about mach, tryselect or Firefox.
It just impersonates fzf, so it works anywhere fzf is used.

## Usage

Build it and place the resulting `fzf.exe` in a directory that appears on `$PATH` before your real fzf installation.
Anything that then runs `fzf` interactively will open `http://127.0.0.1:5039/` in your browser instead of drawing a terminal UI.

```
go build -o fzf.exe
```

fzfweb needs to find the real fzf binary to do the actual matching.
By default it looks for it at `~/.mozbuild/fzf/fzf.exe`, which is where Firefox's build bootstrap installs it.
Set `REAL_FZF` to point at a different fzf binary if yours lives elsewhere.

## Status

Implemented and verified against `mach try fuzzy --no-push` and `mach try perf --no-push` in a Firefox checkout.
