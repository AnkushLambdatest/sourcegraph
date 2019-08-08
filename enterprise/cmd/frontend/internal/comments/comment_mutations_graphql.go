package comments

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/events"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/comments/commentobjectdb"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/comments/internal"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/comments/types"
)

func CommentActorFromContext(ctx context.Context) (authorUserID int32, err error) {
	actor, err := graphqlbackend.CurrentUser(ctx)
	if err != nil {
		return 0, err
	}
	if actor == nil {
		return 0, errors.New("authenticated required to create comment")
	}
	return actor.DatabaseID(), nil
}

func (GraphQLResolver) AddCommentReply(ctx context.Context, arg *graphqlbackend.AddCommentReplyArgs) (graphqlbackend.Comment, error) {
	// TODO!(sqs): add auth checks
	authorUserID, err := CommentActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	parentComment, err := commentByGQLID(ctx, arg.Input.ParentComment)
	if err != nil {
		return nil, err
	}

	v := &internal.DBComment{
		AuthorUserID: authorUserID,
		Body:         arg.Input.Body,
		Object:       types.CommentObject{ParentCommentID: parentComment.ID},
	}
	comment, err := internal.DBComments{}.Create(ctx, nil, v)
	if err != nil {
		return nil, err
	}

	if err := createCommentEventForReply(ctx, nil, comment.ID, authorUserID, "", "", v.CreatedAt); err != nil {
		return nil, err
	}

	return newGQLToComment(ctx, comment)
}

func createCommentEventForReply(ctx context.Context, tx *sql.Tx, commentID int64, actorUserID int32, externalActorUsername, externalActorURL string, createdAt time.Time) error {
	return events.CreateEvent(ctx, tx, events.CreationData{
		Type:                  EventTypeComment,
		Objects:               events.Objects{Comment: commentID},
		ActorUserID:           actorUserID,
		ExternalActorUsername: externalActorUsername,
		ExternalActorURL:      externalActorURL,
		CreatedAt:             createdAt,
	})
}

type ExternalComment struct {
	ThreadPrimaryCommentID int64
	commentobjectdb.DBObjectCommentFields
}

// TODO!(sqs) hack
func CreateExternalCommentReply(ctx context.Context, tx *sql.Tx, comment ExternalComment) error {
	v := &internal.DBComment{
		Object:                      types.CommentObject{ParentCommentID: comment.ThreadPrimaryCommentID},
		AuthorExternalActorUsername: sql.NullString{String: comment.AuthorExternalActorUsername, Valid: true},
		AuthorExternalActorURL:      sql.NullString{String: comment.AuthorExternalActorURL, Valid: true},
		Body:                        comment.Body,
		CreatedAt:                   comment.CreatedAt,
		UpdatedAt:                   comment.UpdatedAt,
	}
	dbComment, err := internal.DBComments{}.Create(ctx, tx, v)
	if err != nil {
		return err
	}
	return createCommentEventForReply(ctx, tx, dbComment.ID, 0, comment.AuthorExternalActorUsername, comment.AuthorExternalActorURL, comment.CreatedAt)
}

func (GraphQLResolver) EditComment(ctx context.Context, arg *graphqlbackend.EditCommentArgs) (graphqlbackend.Comment, error) {
	v, err := commentByGQLID(ctx, arg.Input.ID)
	if err != nil {
		return nil, err
	}
	comment, err := internal.DBComments{}.Update(ctx, v.ID, internal.DBCommentUpdate{
		Body: &arg.Input.Body,
	})
	if err != nil {
		return nil, err
	}
	return newGQLToComment(ctx, comment)
}

func (GraphQLResolver) DeleteComment(ctx context.Context, arg *graphqlbackend.DeleteCommentArgs) (*graphqlbackend.EmptyResponse, error) {
	v, err := commentByGQLID(ctx, arg.Comment)
	if err != nil {
		return nil, err
	}
	return nil, internal.DBComments{}.DeleteByID(ctx, v.ID)
}
