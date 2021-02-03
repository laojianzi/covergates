package main

import (
	"github.com/google/wire"

	"github.com/covergates/covergates/config"
	"github.com/covergates/covergates/core"
	"github.com/covergates/covergates/modules/login"
	"github.com/covergates/covergates/routers"
)

// nolint:deadcode,varcheck,unused
var routerSet = wire.NewSet(
	provideLogin,
	provideRouter,
)

func provideLogin(config *config.Config) core.LoginMiddleware {
	return login.NewMiddleware(config)
}

func provideRouter(
	session core.Session,
	config *config.Config,
	login core.LoginMiddleware,
	// service
	scmService core.SCMService,
	coverageService core.CoverageService,
	chartService core.ChartService,
	reportService core.ReportService,
	repoService core.RepoService,
	hookService core.HookService,
	oauthSerice core.OAuthService,
	// store
	userStore core.UserStore,
	reportStore core.ReportStore,
	repoStore core.RepoStore,
	oauthStore core.OAuthStore,
) *routers.Routers {
	return &routers.Routers{
		Config:          config,
		Session:         session,
		LoginMiddleware: login,
		SCMService:      scmService,
		CoverageService: coverageService,
		ChartService:    chartService,
		RepoService:     repoService,
		ReportService:   reportService,
		HookService:     hookService,
		OAuthService:    oauthSerice,
		UserStore:       userStore,
		ReportStore:     reportStore,
		RepoStore:       repoStore,
		OAuthStore:      oauthStore,
	}
}
