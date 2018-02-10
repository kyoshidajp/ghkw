# ghkw

[![GitHub release](https://img.shields.io/github/release/kyoshidajp/ghkw.svg?style=flat-square)][release]
[![Travis](https://travis-ci.org/kyoshidajp/ghkw.svg?branch=master)](https://travis-ci.org/kyoshidajp/ghkw)
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]

[release]: https://github.com/kyoshidajp/ghkw/releases
[license]: https://github.com/kyoshidajp/ghkw/blob/master/LICENSE
[godocs]: http://godoc.org/github.com/kyoshidajp/ghkw

**ghkw** is **G**it**H**ub **K**ey**W**ord.

Search how many keywords in GitHub Code by GitHub API.

## Usage

```
$ ghkw [options...] [keyword ...]
```

### Example

Output markdown format.

```
$ ghkw exclusion_condition exclude_condition excluded_condition
| RANK |       KEYWORD       | TOTAL |
|------|---------------------|-------|
|    1 | exclude_condition   |   272 |
|    2 | exclusion_condition |    64 |
|    3 | excluded_condition  |     2 |
```

### Options

```
--language     Add language to search term.

--filename     Add filename to search term.

--extension    Add extension to search term.

--user         Add user to search term.

--repo         Add repo to search term.

-d, --debug    Enable debug mode.
               Print debug log.

-h, --help     Show this help message and exit.

-v, --version  Print current version.
```

*NOTE*: Set Github Access Token which has "Full control of private repositories" scope as an environment variable `GITHUB_TOKEN`. If not set, `ghkw` requires your Github username and password(and two-factor auth code if you are setting). Because of using [GitHub API v3](https://developer.github.com/v3/).

## Install

### Homebrew

If you have already installed [Homebrew](http://brew.sh/); then can install by brew command.

```
$ brew tap kyoshidajp/ghkw
$ brew install ghkw
```

### go get

If you are a Golang developper/user; then execute `go get`.

```
$ go get -u github.com/kyoshidajp/ghkw
```

### Manual

1. Download binary which meets your system from [Releases](release).
1. Unarchive it.
1. Put `ghkw` where you want.
1. Add `ghkw` path to `$PATH`.

## Author

[Katsuhiko YOSHIDA](https://github.com/kyoshidajp)
