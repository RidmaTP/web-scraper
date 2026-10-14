package analysis

import (
	"context"

	"github.com/RidmaTP/web-analyzer/internal/utils"
)

func (a *BodyAnalyzer) emitLocked() error {
	if a.Stream == nil {
		return nil
	}
	jsonStr, err := utils.JsonToText(a.Output)
	if err != nil {
		return err
	}
	select {
	case a.Stream <- *jsonStr:
		return nil
	case <-a.context().Done():
		return a.context().Err()
	}
}

func (a *BodyAnalyzer) context() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}
