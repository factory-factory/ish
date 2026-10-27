package parse

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/factory-factory/ish/models"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type Byline struct {
	Author      *string
	DateCreated *string
}

// func main() {
// 	issue := InitialModel("01AN4Z07BY")
// 	issue.Print()
// }

// IssuesDir is the directory issue files are read from.
const IssuesDir = "./.ish"

// InitialModel loads and parses the issue with the given ID from the issues
// directory.
func InitialModel(issueId string) models.Issue {
	issue, err := LoadIssue(IssuesDir, issueId)

	if err != nil {
		log.Fatal(err)
	}

	return *issue
}

// LoadIssue reads and parses the issue with the given ID from dir.
func LoadIssue(dir, issueId string) (*models.Issue, error) {
	path := filepath.Join(dir, issueId+".md")
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	issue, err := ParseIssue(file)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	issue.Id = issueId

	return issue, nil
}

// ListIssues reads and parses every issue file in dir, ordered by ID.
func ListIssues(dir string) ([]models.Issue, error) {
	entries, err := os.ReadDir(dir)

	if err != nil {
		return nil, err
	}

	issues := make([]models.Issue, 0, len(entries))

	// ReadDir returns entries sorted by file name, and so by issue ID.
	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || filepath.Ext(name) != ".md" {
			continue
		}

		issue, err := LoadIssue(dir, strings.TrimSuffix(name, ".md"))

		if err != nil {
			return nil, err
		}

		issues = append(issues, *issue)
	}

	return issues, nil
}

// markdown parses issue files with the same extensions glamour uses to render
// them.
var markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.DefinitionList,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
)

func ParseIssue(r io.Reader) (*models.Issue, error) {
	source, err := io.ReadAll(r)

	if err != nil {
		return nil, err
	}

	document := markdown.Parser().Parse(text.NewReader(source))

	return parseIssue(document, source)
}

func parseIssue(document ast.Node, source []byte) (*models.Issue, error) {
	node := document.FirstChild()

	title, err := matchTitle(node, source)

	if err != nil {
		return nil, err
	}

	node = node.NextSibling()
	byline, err := matchByline(node, source)

	if err != nil {
		return nil, err
	}

	// The metadata list is optional.
	node = node.NextSibling()
	metadata := []models.Tag{}

	if _, ok := node.(*ast.List); ok {
		metadata, err = matchMetadata(node, source)

		if err != nil {
			return nil, err
		}

		node = node.NextSibling()
	}

	issue := &models.Issue{
		Title:        *title,
		Author:       *byline.Author,
		DateCreated:  *byline.DateCreated,
		Metadata:     metadata,
		Conversation: []models.Card{},
	}

	for node != nil {
		card, next, err := parseCard(node, source)

		if err != nil {
			return nil, err
		}

		issue.Conversation = append(issue.Conversation, *card)
		node = next
	}

	return issue, nil
}

// parseCard parses the card that begins with the horizontal rule at node,
// returning the card and the horizontal rule that begins the next card, if
// any. The card's content is copied verbatim from the source, so it keeps
// exactly the markdown that was written.
func parseCard(node ast.Node, source []byte) (*models.Card, ast.Node, error) {
	err := matchBreak(node, source)

	if err != nil {
		return nil, nil, err
	}

	node = node.NextSibling()
	byline, err := matchByline(node, source)

	if err != nil {
		return nil, nil, err
	}

	lines := node.Lines()
	start := lines.At(lines.Len() - 1).Stop
	end := len(source)

	next := node.NextSibling()

	for next != nil && next.Kind() != ast.KindThematicBreak {
		next = next.NextSibling()
	}

	if next != nil {
		end = next.Pos()
	}

	content := trimContent(string(source[start:end]))

	card := &models.Card{
		Author:      *byline.Author,
		DateCreated: *byline.DateCreated,
		Content:     &content,
	}

	return card, next, nil
}

// trimContent trims blank lines around card content, but keeps the
// indentation of its first line, which may be significant.
func trimContent(content string) string {
	content = strings.TrimLeft(content, "\r\n")
	return strings.TrimRightFunc(content, unicode.IsSpace)
}

// lineBreak matches a line break and the spaces and tabs around it, including
// the two trailing spaces of a markdown hard line break, and any line breaks
// after it.
var lineBreak = regexp.MustCompile(`[ \t]*[\r\n][ \t\r\n]*`)

// singleLine returns s with each line break, and the space around it, replaced
// with a space, and each tab replaced with a space, so a title or attribute
// value can be shown on one line and in a tab-separated column. The issue file
// itself is left as it's written.
func singleLine(s string) string {
	return strings.ReplaceAll(lineBreak.ReplaceAllString(s, " "), "\t", " ")
}

func matchTitle(n ast.Node, b []byte) (*string, error) {
	switch n.(type) {
	case *ast.Heading:
		str := singleLine(string(n.Text(b)))
		return &str, nil
	default:
		return nil, errors.New("Expected a heading")
	}
}

// bylinePattern matches a byline: an author and a timestamp in brackets.
var bylinePattern = regexp.MustCompile(`\[(.*)\]`)

// matchByline reads a byline: a mailbox, as in alice@example.com or
// "Alice" <alice@example.com>, then a timestamp, in brackets. The author is
// kept as it's written, so the byline is written back unchanged.
func matchByline(n ast.Node, b []byte) (*Byline, error) {
	errStr := "Expected a line in the form [<author> <timestamp>]"

	if _, ok := n.(*ast.Paragraph); !ok {
		return nil, errors.New(errStr)
	}

	matches := bylinePattern.FindStringSubmatch(string(n.Text(b)))
	if matches == nil {
		return nil, errors.New(errStr)
	}

	_, author, date, err := models.SplitMailbox(matches[1])
	if err != nil {
		return nil, err
	}

	if date == "" {
		return nil, errors.New(errStr)
	}

	return &Byline{&author, &date}, nil
}

func matchMetadata(n ast.Node, b []byte) ([]models.Tag, error) {
	errStr := "Expected a list of tags in the form <key>: <value>[, <value>]"
	first := regexp.MustCompile(`([^ ]+): ([^,]+)`)
	rest := regexp.MustCompile(`, ([^,]+)`)

	switch n.(type) {
	case *ast.List:
		result := make([]models.Tag, 0)

		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			// Values continued on the next line are split as though they
			// were on one.
			str := singleLine(string(item.Text(b)))
			key := first.FindStringSubmatch(str)

			if key == nil {
				// TODO: Handle the case where the values are in a nested list
				return nil, errors.New(errStr)
			}

			idx := first.FindStringIndex(str)[1]

			values := []string{key[2]}

			for _, v := range rest.FindAllStringSubmatch(str[idx:], -1) {
				values = append(values, v[1])
			}

			result = append(result, models.Tag{
				Key:    key[1],
				Values: models.Tokens(values...),
			})
		}

		return result, nil
	default:
		return nil, errors.New(errStr)
	}
}

func matchBreak(n ast.Node, b []byte) error {
	switch n.(type) {
	case *ast.ThematicBreak:
		return nil
	default:
		return errors.New("Expected a horizontal rule")
	}
}
