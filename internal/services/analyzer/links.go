package analyzer

import (
	"context"

	"github.com/RidmaTP/web-analyzer/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/net/html"
)

func (a *BodyAnalyzer) FindLinks(tokenType html.TokenType, token html.Token, baseURL string, linkJobQueue *chan string) error {
	if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
		return nil
	}
	if token.Data != "a" && token.Data != "link" {
		return nil
	}
	for _, attr := range token.Attr {
		if attr.Key != "href" {
			continue
		}
		if linkJobQueue != nil {
			select {
			case *linkJobQueue <- attr.Val:
			case <-a.context().Done():
				return a.context().Err()
			}
		}

		a.muOutput.Lock()
		if utils.IsExternalLink(attr.Val, baseURL) {
			a.Output.ExternalLinks.Count++
			a.Output.ExternalLinks.Links = append(a.Output.ExternalLinks.Links, attr.Val)
		} else {
			a.Output.InternalLinks.Count++
			a.Output.InternalLinks.Links = append(a.Output.InternalLinks.Links, attr.Val)
		}
		err := a.emitLocked()
		a.muOutput.Unlock()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *BodyAnalyzer) ActiveCheckWorker(baseURL string, linkJobQueue *chan string) {
	a.ActiveCheckWorkerContext(context.Background(), baseURL, *linkJobQueue)
}

func (a *BodyAnalyzer) ActiveCheckWorkerContext(ctx context.Context, baseURL string, linkJobQueue chan string) {
	for link := range linkJobQueue {
		link = utils.AddInternalHost(link, baseURL)
		linkCtx, span := a.getTracer().Start(ctx, "check link")
		span.SetAttributes(attribute.String("url.full", link))
		err := a.checkLink(linkCtx, link)
		if err != nil {
			span.RecordError(err)
		}
		span.End()

		a.muOutput.Lock()
		if err != nil {
			a.Output.InactiveLinks.Count++
			a.Output.InactiveLinks.Links = append(a.Output.InactiveLinks.Links, link)
		} else {
			a.Output.ActiveLinks.Count++
			a.Output.ActiveLinks.Links = append(a.Output.ActiveLinks.Links, link)
		}
		emitErr := a.emitLocked()
		a.muOutput.Unlock()
		if emitErr != nil {
			return
		}
	}
}
