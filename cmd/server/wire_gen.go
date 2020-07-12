// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package main

import (
	"github.com/code-devel-cover/CodeCover/config"
	"github.com/jinzhu/gorm"
)

import (
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// Injectors from wire.go:

func InitializeApplication(config2 *config.Config, db *gorm.DB) (application, error) {
	session := provideSession()
	loginMiddleware := provideLogin(config2)
	databaseService := provideDatabaseService(db)
	userStore := provideUserStore(databaseService)
	git := provideGit()
	scmService := provideSCMService(config2, userStore, git)
	coverageService := provideCoverageService()
	chartService := provideChartService()
	reportStore := provideReportStore(databaseService)
	repoStore := provideRepoStore(databaseService)
	routers := provideRouter(session, config2, loginMiddleware, scmService, coverageService, chartService, reportStore, repoStore)
	mainApplication := newApplication(routers, databaseService)
	return mainApplication, nil
}
