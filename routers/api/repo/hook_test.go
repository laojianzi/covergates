package repo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"

	"github.com/covergates/covergates/core"
	"github.com/covergates/covergates/mock"
)

var repo = &core.Repo{
	ID:        uint(1),
	ReportID:  "ABC",
	Name:      "name",
	NameSpace: "space",
	SCM:       core.Gitea,
}

func mockRepo(store *mock.MockRepoStore) *core.Repo {
	store.EXPECT().Find(gomock.Eq(
		&core.Repo{
			Name:      repo.Name,
			NameSpace: repo.NameSpace,
			SCM:       repo.SCM,
		})).AnyTimes().Return(repo, nil)
	return repo
}

func mockSCM(
	ctrl *gomock.Controller,
	scm *mock.MockSCMService,
) *mock.MockClient {
	client := mock.NewMockClient(ctrl)
	scm.EXPECT().Client(gomock.Eq(repo.SCM)).Return(client, nil)
	return client
}

func TestHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hook := struct{}{}

	store := mock.NewMockRepoStore(ctrl)
	scm := mock.NewMockSCMService(ctrl)
	service := mock.NewMockHookService(ctrl)
	webhook := mock.NewMockWebhookService(ctrl)
	repo := mockRepo(store)
	client := mockSCM(ctrl, scm)
	client.EXPECT().Webhooks().Return(webhook)
	webhook.EXPECT().Parse(gomock.Any()).Return(hook, nil)
	service.EXPECT().Resolve(gomock.Any(), gomock.Eq(repo), hook).Return(nil)

	r := gin.Default()
	r.POST("/repos/:scm/:namespace/:name/hook", WithRepo(store), HandleHook(scm, service))

	req, _ := http.NewRequest("POST", "/repos/gitea/space/name/hook", nil)
	testRequest(r, req, func(w *httptest.ResponseRecorder) {
		rst := w.Result()
		defer rst.Body.Close()
		if rst.StatusCode != 200 {
			t.Fatal("request fail")
		}
	})
}
