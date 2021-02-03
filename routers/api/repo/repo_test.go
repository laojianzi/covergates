package repo

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"

	"github.com/covergates/covergates/core"
	"github.com/covergates/covergates/mock"
	"github.com/covergates/covergates/routers/api/request"
)

func testRequest(r http.Handler, req *http.Request, f func(*httptest.ResponseRecorder)) {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	f(w)
}

func TestCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := &core.Repo{
		URL:       "http://gitea/org/repo",
		NameSpace: "org",
		Name:      "repo",
		SCM:       core.Gitea,
		Branch:    "master",
	}
	user := &core.User{}
	store := mock.NewMockRepoStore(ctrl)
	store.EXPECT().Create(gomock.Eq(repo)).Return(nil)
	service := mock.NewMockSCMService(ctrl)

	data, err := json.Marshal(repo)
	if err != nil {
		t.Error(err)
		return
	}
	read := bytes.NewReader(data)
	req, _ := http.NewRequest("POST", "/repo", read)
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		request.WithUser(c, user)
	})
	r.POST("/repo", HandleCreate(store, service))
	testRequest(r, req, func(w *httptest.ResponseRecorder) {
		rst := w.Result()
		defer rst.Body.Close()
		if rst.StatusCode != 200 {
			t.Fail()
			return
		}
		data, err := ioutil.ReadAll(rst.Body)
		if err != nil {
			t.Error(err)
			return
		}
		rstRepo := &core.Repo{}
		_ = json.Unmarshal(data, rstRepo)
		if !reflect.DeepEqual(repo, rstRepo) {
			t.Fail()
		}
	})
}

func TestListSCM(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := &core.User{}
	scmRepos := []*core.Repo{
		{
			Name: "repo1",
			URL:  "url1",
		},
		{
			Name: "repo2",
			URL:  "url2",
		},
	}
	storeRepos := []*core.Repo{
		{
			URL:      "url2",
			ReportID: "report_id",
		},
	}
	urls := make([]string, len(scmRepos))
	for i, repo := range scmRepos {
		urls[i] = repo.URL
	}

	mockService := mock.NewMockSCMService(ctrl)
	mockClient := mock.NewMockClient(ctrl)
	mockRepoService := mock.NewMockGitRepoService(ctrl)
	mockStore := mock.NewMockRepoStore(ctrl)

	mockService.EXPECT().Client(gomock.Eq(core.Github)).Return(mockClient, nil)
	mockClient.EXPECT().Repositories().Return(mockRepoService)
	mockRepoService.EXPECT().List(gomock.Any(), gomock.Eq(user)).Return(scmRepos, nil)
	mockStore.EXPECT().Finds(gomock.Eq(urls)).Return(storeRepos, nil)

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		request.WithUser(c, user)
	})
	r.GET("/repos/:scm", HandleListSCM(mockService, mockStore))

	req, _ := http.NewRequest("GET", "/repos/github", nil)
	testRequest(r, req, func(w *httptest.ResponseRecorder) {
		rst := w.Result()
		defer rst.Body.Close()
		if rst.StatusCode != 200 {
			t.Fail()
			return
		}
		data, _ := ioutil.ReadAll(rst.Body)
		var repos []*core.Repo
		_ = json.Unmarshal(data, &repos)
		if len(repos) < 2 {
			t.Fail()
			return
		}
		if repos[0].ReportID != "" {
			t.Fail()
		}
		if repos[1].ReportID != "report_id" {
			t.Fail()
		}
	})
}

func TestReportIDRenew(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// data
	user := &core.User{}

	repo := &core.Repo{
		Name:      "repo",
		NameSpace: "github",
		SCM:       core.Github,
	}

	// mock
	mockStore := mock.NewMockRepoStore(ctrl)
	mockStore.EXPECT().Find(gomock.Eq(repo)).Return(repo, nil)
	mockStore.EXPECT().Update(gomock.Eq(&core.Repo{
		Name:      repo.Name,
		NameSpace: repo.NameSpace,
		SCM:       repo.SCM,
		ReportID:  "123",
	})).Return(nil)
	mockStore.EXPECT().UpdateCreator(gomock.Any(), gomock.Eq(user)).Return(nil)
	mockService := mock.NewMockSCMService(ctrl)
	mockClient := mock.NewMockClient(ctrl)
	mockRepositories := mock.NewMockGitRepoService(ctrl)
	mockService.EXPECT().Client(gomock.Eq(core.Github)).Return(mockClient, nil)
	mockClient.EXPECT().Repositories().Return(mockRepositories)
	mockRepositories.EXPECT().NewReportID(gomock.Eq(repo)).Return("123")

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		request.WithUser(c, user)
	})
	r.PATCH("/repos/:scm/:namespace/:name/report", HandleReportIDRenew(mockStore, mockService))

	req, _ := http.NewRequest("PATCH", "/repos/github/github/repo/report", nil)
	testRequest(r, req, func(h *httptest.ResponseRecorder) {
		result := h.Result()
		defer result.Body.Close()
		if result.StatusCode != 200 {
			t.Fatal()
		}
	})
}
