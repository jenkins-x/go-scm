package github

import (
	"context"
	"fmt"
	"github.com/jenkins-x/go-scm/scm"
)

type commitService struct {
	client *wrapper
}

type commitCommentInput struct {
	Body string `json:"body"`
}

func (s *commitService) UpdateCommitStatus(ctx context.Context,
	repo string, sha string, options scm.CommitStatusUpdateOptions) (*scm.CommitStatus, *scm.Response, error) {
	return nil, nil, fmt.Errorf("not support yet")
}

func (s *commitService) CreateCommitComment(ctx context.Context,
	repo string, sha string, body string) (*scm.Comment, *scm.Response, error) {
	path := fmt.Sprintf("/repos/%s/commits/%s/comments", repo, sha)
	in := &commitCommentInput{
		Body: body,
	}
	out := new(issueComment)
	res, err := s.client.do(ctx, "POST", path, in, out)
	return convertCommitComment(out), res, err
}

func convertCommitComment(from *issueComment) *scm.Comment {
	return &scm.Comment{
		ID:   from.ID,
		Body: from.Body,
		Author: scm.User{
			Login:  from.User.Login,
			Avatar: from.User.AvatarURL,
		},
		Link:    from.HTMLURL,
		Created: from.CreatedAt,
		Updated: from.UpdatedAt,
	}
}
