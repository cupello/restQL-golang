package runner

import (
	"context"
	"github.com/b2wdigital/restQL-golang/internal/domain"
	"github.com/b2wdigital/restQL-golang/pkg/restql"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"time"
)

var errNoTimeoutProvided = errors.New("no timeout provided")

type Executor struct {
	client          domain.HttpClient
	log             restql.Logger
	resourceTimeout time.Duration
	forwardPrefix   string
}

func NewExecutor(log restql.Logger, client domain.HttpClient, resourceTimeout time.Duration, forwardPrefix string) Executor {
	return Executor{client: client, log: log, resourceTimeout: resourceTimeout, forwardPrefix: forwardPrefix}
}

func (e Executor) DoStatement(ctx context.Context, statement domain.Statement, queryCtx domain.QueryContext) (domain.DoneResource, error) {
	debug := IsDebugEnabled(queryCtx)
	drOptions := DoneResourceOptions{
		Debugging:    debug,
		IgnoreErrors: statement.IgnoreErrors,
		MaxAge:       statement.CacheControl.MaxAge,
		SMaxAge:      statement.CacheControl.SMaxAge,
	}

	emptyChainedParams := GetEmptyChainedParams(statement)
	if len(emptyChainedParams) > 0 {
		emptyChainedResponse := NewEmptyChainedResponse(emptyChainedParams, drOptions)
		e.log.Debug("request execution skipped due to empty chained parameters", "resource", statement.Resource, "method", statement.Method)
		return emptyChainedResponse, nil
	}

	request := MakeRequest(e.resourceTimeout, e.forwardPrefix, statement, queryCtx)

	e.log.Debug("executing request for statement", "resource", statement.Resource, "method", statement.Method, "request", request)

	response, err := e.client.Do(ctx, request)

	switch {
	case err == domain.ErrRequestTimeout:
		return NewErrorResponse(err, request, response, drOptions), nil
	case err != nil:
		e.log.Debug("request execution failed", "error", err)
		return NewErrorResponse(err, request, response, drOptions), err
	}

	dr := NewDoneResource(request, response, drOptions)

	e.log.Debug("request execution done", "resource", statement.Resource, "method", statement.Method, "response", dr)

	return dr, nil
}

func (e Executor) DoMultiplexedStatement(ctx context.Context, statements []interface{}, queryCtx domain.QueryContext) (domain.DoneResources, error) {
	responseChans := make([]chan interface{}, len(statements))
	for i := range responseChans {
		responseChans[i] = make(chan interface{}, 1)
	}
	defer func() {
		for _, ch := range responseChans {
			close(ch)
		}
	}()

	sem := make(chan struct{}, 10)

	var g errgroup.Group

	for i, stmt := range statements {
		i, stmt := i, stmt
		ch := responseChans[i]
		sem <- struct{}{}

		g.Go(func() error {
			response, err := e.doCurrentStatement(stmt, ctx, queryCtx)
			if err != nil {
				return err
			}
			ch <- response
			<-sem
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	responses := make(domain.DoneResources, len(statements))
	for i, ch := range responseChans {
		responses[i] = <-ch
	}

	return responses, nil
}

func (e Executor) doCurrentStatement(stmt interface{}, ctx context.Context, queryCtx domain.QueryContext) (interface{}, error) {
	switch stmt := stmt.(type) {
	case domain.Statement:
		r, err := e.DoStatement(ctx, stmt, queryCtx)
		if err != nil {
			return nil, err
		}
		return r, nil
	case []interface{}:
		r, err := e.DoMultiplexedStatement(ctx, stmt, queryCtx)
		if err != nil {
			return nil, err
		}
		return r, nil
	default:
		return nil, errors.Errorf("unknown statement type: %T", stmt)
	}
}
