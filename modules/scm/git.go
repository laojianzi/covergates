package scm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/github"
	log "github.com/sirupsen/logrus"

	"github.com/covergates/covergates/core"
)

type gitService struct {
	git       core.Git
	scm       core.SCMProvider
	scmClient *scm.Client
}

type giteaCommit struct {
	Sha       string           `json:"sha"`
	Commit    *giteaRepoCommit `json:"commit"`
	Committer *giteaUser       `json:"committer"`
}

type githubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
		Message string `json:"message"`
	} `json:"commit"`
	Committer struct {
		AvatarURL string `json:"avatar_url"`
		Login     string `json:"login"`
	} `json:"committer"`
}

type giteaRepoCommit struct {
	Message   string           `json:"message"`
	Committer *giteaCommitUser `json:"committer"`
}

type giteaCommitUser struct {
	Name string `json:"name"`
}

type giteaUser struct {
	UserName string `json:"username"`
	Avatar   string `json:"avatar_url"`
}

func (service *gitService) FindCommit(ctx context.Context, user *core.User, repo *core.Repo) string {
	client := service.scmClient
	ctx = withUser(ctx, service.scm, user)
	ref, _, err := client.Git.FindBranch(
		ctx,
		fmt.Sprintf("%s/%s", repo.NameSpace, repo.Name),
		repo.Branch,
	)
	if err != nil {
		return ""
	}
	return ref.Sha
}

func (service *gitService) ListCommits(ctx context.Context, user *core.User, repo string) ([]*core.Commit, error) {
	ctx = withUser(ctx, service.scm, user)
	if service.scm == core.Gitea {
		return service.listGiteaCommits(ctx, repo, "")
	}
	return service.listCommits(ctx, repo, "")
}

func (service *gitService) ListCommitsByRef(ctx context.Context, user *core.User, repo, ref string) ([]*core.Commit, error) {
	ctx = withUser(ctx, service.scm, user)
	switch service.scm {
	case core.Gitea:
		return service.listGiteaCommits(ctx, repo, ref)
	case core.Github:
		return service.listGithubCommits(ctx, repo, ref)
	default:
		return service.listCommits(ctx, repo, ref)
	}
}

func (service *gitService) listCommits(ctx context.Context, repo, ref string) ([]*core.Commit, error) {
	client := service.scmClient
	options := scm.CommitListOptions{Size: 20}
	if ref != "" {
		options.Ref = ref
	}
	commits, _, err := client.Git.ListCommits(ctx, repo, options)
	if err != nil {
		return nil, err
	}
	results := make([]*core.Commit, len(commits))
	for i, commit := range commits {
		results[i] = &core.Commit{
			Sha:             commit.Sha,
			Message:         commit.Message,
			Committer:       commit.Committer.Name,
			CommitterAvater: commit.Committer.Avatar,
		}
	}
	return results, nil
}

func mustGetGiteaCommitsQuery(repo, ref string) string {
	u, err := url.Parse(fmt.Sprintf("api/v1/repos/%s/commits", repo))
	if err != nil {
		log.Fatal(err)
	}
	query := u.Query()
	if ref != "" {
		query.Set("sha", ref)
	}
	u.RawQuery = query.Encode()
	return u.String()
}

func (service *gitService) listGiteaCommits(ctx context.Context, repo, ref string) ([]*core.Commit, error) {
	client := service.scmClient
	res, err := client.Do(ctx, &scm.Request{
		Header: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Method: "GET",
		Path:   mustGetGiteaCommitsQuery(repo, ref),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.Status > 300 {
		return nil, errors.New(http.StatusText(res.Status))
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var commits []giteaCommit
	if err := json.Unmarshal(data, &commits); err != nil {
		return nil, err
	}
	results := make([]*core.Commit, len(commits))
	for i, commit := range commits {
		results[i] = &core.Commit{
			Sha:     commit.Sha,
			Message: commit.Commit.Message,
		}
		if commit.Committer != nil {
			results[i].Committer = commit.Committer.UserName
			results[i].CommitterAvater = commit.Committer.Avatar
		} else {
			results[i].Committer = commit.Commit.Committer.Name
		}
	}
	return results, nil
}

func (service *gitService) listGithubCommits(ctx context.Context, repo, ref string) ([]*core.Commit, error) {
	params := url.Values{}
	params.Add("sha", ref)
	params.Add("per_page", "25")

	res, err := service.scmClient.Do(ctx, &scm.Request{
		Method: "GET",
		Path:   fmt.Sprintf("repos/%s/commits?%s", repo, params.Encode()),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.Status > 300 {
		var err *github.Error
		_ = json.NewDecoder(res.Body).Decode(err)
		return nil, err
	}
	var commits []*githubCommit
	_ = json.NewDecoder(res.Body).Decode(&commits)
	result := make([]*core.Commit, len(commits))
	for i, commit := range commits {
		result[i] = &core.Commit{
			Committer:       commit.Commit.Committer.Name,
			CommitterAvater: commit.Committer.AvatarURL,
			Message:         commit.Commit.Message,
			Sha:             commit.SHA,
		}
	}
	return result, nil
}

func (service *gitService) ListBranches(ctx context.Context, user *core.User, repo string) ([]string, error) {
	client := service.scmClient
	ctx = withUser(ctx, service.scm, user)
	references, _, err := client.Git.ListBranches(ctx, repo, scm.ListOptions{})
	if err != nil {
		return []string{}, err
	}
	branches := make([]string, len(references))
	for i, ref := range references {
		branches[i] = ref.Name
	}
	return branches, nil
}

// GitRepository clone
func (service *gitService) GitRepository(ctx context.Context, user *core.User, repo string) (core.GitRepository, error) {
	client := service.scmClient
	rs := &repoService{scm: service.scm, client: client}
	token := userToken(service.scm, user)
	ctx = withUser(ctx, service.scm, user)
	cloneURL, err := rs.CloneURL(ctx, user, repo)
	if err != nil {
		return nil, err
	}
	return service.git.Clone(ctx, cloneURL, token.Token)
}
