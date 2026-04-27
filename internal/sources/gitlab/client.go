/*
Copyright 2026 Devon Warren.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package gitlab wraps the official GitLab Go client for the operations the
// controller needs: minting and revoking project access tokens.
package gitlab

import (
	"context"
	"fmt"
	"time"

	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/devonwarren/token-rotator/api/v1alpha1"
)

// Client mints and revokes GitLab tokens. A zero Client is not usable;
// construct one with NewClient.
type Client struct {
	api *gl.Client
}

// NewClient builds an authenticated GitLab client. baseURL may be empty for
// gitlab.com; otherwise it points at a self-hosted instance (e.g.
// "https://gitlab.example.com").
func NewClient(apiToken, baseURL string) (*Client, error) {
	var opts []gl.ClientOptionFunc
	if baseURL != "" {
		opts = append(opts, gl.WithBaseURL(baseURL))
	}
	api, err := gl.NewClient(apiToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("construct gitlab client: %w", err)
	}
	return &Client{api: api}, nil
}

// MintedToken is what successful rotation returns.
type MintedToken struct {
	ID        int64
	Value     string
	ExpiresAt time.Time
}

// MintProjectAccessToken creates a new project access token on the GitLab
// project named in spec.Project with the supplied name and expiry.
func (c *Client) MintProjectAccessToken(
	ctx context.Context,
	spec v1alpha1.GitLabProjectAccessTokenSpec,
	name string,
	expiresAt time.Time,
) (*MintedToken, error) {
	level, err := toAccessLevel(spec.AccessLevel)
	if err != nil {
		return nil, err
	}
	expiry := gl.ISOTime(expiresAt)
	opts := &gl.CreateProjectAccessTokenOptions{
		Name:        gl.Ptr(name),
		Scopes:      gl.Ptr(spec.Scopes),
		AccessLevel: gl.Ptr(level),
		ExpiresAt:   &expiry,
	}

	token, _, err := c.api.ProjectAccessTokens.CreateProjectAccessToken(
		spec.Project, opts, gl.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("create project access token: %w", err)
	}

	var expiresAtResp time.Time
	if token.ExpiresAt != nil {
		expiresAtResp = time.Time(*token.ExpiresAt)
	}
	return &MintedToken{
		ID:        token.ID,
		Value:     token.Token,
		ExpiresAt: expiresAtResp,
	}, nil
}

// RevokeProjectAccessToken invalidates a previously-minted token by its ID.
func (c *Client) RevokeProjectAccessToken(
	ctx context.Context, project string, tokenID int64,
) error {
	if _, err := c.api.ProjectAccessTokens.RevokeProjectAccessToken(
		project, tokenID, gl.WithContext(ctx),
	); err != nil {
		return fmt.Errorf("revoke project access token %d: %w", tokenID, err)
	}
	return nil
}

func toAccessLevel(lvl v1alpha1.GitLabAccessLevel) (gl.AccessLevelValue, error) {
	switch lvl {
	case v1alpha1.GitLabAccessLevelGuest:
		return gl.GuestPermissions, nil
	case v1alpha1.GitLabAccessLevelReporter:
		return gl.ReporterPermissions, nil
	case v1alpha1.GitLabAccessLevelDeveloper:
		return gl.DeveloperPermissions, nil
	case v1alpha1.GitLabAccessLevelMaintainer:
		return gl.MaintainerPermissions, nil
	case v1alpha1.GitLabAccessLevelOwner:
		return gl.OwnerPermissions, nil
	}
	return 0, fmt.Errorf("unknown GitLab access level %q", lvl)
}
