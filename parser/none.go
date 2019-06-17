package parser

import (
	"context"
	"strings"
	"time"

	"github.com/k1LoW/harvest/client"
	"github.com/k1LoW/harvest/config"
)

// NoneParser ...
type NoneParser struct {
	t *config.Target
}

// NewNoneParser ...
func NewNoneParser(t *config.Target) (Parser, error) {
	return &NoneParser{
		t: t,
	}, nil
}

// Parse ...
func (p *NoneParser) Parse(ctx context.Context, cancel context.CancelFunc, lineChan <-chan client.Line, tz string, st *time.Time, et *time.Time) <-chan Log {
	if p.t.MultiLine {
		return p.parseMultipleLine(ctx, cancel, lineChan, tz)
	}
	return p.parseSingleLine(ctx, cancel, lineChan, tz)
}

func (p *NoneParser) parseSingleLine(ctx context.Context, cancel context.CancelFunc, lineChan <-chan client.Line, tz string) <-chan Log {
	logChan := make(chan Log)

	go func() {
		defer close(logChan)
	L:
		for line := range lineChan {
			logChan <- Log{
				Host:           line.Host,
				Path:           line.Path,
				Timestamp:      0,
				FilledByPrevTs: false,
				Content:        line.Content,
				Target:         p.t,
			}

			select {
			case <-ctx.Done():
				break L
			default:
			}
		}
	}()

	return logChan
}

func (p *NoneParser) parseMultipleLine(ctx context.Context, cancel context.CancelFunc, lineChan <-chan client.Line, tz string) <-chan Log {
	logChan := make(chan Log)
	contentStash := []string{}

	var (
		hostStash string
		pathStash string
	)

	go func() {
		defer func() {
			logChan <- Log{
				Host:           hostStash,
				Path:           pathStash,
				Timestamp:      0,
				FilledByPrevTs: false,
				Content:        strings.Join(contentStash, "\n"),
				Target:         p.t,
			}
			close(logChan)
		}()
	L:
		for line := range lineChan {
			hostStash = line.Host
			pathStash = line.Path

			if strings.HasPrefix(line.Content, " ") || strings.HasPrefix(line.Content, "\t") {
				contentStash = append(contentStash, line.Content)
				if len(contentStash) > maxContentStash {
					logChan <- Log{
						Host:           line.Host,
						Path:           line.Path,
						Timestamp:      0,
						FilledByPrevTs: false,
						Content:        strings.Join(contentStash, "\n"),
						Target:         p.t,
					}
					logChan <- Log{
						Host:           line.Host,
						Path:           line.Path,
						Timestamp:      0,
						FilledByPrevTs: false,
						Content:        "Harvest parse error: too many rows",
						Target:         p.t,
					}
					contentStash = nil
				}
				continue
			}

			if len(contentStash) > 0 {
				logChan <- Log{
					Host:           line.Host,
					Path:           line.Path,
					Timestamp:      0,
					FilledByPrevTs: false,
					Content:        strings.Join(contentStash, "\n"),
					Target:         p.t,
				}
			}

			contentStash = nil
			contentStash = append(contentStash, line.Content)

			select {
			case <-ctx.Done():
				break L
			default:
			}
		}
	}()

	return logChan
}
