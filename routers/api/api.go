package api

import (
	"net/url"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/covergates/covergates/config"
	"github.com/covergates/covergates/core"
	"github.com/covergates/covergates/routers/api/repo"
	"github.com/covergates/covergates/routers/api/report"
	"github.com/covergates/covergates/routers/api/request"
	"github.com/covergates/covergates/routers/api/user"
	"github.com/covergates/covergates/routers/docs"
)

//go:generate swag init -g ./api.go -d ./ -o ../docs

// @title CodeCover API
// @version 1.0
// @description REST API for CodeCover
// @termsOfService http://swagger.io/terms/

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// Router for API
type Router struct {
	Config  *config.Config
	Session core.Session
	// service
	CoverageService core.CoverageService
	ChartService    core.ChartService
	SCMService      core.SCMService
	RepoService     core.RepoService
	ReportService   core.ReportService
	HookService     core.HookService
	OAuthService    core.OAuthService
	// store
	UserStore   core.UserStore
	ReportStore core.ReportStore
	RepoStore   core.RepoStore
	OAuthStore  core.OAuthStore
}

func host(addr string) string {
	u, err := url.Parse(addr)
	if err != nil {
		return addr
	}
	return u.Host
}

// RegisterRoutes for API
func (r *Router) RegisterRoutes(e *gin.Engine) {
	docs.SwaggerInfo.Host = host(r.Config.Server.Addr)
	checkLogin := request.CheckLogin(r.Session, r.OAuthService)
	e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	g := e.Group("/api/v1")
	{
		// nolint:govet
		g := g.Group("/user")
		g.GET("", checkLogin, user.HandleGet())
		g.POST("", user.HandleCreate())
		g.GET("/scm", checkLogin, user.HandleGetSCM(r.Config))
		g.GET("/owner/:scm/:namespace/:name", checkLogin, user.HandleGetOwner(r.RepoStore, r.SCMService))
		// tokens
		g.POST("/tokens", checkLogin, user.HandleCreateToken(r.OAuthService))
		g.GET("/tokens", checkLogin, user.HandleListTokens(r.OAuthService))
		g.DELETE("tokens/:id", checkLogin, user.HandleDeleteToken(r.OAuthService, r.OAuthStore))
		// repo
		g.PATCH("/repos", checkLogin, user.HandleSynchronizeRepo(r.RepoService))
		g.GET("/repos", checkLogin, user.HandleListRepo(r.UserStore))
	}
	{
		// nolint:govet
		g := g.Group("/reports")
		g.POST("/:id",
			report.InjectReportContext(r.RepoStore),
			report.ProtectReport(
				checkLogin,
				r.RepoStore,
				r.SCMService,
			),
			report.HandleUpload(
				r.CoverageService,
				r.ReportStore,
			))
		g.POST("/:id/comment/:number", report.HandleComment(
			r.Config,
			r.SCMService,
			r.RepoStore,
			r.ReportStore,
			r.ReportService,
		))
		g.GET("/:id", report.HandleGet(r.ReportStore, r.RepoStore, r.SCMService))
		g.GET("/:id/treemap/*ref", report.HandleGetTreeMap(
			r.ReportStore,
			r.RepoStore,
			r.ChartService,
		))
		g.GET("/:id/card", report.HandleGetCard(r.RepoStore, r.ReportStore, r.ChartService))
		g.GET("/:id/badge", report.HandleGetBadge(r.ReportStore, r.RepoStore))
	}
	{
		// nolint:govet
		g := g.Group("/repos")
		g.Use(checkLogin)
		g.GET("", repo.HandleListAll(r.Config, r.SCMService, r.RepoStore))
		g.POST("", repo.HandleCreate(r.RepoStore, r.SCMService))
		g.GET("/:scm", repo.HandleListSCM(r.SCMService, r.RepoStore))
		{
			// nolint:govet
			g := g.Group("/:scm/:namespace/:name")
			g.PATCH("", repo.HandleSync(r.SCMService, r.RepoStore))
			g.GET("/setting", repo.HandleGetSetting(r.RepoStore))
			g.POST("/setting", repo.WithRepo(r.RepoStore), repo.HandleUpdateSetting(r.RepoStore, r.SCMService))
			g.PATCH("/report", repo.HandleReportIDRenew(r.RepoStore, r.SCMService))
			g.GET("/files", repo.HandleGetFiles(r.SCMService))
			g.GET("/content/*path", repo.HandleGetFileContent(r.SCMService))
			g.POST("/hook/create", repo.WithRepo(r.RepoStore), repo.HandleHookCreate(r.HookService))
			g.GET("/commits", repo.WithRepo(r.RepoStore), repo.HandleListCommits(r.SCMService))
			g.GET("/branches", repo.WithRepo(r.RepoStore), repo.HandleListBranches(r.SCMService))
		}
	}
	{
		// repo api without authorization required
		// nolint:govet
		g := g.Group("/repos/:scm/:namespace/:name")
		g.GET("", repo.HandleGet(r.RepoStore))
		g.POST("/hook", repo.WithRepo(r.RepoStore), repo.HandleHook(r.SCMService, r.HookService))
	}
}
