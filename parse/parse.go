package parse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"

	"github.com/factory-factory/ish/glam"
	"github.com/factory-factory/ish/models"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
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

func ParseIssue(r io.Reader) (*models.Issue, error) {
	bytes, err := io.ReadAll(r)

	if err != nil {
		return nil, err
	}

	reader := text.NewReader(bytes)
	re, err := glam.NewTermRenderer(glam.WithStandardStyle("pink"))
	//glamour.WithStylePath("auto"))

	if err != nil {
		return nil, err
	}

	document := re.Parser().Parse(reader)
	issue, err := parseIssue(document, bytes, re.Renderer())

	if err != nil {
		return nil, err
	}

	return issue, nil
}

func parseIssue(document ast.Node, bytes []byte, r renderer.Renderer) (*models.Issue, error) {
	node := document.FirstChild()

	title, err := matchTitle(node, bytes)

	if err != nil {
		return nil, err
	}

	node = node.NextSibling()
	byline, err := matchByline(node, bytes)

	if err != nil {
		return nil, err
	}

	node = node.NextSibling()
	metadata, err := matchMetadata(node, bytes)

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

	for {
		card, next, err := parseCard(node, bytes, r)

		if err != nil {
			return nil, err
		}

		if card == nil {
			break
		} else {
			issue.Conversation = append(issue.Conversation, *card)
		}

		if next == nil {
			break
		}
		node = next
	}

	return issue, nil
}

func parseCard(node ast.Node, b []byte, r renderer.Renderer) (*models.Card, ast.Node, error) {
	node = node.NextSibling()

	if node == nil {
		return nil, node, nil
	}

	err := matchBreak(node, b)

	if err != nil {
		return nil, nil, err
	}

	node = node.NextSibling()
	byline, err := matchByline(node, b)

	if err != nil {
		return nil, nil, err
	}

	doc := ast.NewDocument()

	for {
		node = node.NextSibling()

		if node == nil {
			break
		} else if node.Kind().String() == "ThematicBreak" {
			node = node.PreviousSibling()
			break
		} else {
			next := node.NextSibling()
			prev := node.PreviousSibling()

			prev.SetNextSibling(next)

			if next != nil {
				next.SetPreviousSibling(prev)
			}

			parent := node.Parent()
			parent.RemoveChild(parent, node)

			doc.AppendChild(doc, node)

			node = prev
		}
	}

	//doc.Dump(b, 0)
	var buf bytes.Buffer

	err = r.Render(&buf, b, doc)
	content := string(buf.Bytes())

	fmt.Print(content)
	card := &models.Card{
		Author:      *byline.Author,
		DateCreated: *byline.DateCreated,
		Content:     &content,
	}

	return card, node, nil
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
