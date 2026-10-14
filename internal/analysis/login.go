package analysis

import (
	"strings"

	"github.com/RidmaTP/web-analyzer/internal/models"
	"golang.org/x/net/html"
)

func (a *BodyAnalyzer) FindIfLogin(tokenType html.TokenType, token html.Token, loginFlags *models.LoginFlags) error {
	a.muOutput.Lock()
	defer a.muOutput.Unlock()
	if a.Output.IsLogin {
		return nil
	}
	if loginFlags.IsLoginButton && loginFlags.IsPasswordField && loginFlags.IsTextField && loginFlags.IsForm {
		a.Output.IsLogin = true
		return a.emitLocked()
	}

	if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
		switch token.Data {
		case "form":
			loginFlags.IsForm = true
			loginFlags.InForm = true
		case "input":
			for _, attr := range token.Attr {
				if attr.Key != "type" {
					continue
				}
				switch attr.Val {
				case "password":
					loginFlags.IsPasswordField = true
				case "email", "text":
					loginFlags.IsTextField = true
				case "submit":
					loginFlags.IsLoginButton = true
				}
			}
		case "button":
			for _, attr := range token.Attr {
				if attr.Key == "type" && attr.Val == "submit" {
					loginFlags.InButton = true
				}
			}
		}
	} else if tokenType == html.EndTagToken && token.Data == "button" {
		if loginFlags.InForm && loginFlags.InButton {
			loginFlags.InButton = false
		}
	} else if tokenType == html.TextToken && loginFlags.InButton && loginFlags.InForm {
		loginKeywords := []string{"login", "log in", "sign in", "signin", "submit", "access"}
		btnText := strings.ToLower(strings.ReplaceAll(token.Data, " ", ""))
		for _, keyword := range loginKeywords {
			if btnText == keyword {
				loginFlags.IsLoginButton = true
				break
			}
		}
	}
	return nil
}
