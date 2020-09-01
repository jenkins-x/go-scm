// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gitea

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/jenkins-x/go-scm/scm"
)

type pullService struct {
	*issueService
}

func (s *pullService) Find(ctx context.Context, repo string, index int) (*scm.PullRequest, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/pulls/%d", repo, index)
	out := new(pullRequest)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertPullRequest(out), res, err
}

func (s *pullService) List(ctx context.Context, repo string, opts scm.PullRequestListOptions) ([]*scm.PullRequest, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/pulls", repo)
	out := []*pullRequest{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertPullRequests(out), res, err
}

func (s *pullService) ListChanges(ctx context.Context, repo string, number int, _ scm.ListOptions) ([]*scm.Change, *scm.Response, error) {
	// Get the patch and then parse it.
	path := fmt.Sprintf("api/v1/repos/%s/pulls/%d.patch", repo, number)
	buf := new(bytes.Buffer)
	res, err := s.client.do(ctx, "GET", path, nil, buf)
	if err != nil {
		return nil, res, err
	}
	changedFiles, _, err := gitdiff.Parse(buf)
	if err != nil {
		return nil, res, err
	}
	var changes []*scm.Change
	for _, c := range changedFiles {
		var linesAdded int64
		var linesDeleted int64

		for _, tf := range c.TextFragments {
			linesAdded += tf.LinesAdded
			linesDeleted += tf.LinesDeleted
		}
		changes = append(changes, &scm.Change{
			Path:         c.NewName,
			PreviousPath: c.OldName,
			Added:        c.IsNew,
			Renamed:      c.IsRename,
			Deleted:      c.IsDelete,
			Additions:    int(linesAdded),
			Deletions:    int(linesDeleted),
		})
	}
	return changes, res, nil
}

func (s *pullService) Merge(ctx context.Context, repo string, index int, options *scm.PullRequestMergeOptions) (*scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/pulls/%d/merge", repo, index)
	res, err := s.client.do(ctx, "POST", path, nil, nil)
	return res, err
}

func (s *pullService) Update(ctx context.Context, repo string, number int, prInput *scm.PullRequestInput) (*scm.PullRequest, *scm.Response, error) {
	return nil, nil, scm.ErrNotSupported
}

func (s *pullService) Close(context.Context, string, int) (*scm.Response, error) {
	return nil, scm.ErrNotSupported
}

func (s *pullService) Reopen(context.Context, string, int) (*scm.Response, error) {
	return nil, scm.ErrNotSupported
}

func (s *pullService) Create(ctx context.Context, repo string, input *scm.PullRequestInput) (*scm.PullRequest, *scm.Response, error) {
	return nil, nil, scm.ErrNotSupported
}

func (s *pullService) RequestReview(ctx context.Context, repo string, number int, logins []string) (*scm.Response, error) {
	return nil, scm.ErrNotSupported
}

func (s *pullService) UnrequestReview(ctx context.Context, repo string, number int, logins []string) (*scm.Response, error) {
	return nil, scm.ErrNotSupported
}

//
// native data structures
//

type pullRequest struct {
	ID         int        `json:"id"`
	Number     int        `json:"number"`
	User       user       `json:"user"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	State      string     `json:"state"`
	HeadBranch string     `json:"head_branch"`
	HeadRepo   repository `json:"head_repo"`
	Head       reference  `json:"head"`
	BaseBranch string     `json:"base_branch"`
	BaseRepo   repository `json:"base_repo"`
	Base       reference  `json:"base"`
	HTMLURL    string     `json:"html_url"`
	Mergeable  bool       `json:"mergeable"`
	Merged     bool       `json:"merged"`
	Created    time.Time  `json:"created_at"`
	Updated    time.Time  `json:"updated_at"`
}

type reference struct {
	Repo repository `json:"repo"`
	Name string     `json:"ref"`
	Sha  string     `json:"sha"`
}

//
// native data structure conversion
//

func convertPullRequests(src []*pullRequest) []*scm.PullRequest {
	dst := []*scm.PullRequest{}
	for _, v := range src {
		dst = append(dst, convertPullRequest(v))
	}
	return dst
}

func convertPullRequest(src *pullRequest) *scm.PullRequest {
	return &scm.PullRequest{
		Number:  src.Number,
		Title:   src.Title,
		Body:    src.Body,
		Sha:     src.Head.Sha,
		Source:  src.Head.Name,
		Target:  src.Base.Name,
		Link:    src.HTMLURL,
		Fork:    src.Base.Repo.FullName,
		Ref:     fmt.Sprintf("refs/pull/%d/head", src.Number),
		Closed:  src.State == "closed",
		Author:  *convertUser(&src.User),
		Merged:  src.Merged,
		Created: src.Created,
		Updated: src.Updated,
	}
}

func convertPullRequestFromIssue(src *issue) *scm.PullRequest {
	return &scm.PullRequest{
		Number:  src.Number,
		Title:   src.Title,
		Body:    src.Body,
		Closed:  src.State == "closed",
		Author:  *convertUser(&src.User),
		Merged:  src.PullRequest.Merged,
		Created: src.Created,
		Updated: src.Updated,
	}
}
