package github

import (
	"context"
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"gopkg.in/h2non/gock.v1"
	"io/ioutil"
	"testing"
)

func TestCreateCommitComment(t *testing.T) {
	defer gock.Off()

	gock.New("https://api.github.com").
		Post("/repos/octocat/hello-world/commits/6dcb09b5b57875f334f61aebed695e2e4193db5e/comments").
		BodyString(`{"body":"Great stuff"}`).
		Reply(200).
		File("testdata/commit_comment_response.json")

	client := NewDefault()
	got, _, err := client.Commits.CreateCommitComment(
		context.Background(),
		"octocat/hello-world",
		"6dcb09b5b57875f334f61aebed695e2e4193db5e",
		"Great stuff")
	if err != nil {
		t.Error(err)
		return
	}

	want := new(scm.Comment)
	raw, _ := ioutil.ReadFile("testdata/commit_comment_response.json.golden")
	json.Unmarshal(raw, want)

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Unexpected Results")
		t.Log(diff)
	}
}
