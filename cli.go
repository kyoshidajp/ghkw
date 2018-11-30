package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/oauth2"

	"github.com/dustin/go-humanize"
	"github.com/github/hub/github"
	api "github.com/google/go-github/github"
	"github.com/mattn/go-colorable"
	"github.com/mitchellh/colorstring"
	"github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
)

const (
	// EnvDebug is environmental var to handle debug mode
	EnvDebug = "GHKW_DEBUG"
)

// Exit codes are in value that represnet an exit code for a paticular error
const (
	ExitCodeOK int = 0

	// Errors start at 10
	ExitCodeError = 10 + iota
	ExitCodeParseFlagsError
	ExitCodeBadArgs
)

// Ignore char in keyword
const IGNORE_KEYWORD_CHAR = ":"

// Debugf prints debug output when EnvDebug is given
func Debugf(format string, args ...interface{}) {
	if env := os.Getenv(EnvDebug); len(env) != 0 {
		log.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// PrintErrorf prints error message on console
func PrintErrorf(format string, args ...interface{}) {
	format = fmt.Sprintf("[red]%s[reset]\n", format)
	fmt.Fprint(colorable.NewColorableStderr(),
		colorstring.Color(fmt.Sprintf(format, args...)))
}

// CLI is the command line object
type CLI struct {
	outStream, errStream io.Writer
}

// Searcher is search keyword object
type Searcher struct {
	client            *api.Client
	repository        *api.Repository
	keywordsWithTotal map[string]int
	searchTerm        *SearchTerm
}

// SearchResult is searching result object
type SearchResult struct {
	Keyword string
	Total   int
	Done    chan struct{}
}

// PairList is list of Pair
type PairList []Pair

// Pair is key-value object
type Pair struct {
	key   string
	value int
}

// Run invokes the CLI with the given arguments
func (c *CLI) Run(args []string) int {
	var (
		debug     bool
		in        string
		language  string
		fork      string
		size      string
		path      string
		filename  string
		extension string
		user      string
		repo      string
		version   bool
	)
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprint(c.errStream, helpText)
	}
	flags.StringVar(&in, "in", "", "")
	flags.StringVar(&language, "language", "", "")
	flags.StringVar(&fork, "fork", "", "")
	flags.StringVar(&size, "size", "", "")
	flags.StringVar(&path, "path", "", "")
	flags.StringVar(&filename, "filename", "", "")
	flags.StringVar(&extension, "extension", "", "")
	flags.StringVar(&user, "user", "", "")
	flags.StringVar(&repo, "repo", "", "")
	flags.BoolVar(&debug, "debug", false, "")
	flags.BoolVar(&debug, "d", false, "")
	flags.BoolVar(&version, "version", false, "")
	flags.BoolVar(&version, "v", false, "")

	// Parse flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeParseFlagsError
	}

	if debug {
		os.Setenv(EnvDebug, "1")
		Debugf("Run as DEBUG mode")
	}

	if version {
		fmt.Fprintf(c.outStream, fmt.Sprintf("%s\n", Version))
		return ExitCodeOK
	}

	parsedArgs := flags.Args()
	if len(parsedArgs) == 0 {
		PrintErrorf("Invalid argument: You must set keyword.")
		return ExitCodeBadArgs
	}

	keywords := parsedArgs
	Debugf("keywords: %s", keywords)

	searchTerm := NewSearchTerm()
	searchTerm.in = in
	searchTerm.language = language
	searchTerm.fork = fork
	searchTerm.size = size
	searchTerm.path = path
	searchTerm.filename = filename
	searchTerm.extension = extension
	searchTerm.user = user
	searchTerm.repo = repo
	searchTerm.debugf()

	searcher, err := NewClient(keywords, *searchTerm)
	if err != nil {
		return ExitCodeError
	}

	status := searcher.search()
	if status != ExitCodeOK {
		return ExitCodeError
	}

	searcher.output(c.outStream)

	return ExitCodeOK
}

func (s *Searcher) keywords() []string {
	keys := make([]string, 0, len(s.keywordsWithTotal))
	for key := range s.keywordsWithTotal {
		keys = append(keys, key)
	}
	return keys
}

func (s *Searcher) searchRequest(res *SearchResult) {
	query := s.searchTerm.query(res.Keyword)
	Debugf("query: %s", query)

	result, response, err := s.client.Search.Code(context.Background(),
		query, nil)
	if err != nil {
		PrintErrorf("%s\n%s", response.Status, response.Body)
	}

	Debugf("keyword: %s (%d)", res.Keyword, *result.Total)
	res.Total = *result.Total
	res.Done <- struct{}{}
}

func (s *Searcher) search() int {
	keywords := s.keywords()
	ch := make(chan *SearchResult, len(keywords))

	for i := range keywords {
		res := &SearchResult{
			Keyword: keywords[i],
			Total:   0,
			Done:    make(chan struct{}),
		}
		ch <- res
		go s.searchRequest(res)
	}
	close(ch)

	for res := range ch {
		<-res.Done
		s.keywordsWithTotal[res.Keyword] = res.Total
	}

	return ExitCodeOK
}

func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].value > p[j].value }

func sortMapByValue(m map[string]int) PairList {
	p := make(PairList, len(m))
	i := 0
	for k, v := range m {
		p[i] = Pair{k, v}
		i++
	}
	sort.Sort(p)
	return p
}

func (s *Searcher) output(outStream io.Writer) {
	data := [][]string{}
	var prevRank, prevTotal int = -1, -1
	var _rank int
	for i, pl := range sortMapByValue(s.keywordsWithTotal) {
		if prevTotal == pl.value {
			_rank = prevRank
		} else {
			_rank = i + 1
			prevRank = _rank
		}
		prevTotal = pl.value

		rank := fmt.Sprintf("%d", _rank)
		keyword := pl.key
		total := fmt.Sprintf("%s", humanize.Comma(int64(pl.value)))
		data = append(data,
			[]string{rank, keyword, total})
	}

	table := tablewriter.NewWriter(outStream)
	table.SetHeader([]string{"Rank", "Keyword", "Total"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetColumnAlignment([]int{tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})
	table.SetCenterSeparator("|")
	table.AppendBulk(data)
	table.Render()
}

func getAccessTokenFromConf() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	confPath := filepath.Join(homeDir, ".config", "ghkw")
	err = os.Setenv("HUB_CONFIG", confPath)
	if err != nil {
		return "", err
	}

	c := github.CurrentConfig()
	host, err := c.DefaultHost()
	if err != nil {
		return "", err
	}

	return host.AccessToken, nil
}

func getAccessToken() (string, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		return token, nil
	}

	token, err := getAccessTokenFromConf()
	if err != nil {
		return "", err
	}

	return token, nil
}

func sanitizeKeyword(keyword string) string {
	return strings.Replace(keyword, IGNORE_KEYWORD_CHAR, "", -1)
}

// NewClient creates SearchClient
func NewClient(keywords []string, searchTerm SearchTerm) (*Searcher, error) {
	token, err := getAccessToken()
	if err != nil {
		return nil, err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	client := api.NewClient(tc)
	repo, _ := Repository(client)

	keywordsWithTotal := map[string]int{}
	for i := range keywords {
		keyword := sanitizeKeyword(keywords[i])
		keywordsWithTotal[keyword] = 0
	}

	return &Searcher{
		client:            client,
		repository:        repo,
		keywordsWithTotal: keywordsWithTotal,
		searchTerm:        &searchTerm,
	}, nil
}

// Repository returns api.Repository
func Repository(client *api.Client) (*api.Repository, error) {
	localRepo, err := github.LocalRepo()
	if err != nil {
		return nil, err
	}
	prj, err := localRepo.MainProject()
	if err != nil {
		return nil, err
	}

	repo, _, err := client.Repositories.Get(context.Background(), prj.Owner, prj.Name)
	if err != nil {
		PrintErrorf("Repository not found.\n%s", err)
		return nil, err
	}
	return repo, err
}

var helpText = `Usage: ghkw [options...] [keyword ...]

ghkw is a tool to know how many keyword is used in GitHub code.

You must specify keyword what you want to know keyword.

Options:

  -d, --debug    Enable debug mode.
                 Print debug log.

  -h, --help     Show this help message and exit.

  -v, --version  Print current version.

  Search Qualifiers:

    --in           Add in to search term.
  
    --language     Add language to search term.
  
    --fork         Add fork to search term.
  
    --size         Add size to search term.
  
    --path         Add path to search term.
  
    --filename     Add filename to search term.
  
    --extension    Add extension to search term.
  
    --user         Add user to search term.
  
    --repo         Add repo to search term.

    See Also:
      https://developer.github.com/v3/search/#parameters-2

Examples:
    The following is how to do ghkw search "exclude_condition" and "exclusion_condition" with search option in the file contents, language is javascript and file size is over 1,000bytes.

    ghkw --in=file --language=javascript --size=">1000" exclude_condition exclusion_condition
`
