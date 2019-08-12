package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/worker/pkg/types"
)

func (s *SQLStore) GetUser(ctx context.Context, userID string) (types.User, error) {
	githubUser, err := s.GetGitHubUser(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "get github user")
	}

	if githubUser != nil {
		return *githubUser, nil
	}

	passwordUser, err := s.GetPasswordUser(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "get password user")
	}

	if passwordUser != nil {
		return *passwordUser, nil
	}

	return nil, errors.New("unknown user type")
}

func (s *SQLStore) GetGitHubUser(ctx context.Context, userID string) (*types.GitHubUser, error) {
	query := `select user_id, username from github_user where user_id = $1`
	row := s.db.QueryRowContext(ctx, query, userID)

	githubUser := types.GitHubUser{}
	if err := row.Scan(&githubUser.ID, &githubUser.Username); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "scan")
	}

	return &githubUser, nil
}

func (s *SQLStore) GetPasswordUser(ctx context.Context, userID string) (*types.PasswordUser, error) {
	query := `select user_id, email from ship_user_local where user_id = $1`
	row := s.db.QueryRowContext(ctx, query, userID)

	passwordUser := types.PasswordUser{}
	if err := row.Scan(&passwordUser.ID, &passwordUser.Email); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "scan")
	}

	return &passwordUser, nil
}
