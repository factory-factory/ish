package parse

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
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
// 	issue := InitialModel()
// 	issue.Print()
// }

func InitialModel() models.Issue {
	issueId := "01AN4Z07BY" //"79KA1307SR"
	file, err := os.Open(fmt.Sprintf("./issues/%s.md", issueId))

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	issue, err := ParseIssue(file)

	if err != nil {
		log.Fatal(err)
	}

	issue.Id = issueId

	return *issue
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

	node = node.NextSibling()
	metadata, err := matchMetadata(node, source)

	if err != nil {
		return nil, err
	}

	issue := &models.Issue{
		Title:        *title,
		Author:       *byline.Author,
		DateCreated:  *byline.DateCreated,
		Metadata:     metadata,
		Conversation: []models.Card{},
	}

	for node = node.NextSibling(); node != nil; {
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

	// Trim blank lines around the content, but keep the indentation of its
	// first line, which may be significant.
	content := strings.TrimLeft(string(source[start:end]), "\r\n")
	content = strings.TrimRightFunc(content, unicode.IsSpace)

	card := &models.Card{
		Author:      *byline.Author,
		DateCreated: *byline.DateCreated,
		Content:     &content,
	}

	return card, next, nil
}

func matchTitle(n ast.Node, b []byte) (*string, error) {
	switch n.(type) {
	case *ast.Heading:
		str := string(n.Text(b))
		return &str, nil
	default:
		return nil, errors.New("Expected a heading")
	}
}

func matchByline(n ast.Node, b []byte) (*Byline, error) {
	errStr := "Expected a line in the form [<author> <timestamp>]"
	//elt := n.FirstChild().NextSibling()
	exp := regexp.MustCompile(`\[([^ ]*) ([^]]*)\]`)
	switch n.(type) {
	case *ast.Paragraph:
		matches := exp.FindStringSubmatch(string(n.Text(b)))

		if len(matches) != 3 {
			return nil, errors.New(errStr)
		}

		value := &Byline{&matches[1], &matches[2]}
		return value, nil
	default:
		return nil, errors.New(errStr)
	}

}

func matchMetadata(n ast.Node, b []byte) ([]models.Tag, error) {
	errStr := "Expected a list of tags in the form <key>: <value>[, <value>]"
	first := regexp.MustCompile(`([^ ]+): ([^,]+)`)
	rest := regexp.MustCompile(`, ([^,]+)`)

	switch n.(type) {
	case *ast.List:
		result := make([]models.Tag, 0)

		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			str := string(item.Text(b))
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
				Values: values,
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
