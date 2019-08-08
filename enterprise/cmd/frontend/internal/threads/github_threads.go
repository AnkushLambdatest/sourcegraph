package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/comments"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/comments/commentobjectdb"
	"github.com/sourcegraph/sourcegraph/pkg/api"
	"github.com/sourcegraph/sourcegraph/pkg/extsvc/github"
)

func init() {
	graphqlbackend.ForceRefreshRepositoryThreads = ImportGitHubRepositoryThreads
}

func ImportGitHubRepositoryThreads(ctx context.Context, repoID api.RepoID, extRepo api.ExternalRepoSpec) error {
	client, externalServiceID, err := getClientForRepo(ctx, repoID)
	if err != nil {
		return err
	}

	results, err := listGitHubRepositoryIssuesAndPullRequests(ctx, client, graphql.ID(extRepo.ID))
	if err != nil {
		return err
	}

	toImport := make([]*externalThread, 0, len(results))
	for _, result := range results {
		// Skip cross-repository PRs because we don't handle those yet.
		if result.IsCrossRepository {
			continue
		}
		// HACK TODO!(sqs): omit renovate PRs
		if strings.HasPrefix(result.Title, "Update dependency ") {
			continue
		}

		thread, threadComment := githubIssueOrPullRequestToThread(result)
		thread.RepositoryID = repoID
		thread.ImportedFromExternalServiceID = externalServiceID

		replyComments := make([]*comments.ExternalComment, len(result.Comments.Nodes))
		for i, c := range result.Comments.Nodes {
			replyComments[i] = githubIssueCommentToExternalComment(c)
		}

		toImport = append(toImport, &externalThread{
			thread:        thread,
			threadComment: threadComment,
			comments:      replyComments,
		})
	}
	return ImportExternalThreads(ctx, repoID, externalServiceID, toImport)
}

func githubIssueOrPullRequestToThread(v *githubIssueOrPullRequest) (*dbThread, commentobjectdb.DBObjectCommentFields) {
	getRefName := func(ref *githubRef, oid string) string {
		// If a base/head is deleted, point to its OID directly.
		if ref == nil || v.State == "MERGED" {
			return oid
		}
		return ref.Prefix + ref.Name
	}

	thread := &dbThread{
		Title:      v.Title,
		State:      v.State,
		IsPreview:  false,
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
		BaseRef:    getRefName(v.BaseRef, v.BaseRefOid),
		HeadRef:    getRefName(v.HeadRef, v.HeadRefOid),
		ExternalID: string(v.ID),
	}
	var err error
	thread.ExternalMetadata, err = json.Marshal(v)
	if err != nil {
		panic(err)
	}

	comment := commentobjectdb.DBObjectCommentFields{
		Body:      v.Body,
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
	githubActorSetDBObjectCommentFields(v.Author, &comment)
	return thread, comment
}

func githubActorSetDBObjectCommentFields(actor *githubActor, f *commentobjectdb.DBObjectCommentFields) {
	// TODO!(sqs): map to sourcegraph user if possible
	f.AuthorExternalActorUsername = actor.Login
	f.AuthorExternalActorURL = actor.URL
}

func githubIssueCommentToExternalComment(v *githubIssueComment) *comments.ExternalComment {
	comment := comments.ExternalComment{}
	githubActorSetDBObjectCommentFields(v.Author, &comment.DBObjectCommentFields)
	comment.CreatedAt = v.CreatedAt
	comment.UpdatedAt = v.UpdatedAt
	comment.Body = v.Body
	return &comment
}

type githubIssueOrPullRequest struct {
	Typename          string       `json:"__typename"`
	ID                graphql.ID   `json:"id"`
	Number            int          `json:"number"`
	Title             string       `json:"title"`
	Body              string       `json:"body"`
	CreatedAt         time.Time    `json:"createdAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
	BaseRef           *githubRef   `json:"baseRef"`
	BaseRefOid        string       `json:"baseRefOid"`
	HeadRef           *githubRef   `json:"headRef"`
	HeadRefOid        string       `json:"headRefOid"`
	IsCrossRepository bool         `json:"isCrossRepository"`
	URL               string       `json:"url"`
	State             string       `json:"state"`
	Author            *githubActor `json:"author"`
	Comments          struct {
		Nodes []*githubIssueComment `json:"nodes"`
	} `json:"comments"`
}

type githubRef struct {
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
}

type githubIssueComment struct {
	ID        graphql.ID   `json:"id"`
	Author    *githubActor `json:"author"`
	Body      string       `json:"body"`
	URL       string       `json:"url"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

const (
	githubIssueOrPullRequestCommonQuery = `
__typename
id
number
title
body
createdAt
updatedAt
url
state
author {
	...ActorFields
}
comments(first: 10) {
	nodes {
		id
		author { ... ActorFields  }
		body
		url
		createdAt
		updatedAt
	}
}
`
)

func listGitHubRepositoryIssuesAndPullRequests(ctx context.Context, client *github.Client, githubRepositoryID graphql.ID) (results []*githubIssueOrPullRequest, err error) {
	var data struct {
		Node *struct {
			PullRequests, Issues struct {
				Nodes []*githubIssueOrPullRequest
			}
		}
	}
	if err := client.RequestGraphQL(ctx, "", `
query ImportGitHubRepositoryIssuesAndPullRequests($repository: ID!) {
	node(id: $repository) {
		... on Repository {
			issues(first: 100, orderBy: { field: UPDATED_AT, direction: DESC }) {
				nodes {
`+githubIssueOrPullRequestCommonQuery+`
				}
			}
			pullRequests(first: 100, orderBy: { field: UPDATED_AT, direction: DESC }) {
				nodes {
`+githubIssueOrPullRequestCommonQuery+`
					baseRef { name prefix }
					baseRefOid
					headRef { name prefix }
					headRefOid
					isCrossRepository
				}
			}
		}
	}
}
`+githubActorFieldsFragment, map[string]interface{}{
		"repository": githubRepositoryID,
	}, &data); err != nil {
		return nil, err
	}
	if data.Node == nil {
		return nil, fmt.Errorf("github repository with ID %q not found", githubRepositoryID)
	}
	return append(data.Node.Issues.Nodes, data.Node.PullRequests.Nodes...), nil
}
