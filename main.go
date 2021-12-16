package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	flag.Usage = func() {
		h := []string{
			"Filter out domains/urls not in scope",
			"",
			"Options:",
			"  -pwd <path>       The workspace (default current workspace if not specified)",
			"  -osp, --out-scope Show out-scope items only",
			"",
		}

		fmt.Print(strings.Join(h, "\n"))
	}
}

func main() {

	pwd, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error setting workspace to %s : %s\n", pwd, err)
	}

	flag.StringVar(&pwd, "pwd", pwd, "")

	var showOutScope bool
	var isInScope bool
	flag.BoolVar(&showOutScope, "osp", false, "")
	flag.BoolVar(&showOutScope, "out-scope", false, "")

	flag.Parse()

	sf, err := openScopefile(pwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening scope file: %s\n", err)
		return
	}

	checker, err := newScopeChecker(sf)
	sf.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing scope file: %s\n", err)
		return
	}

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		domain := strings.TrimSpace(sc.Text())

		isInScope = checker.inScope(domain)

		if (isInScope && !showOutScope) || (!isInScope && showOutScope) {
			fmt.Println(domain)
		}
	}
}

type scopeChecker struct {
	patterns     []*regexp.Regexp
	antipatterns []*regexp.Regexp
}

func newScopeChecker(r io.Reader) (*scopeChecker, error) {
	sc := bufio.NewScanner(r)
	s := &scopeChecker{
		patterns: make([]*regexp.Regexp, 0),
	}

	for sc.Scan() {
		p := strings.TrimSpace(sc.Text())
		if p == "" {
			continue
		}

		isAnti := false
		if p[0] == '!' {
			isAnti = true
			p = p[1:]
		}

		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}

		if isAnti {
			s.antipatterns = append(s.antipatterns, re)
		} else {
			s.patterns = append(s.patterns, re)
		}
	}

	return s, nil
}

func (s *scopeChecker) inScope(domain string) bool {

	// if it's a URL pull the hostname out to avoid matching
	// on part of the path or something like that
	if isURL(domain) {
		var err error
		domain, err = getHostname(domain)
		if err != nil {
			return false
		}
	}

	inScope := false
	for _, p := range s.patterns {
		if p.MatchString(domain) {
			inScope = true
			break
		}
	}

	for _, p := range s.antipatterns {
		if p.MatchString(domain) {
			return false
		}
	}
	return inScope
}

func getHostname(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}

func isURL(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))

	if len(s) < 6 {
		return false
	}

	return s[:5] == "http:" || s[:6] == "https:"
}

func openScopefile(pwd string) (io.ReadCloser, error) {
	filePath := filepath.Join(pwd, "scope.txt")
	f, err := os.Open(filePath)

	if err == nil {
		return f, nil
	}

	return nil, fmt.Errorf("unable to find SCOPE file in %s", filePath)
}
