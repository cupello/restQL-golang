package web

import (
	"net/http"

	"github.com/b2wdigital/restQL-golang/v4/internal/eval"
	"github.com/b2wdigital/restQL-golang/v4/internal/parser"
	"github.com/b2wdigital/restQL-golang/v4/internal/platform/cache"
	"github.com/b2wdigital/restQL-golang/v4/internal/platform/conf"
	"github.com/b2wdigital/restQL-golang/v4/internal/platform/httpclient"
	"github.com/b2wdigital/restQL-golang/v4/internal/platform/logger"
	"github.com/b2wdigital/restQL-golang/v4/internal/platform/persistence"
	"github.com/b2wdigital/restQL-golang/v4/internal/platform/plugins"
	"github.com/b2wdigital/restQL-golang/v4/internal/runner"
	"github.com/valyala/fasthttp"
)

func API(log *logger.Logger, cfg *conf.Config) (fasthttp.RequestHandler, error) {
	log.Debug("starting api")
	defaultParser, err := parser.New()
	if err != nil {
		log.Error("failed to compile parser", err)
		return nil, err
	}
	parserCacheLoader := cache.New(log, cfg.Cache.Parser.MaxSize, cache.ParserCacheLoader(defaultParser))
	parserCache := cache.NewParserCache(log, parserCacheLoader)

	db, err := persistence.NewDatabase(log)
	if err != nil {
		log.Error("failed to establish connection to database", err)
		return nil, err
	}

	lifecycle, err := plugins.NewLifecycle(log)
	if err != nil {
		log.Error("failed to initialize plugins", err)
	}

	app := NewApp(log, cfg, lifecycle)
	client := httpclient.New(log, lifecycle, cfg)
	executor := runner.NewExecutor(log, client, cfg.Http.QueryResourceTimeout, cfg.Http.ForwardPrefix)
	r := runner.NewRunner(log, executor, cfg.Http.GlobalQueryTimeout)

	mr := persistence.NewMappingReader(log, cfg.Env, cfg.Mappings, db)
	tenantCache := cache.New(log, cfg.Cache.Mappings.MaxSize,
		cache.TenantCacheLoader(mr),
		cache.WithExpiration(cfg.Cache.Mappings.Expiration),
		cache.WithRefreshInterval(cfg.Cache.Mappings.RefreshInterval),
		cache.WithRefreshQueueLength(cfg.Cache.Mappings.RefreshQueueLength),
	)
	cacheMr := cache.NewMappingsReaderCache(log, tenantCache)

	qr := persistence.NewQueryReader(log, cfg.Queries, db)
	queryCache := cache.New(log, cfg.Cache.Query.MaxSize, cache.QueryCacheLoader(qr))
	cacheQr := cache.NewQueryReaderCache(log, queryCache)

	e := eval.NewEvaluator(log, cacheMr, cacheQr, r, parserCache, lifecycle)

	restQl := NewRestQl(log, cfg, e, defaultParser)

	app.Handle(http.MethodPost, "/validate-query", restQl.ValidateQuery)
	app.Handle(http.MethodPost, "/run-query", restQl.RunAdHocQuery)
	app.Handle(http.MethodGet, "/run-query/:namespace/:queryId/:revision", restQl.RunSavedQuery)
	app.Handle(http.MethodPost, "/run-query/:namespace/:queryId/:revision", restQl.RunSavedQuery)

	return app.RequestHandler(), nil
}

func Health(log *logger.Logger, cfg *conf.Config) fasthttp.RequestHandler {
	app := NewApp(log, cfg, plugins.NoOpLifecycle)
	check := NewCheck(cfg.Build)

	app.Handle(http.MethodGet, "/health", check.Health)
	app.Handle(http.MethodGet, "/resource-status", check.ResourceStatus)

	return app.RequestHandlerWithoutMiddlewares()
}

func Debug(log *logger.Logger, cfg *conf.Config) fasthttp.RequestHandler {
	app := NewApp(log, cfg, plugins.NoOpLifecycle)
	pprof := NewPprof()

	app.Handle(http.MethodGet, "/debug/pprof/goroutine", pprof.Index)
	app.Handle(http.MethodGet, "/debug/pprof/heap", pprof.Index)
	app.Handle(http.MethodGet, "/debug/pprof/threadcreate", pprof.Index)
	app.Handle(http.MethodGet, "/debug/pprof/block", pprof.Index)
	app.Handle(http.MethodGet, "/debug/pprof/mutex", pprof.Index)

	app.Handle(http.MethodGet, "/debug/pprof/profile", pprof.Profile)

	return app.RequestHandlerWithoutMiddlewares()
}
