package gitea

import (
	"fmt"

	sdk "code.gitea.io/sdk/gitea"
)

const pageSize = 50

type Client struct {
	client *sdk.Client
}

func NewClient(baseURL, token string) (*Client, error) {
	client, err := sdk.NewClient(baseURL, sdk.SetToken(token))
	if err != nil {
		return nil, fmt.Errorf("create gitea client for %s: %w", baseURL, err)
	}
	return &Client{client: client}, nil
}

func (c *Client) CurrentUser() (*sdk.User, error) {
	user, _, err := c.client.GetMyUserInfo()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	return user, nil
}

func (c *Client) IssuesWithDueDate() ([]*sdk.Issue, error) {
	var issues []*sdk.Issue
	opt := sdk.ListIssueOption{
		PageSize: pageSize,
		State:    sdk.StateOpen,
		Type:     sdk.IssueTypeIssue,
	}
	for {
		list, resp, err := c.client.ListIssues(opt)
		if err != nil {
			return nil, fmt.Errorf("list issues: %w", err)
		}
		for _, issue := range list {
			if issue.Deadline != nil {
				issues = append(issues, issue)
			}
		}
		if len(list) < pageSize || resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return issues, nil
}
