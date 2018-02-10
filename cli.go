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
	"time"

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
		language  string
		filename  string
		extension string
		version   bool
	)
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprint(c.errStream, helpText)
	}
	flags.StringVar(&language, "language", "", "")
	flags.StringVar(&filename, "filename", "", "")
	flags.StringVar(&extension, "extension", "", "")
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
	searchTerm.language = language
	searchTerm.filename = filename
	searchTerm.extension = extension
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

func (s *Searcher) searchRequest(keyword string, ch chan int) {
	query := s.searchTerm.query(keyword)
	Debugf("query: %s", query)

	result, response, err := s.client.Search.Code(context.Background(),
		query, nil)
	if err != nil {
		PrintErrorf("%s\n%s", response.Status, response.Body)
	}

	Debugf("keyword: %s (%d)", keyword, *result.Total)
	ch <- *result.Total
}

func (s *Searcher) search() int {
	ch := make(chan int)
	keywords := s.keywords()

	for i := range keywords {
		keyword := keywords[i]
		go s.searchRequest(keyword, ch)
		s.keywordsWithTotal[keyword] = <-ch
	}

	time.Sleep(1 * time.Second)

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
		keyword := keywords[i]
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

  --language     Add language to search term.

  --filename     Add filename to search term.

  --extension    Add extension to search term.

  -d, --debug    Enable debug mode.
                 Print debug log.

  -h, --help     Show this help message and exit.

  -v, --version  Print current version.
`
