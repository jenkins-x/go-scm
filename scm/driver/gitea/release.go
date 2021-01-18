package gitea

import (
	"context"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/jenkins-x/go-scm/scm"
)

type releaseService struct {
	client *wrapper
}

func (s *releaseService) Find(ctx context.Context, repo string, id int) (*scm.Release, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	out, resp, err := s.client.GiteaClient.GetRelease(namespace, name, int64(id))
	return convertRelease(out), toSCMResponse(resp), err
}

func (s *releaseService) FindByTag(ctx context.Context, repo string, tag string) (*scm.Release, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	out, resp, err := s.client.GiteaClient.GetReleaseByTag(namespace, name, tag)
	if err != nil {
		// when a tag exists but has no release published, gitea returns a http 500 error, so normalise this to 404 not found
		// workaround until gitea releases 1.13.2 https://github.com/go-gitea/gitea/issues/14365
		if resp.StatusCode == 500 && (err.Error() == "" || strings.HasPrefix(err.Error(), "user does not exist")) {
			return nil, &scm.Response{
				Status: 404,
				Header: resp.Header,
				Body:   resp.Body,
			}, scm.ErrNotFound
		}
	}
	return convertRelease(out), toSCMResponse(resp), err
}

func (s *releaseService) List(ctx context.Context, repo string, opts scm.ReleaseListOptions) ([]*scm.Release, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	out, resp, err := s.client.GiteaClient.ListReleases(namespace, name, gitea.ListReleasesOptions{ListOptions: releaseListOptionsToGiteaListOptions(opts)})
	return convertReleaseList(out), toSCMResponse(resp), err
}

func (s *releaseService) Create(ctx context.Context, repo string, input *scm.ReleaseInput) (*scm.Release, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	out, resp, err := s.client.GiteaClient.CreateRelease(namespace, name, gitea.CreateReleaseOption{
		TagName:      input.Tag,
		Target:       input.Commitish,
		Title:        input.Title,
		Note:         input.Description,
		IsDraft:      input.Draft,
		IsPrerelease: input.Prerelease,
	})
	return convertRelease(out), toSCMResponse(resp), err
}

func (s *releaseService) Delete(ctx context.Context, repo string, id int) (*scm.Response, error) {
	namespace, name := scm.Split(repo)
	resp, err := s.client.GiteaClient.DeleteRelease(namespace, name, int64(id))
	return toSCMResponse(resp), err
}

func (s *releaseService) DeleteByTag(ctx context.Context, repo string, tag string) (*scm.Response, error) {
	rel, _, err := s.FindByTag(ctx, repo, tag)
	if err != nil {
		return nil, err
	}
	return s.Delete(ctx, repo, rel.ID)
}

func (s *releaseService) Update(ctx context.Context, repo string, id int, input *scm.ReleaseInput) (*scm.Release, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	out, resp, err := s.client.GiteaClient.EditRelease(namespace, name, int64(id), gitea.EditReleaseOption{
		TagName:      input.Tag,
		Target:       input.Commitish,
		Title:        input.Title,
		Note:         input.Description,
		IsDraft:      &input.Draft,
		IsPrerelease: &input.Prerelease,
	})
	return convertRelease(out), toSCMResponse(resp), err
}

func (s *releaseService) UpdateByTag(ctx context.Context, repo string, tag string, input *scm.ReleaseInput) (*scm.Release, *scm.Response, error) {
	rel, _, err := s.FindByTag(ctx, repo, tag)
	if err != nil {
		return nil, nil, err
	}
	return s.Update(ctx, repo, rel.ID, input)
}

func convertReleaseList(from []*gitea.Release) []*scm.Release {
	var to []*scm.Release
	for _, m := range from {
		to = append(to, convertRelease(m))
	}
	return to
}

func convertRelease(from *gitea.Release) *scm.Release {
	return &scm.Release{
		ID:          int(from.ID),
		Title:       from.Title,
		Description: from.Note,
		Link:        from.URL,
		Tag:         from.TagName,
		Commitish:   from.Target,
		Draft:       from.IsDraft,
		Prerelease:  from.IsPrerelease,
	}
}

func releaseListOptionsToGiteaListOptions(in scm.ReleaseListOptions) gitea.ListOptions {
	return gitea.ListOptions{
		Page:     in.Page,
		PageSize: in.Size,
	}
}