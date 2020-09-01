// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gitea

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"time"

	"github.com/jenkins-x/go-scm/scm"
)

type stateType string

const (
	// stateOpen pr/issue is opend
	stateOpen stateType = "open"
	// stateClosed pr/issue is closed
	stateClosed stateType = "closed"
	// stateAll is all
	stateAll stateType = "all"
)

type issueService struct {
	client *wrapper
}

func (s *issueService) Search(context.Context, scm.SearchOptions) ([]*scm.SearchIssue, *scm.Response, error) {
	// TODO implemment
	return nil, nil, scm.ErrNotSupported
}

func (s *issueService) AssignIssue(ctx context.Context, repo string, number int, logins []string) (*scm.Response, error) {
	issue, res, err := s.Find(ctx, repo, number)
	if err != nil {
		return res, errors.Wrapf(err, "couldn't lookup issue %d in repository %s", number, repo)
	}
	if issue == nil {
		return res, fmt.Errorf("couldn't find issue %d in repository %s", number, repo)
	}
	assignees := sets.NewString(logins...)
	for _, existingAssignee := range issue.Assignees {
		assignees.Insert(existingAssignee.Login)
	}

	path := fmt.Sprintf("api/v1/repos/%s/issues/%d", repo, number)
	in := &assignUnassignInput{
		Assignees: assignees.List(),
	}
	return s.client.do(ctx, "PATCH", path, in, nil)
}

func (s *issueService) UnassignIssue(ctx context.Context, repo string, number int, logins []string) (*scm.Response, error) {
	issue, res, err := s.Find(ctx, repo, number)
	if err != nil {
		return res, errors.Wrapf(err, "couldn't lookup issue %d in repository %s", number, repo)
	}
	if issue == nil {
		return res, fmt.Errorf("couldn't find issue %d in repository %s", number, repo)
	}
	assignees := sets.NewString()
	for _, existingAssignee := range issue.Assignees {
		assignees.Insert(existingAssignee.Login)
	}
	assignees.Delete(logins...)

	path := fmt.Sprintf("api/v1/repos/%s/issues/%d", repo, number)
	in := &assignUnassignInput{
		Assignees: assignees.List(),
	}
	return s.client.do(ctx, "PATCH", path, in, nil)
}

func (s *issueService) ListEvents(context.Context, string, int, scm.ListOptions) ([]*scm.ListedIssueEvent, *scm.Response, error) {
	return nil, nil, scm.ErrNotSupported
}

func (s *issueService) ListLabels(ctx context.Context, repo string, number int, _ scm.ListOptions) ([]*scm.Label, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues/%d/labels", repo, number)
	out := []*label{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertLabelObjects(out), res, err
}

func (s *issueService) lookupLabel(ctx context.Context, repo string, lbl string) (int64, *scm.Response, error) {
	var labelID int64
	labelID = -1
	repoLabels, res, err := s.client.Repositories.ListLabels(ctx, repo, scm.ListOptions{})
	if err != nil {
		return labelID, res, errors.Wrapf(err, "listing labels in repository %s", repo)
	}
	for _, l := range repoLabels {
		if l.Name == lbl {
			labelID = l.ID
			break
		}
	}
	return labelID, res, nil
}

func (s *issueService) AddLabel(ctx context.Context, repo string, number int, lbl string) (*scm.Response, error) {
	labelID, res, err := s.lookupLabel(ctx, repo, lbl)
	if err != nil {
		return res, err
	}
	if labelID == -1 {
		lblInput := &createLabelInput{
			Color:       "#00aabb",
			Description: "",
			Name:        lbl,
		}
		newLabel := new(label)
		lblPath := fmt.Sprintf("api/v1/repos/%s/labels", repo)
		res, err = s.client.do(ctx, "POST", lblPath, lblInput, newLabel)
		if err != nil {
			return res, errors.Wrapf(err, "failed to create label %s in repository %s", lbl, repo)
		}
		labelID = newLabel.ID
	}

	path := fmt.Sprintf("api/v1/repos/%s/issues/%d/labels", repo, number)
	in := &addLabelInput{Labels: []int64{labelID}}

	return s.client.do(ctx, "POST", path, in, nil)
}

func (s *issueService) DeleteLabel(ctx context.Context, repo string, number int, lbl string) (*scm.Response, error) {
	labelID, res, err := s.lookupLabel(ctx, repo, lbl)
	if err != nil {
		return res, err
	}
	if labelID == -1 {
		return nil, nil
	}

	path := fmt.Sprintf("api/v1/repos/%s/issues/%d/labels/%d", repo, number, labelID)
	return s.client.do(ctx, "DELETE", path, nil, nil)
}

func (s *issueService) Find(ctx context.Context, repo string, number int) (*scm.Issue, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues/%d", repo, number)
	out := new(issue)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertIssue(out), res, err
}

func (s *issueService) FindComment(ctx context.Context, repo string, index, id int) (*scm.Comment, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues/%d/comment/%d", repo, index, id)
	out := new(issueComment)
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertIssueComment(out), res, err
}

func (s *issueService) List(ctx context.Context, repo string, _ scm.IssueListOptions) ([]*scm.Issue, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues?type=issues", repo)
	out := []*issue{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertIssueList(out), res, err
}

func (s *issueService) ListComments(ctx context.Context, repo string, index int, _ scm.ListOptions) ([]*scm.Comment, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues/%d/comments", repo, index)
	out := []*issueComment{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertIssueCommentList(out), res, err
}

func (s *issueService) Create(ctx context.Context, repo string, input *scm.IssueInput) (*scm.Issue, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues", repo)
	in := &issueInput{
		Title: input.Title,
		Body:  input.Body,
	}
	out := new(issue)
	res, err := s.client.do(ctx, "POST", path, in, out)
	return convertIssue(out), res, err
}

func (s *issueService) CreateComment(ctx context.Context, repo string, index int, input *scm.CommentInput) (*scm.Comment, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues/%d/comments", repo, index)
	in := &issueCommentInput{
		Body: input.Body,
	}
	out := new(issueComment)
	res, err := s.client.do(ctx, "POST", path, in, out)
	return convertIssueComment(out), res, err
}

func (s *issueService) DeleteComment(ctx context.Context, repo string, index, id int) (*scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues/%d/comments/%d", repo, index, id)
	return s.client.do(ctx, "DELETE", path, nil, nil)
}

func (s *issueService) EditComment(ctx context.Context, repo string, number int, id int, input *scm.CommentInput) (*scm.Comment, *scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues/%d/comment/%d", repo, number, id)
	in := &issueCommentInput{
		Body: input.Body,
	}
	out := new(issueComment)
	res, err := s.client.do(ctx, "PATCH", path, in, out)
	return convertIssueComment(out), res, err
}

func (s *issueService) Close(ctx context.Context, repo string, number int) (*scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues/%d", repo, number)
	in := &closeReopenInput{
		State: stateClosed,
	}
	return s.client.do(ctx, "PATCH", path, in, nil)
}

func (s *issueService) Reopen(ctx context.Context, repo string, number int) (*scm.Response, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues/%d", repo, number)
	in := &closeReopenInput{
		State: stateOpen,
	}
	return s.client.do(ctx, "PATCH", path, in, nil)
}

func (s *issueService) Lock(ctx context.Context, repo string, number int) (*scm.Response, error) {
	return nil, scm.ErrNotSupported
}

func (s *issueService) Unlock(ctx context.Context, repo string, number int) (*scm.Response, error) {
	return nil, scm.ErrNotSupported
}

//
// native data structures
//

type createLabelInput struct {
	Color       string `json:"color"`
	Description string `json:"description"`
	Name        string `json:"name"`
}

type addLabelInput struct {
	Labels []int64 `json:"labels"`
}

type label struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// example: 00aabb
	Color       string `json:"color"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type assignUnassignInput struct {
	Assignees []string `json:"assignees"`
}

type closeReopenInput struct {
	State stateType `json:"state"`
}

type (
	// gitea issue response object.
	issue struct {
		ID          int       `json:"id"`
		Number      int       `json:"number"`
		User        user      `json:"user"`
		Title       string    `json:"title"`
		Body        string    `json:"body"`
		State       stateType `json:"state"`
		Labels      []label   `json:"labels"`
		Comments    int       `json:"comments"`
		Assignees   []user    `json:"assignees"`
		Created     time.Time `json:"created_at"`
		Updated     time.Time `json:"updated_at"`
		PullRequest *struct {
			Merged   bool        `json:"merged"`
			MergedAt interface{} `json:"merged_at"`
		} `json:"pull_request"`
	}

	// gitea issue request object.
	issueInput struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	// gitea issue comment response object.
	issueComment struct {
		ID        int       `json:"id"`
		HTMLURL   string    `json:"html_url"`
		User      user      `json:"user"`
		Body      string    `json:"body"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	// gitea issue comment request object.
	issueCommentInput struct {
		Body string `json:"body"`
	}
)

//
// native data structure conversion
//

func convertIssueList(from []*issue) []*scm.Issue {
	to := []*scm.Issue{}
	for _, v := range from {
		to = append(to, convertIssue(v))
	}
	return to
}

func convertIssue(from *issue) *scm.Issue {
	return &scm.Issue{
		Number:    from.Number,
		Title:     from.Title,
		Body:      from.Body,
		Link:      "", // TODO construct the link to the issue.
		Closed:    from.State == "closed",
		Labels:    convertLabels(from),
		Author:    *convertUser(&from.User),
		Assignees: convertUsers(from.Assignees),
		Created:   from.Created,
		Updated:   from.Updated,
	}
}

func convertIssueCommentList(from []*issueComment) []*scm.Comment {
	to := []*scm.Comment{}
	for _, v := range from {
		to = append(to, convertIssueComment(v))
	}
	return to
}

func convertIssueComment(from *issueComment) *scm.Comment {
	return &scm.Comment{
		ID:      from.ID,
		Body:    from.Body,
		Author:  *convertUser(&from.User),
		Created: from.CreatedAt,
		Updated: from.UpdatedAt,
	}
}

func convertLabels(from *issue) []string {
	var labels []string
	for _, label := range from.Labels {
		labels = append(labels, label.Name)
	}
	return labels
}

func convertLabelObjects(from []*label) []*scm.Label {
	var labels []*scm.Label
	for _, label := range from {
		labels = append(labels, &scm.Label{
			Name:        label.Name,
			Description: label.Description,
			URL:         label.URL,
			Color:       label.Color,
		})
	}
	return labels
}
