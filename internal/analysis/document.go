package analysis

import (
	"strings"

	"github.com/RidmaTP/web-analyzer/internal/utils"
	"golang.org/x/net/html"
)

func (a *BodyAnalyzer) FindTitle(tokenType html.TokenType, token html.Token, inTitle bool) (bool, error) {
	a.muOutput.Lock()
	defer a.muOutput.Unlock()
	if a.Output.Title != "" {
		return false, nil
	}
	if token.Data == "title" {
		if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
			return true, nil
		} else if tokenType == html.EndTagToken {
			return false, nil
		}
	}
	if tokenType == html.TextToken && inTitle {
		trimmed := strings.TrimSpace(string(token.Data))
		if trimmed != "" {
			a.Output.Title = trimmed
			if err := a.emitLocked(); err != nil {
				return inTitle, err
			}
		}
	}
	return inTitle, nil
}

func (a *BodyAnalyzer) FindHTMLVersion(tokenType html.TokenType, token html.Token) error {
	a.muOutput.Lock()
	defer a.muOutput.Unlock()
	if a.Output.Version != "" {
		return nil
	}

	version := ""
	if tokenType == html.DoctypeToken {
		doctype := strings.TrimSpace(token.Data)
		fields := strings.Fields(doctype)
		if len(fields) == 0 {
			return nil
		}
		doctypeName := strings.ToLower(fields[0])
		switch {
		case utils.ContainsIgnoreCase(doctype, "HTML 4.01"):
			version = "HTML 4.01"
		case utils.ContainsIgnoreCase(doctype, "XHTML"):
			version = "XHTML"
		case doctypeName == "html":
			version = "HTML5"
		default:
			version = doctype
		}
	}
	if version != "" {
		a.Output.Version = version
		return a.emitLocked()
	}
	return nil
}

func (a *BodyAnalyzer) FindHeaderCount(tokenType html.TokenType, token html.Token) error {
	a.muOutput.Lock()
	defer a.muOutput.Unlock()
	if a.Output.Headers == nil {
		a.Output.Headers = make(map[string]int)
	}
	if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
		switch token.Data {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			a.Output.Headers[token.Data]++
			return a.emitLocked()
		}
	}
	return nil
}
