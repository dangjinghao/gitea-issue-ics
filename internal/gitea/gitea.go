package gitea

import (
	"fmt"
	"net/url"
	"strings"

	sdk "code.gitea.io/sdk/gitea"
)

const pageSize = 50

type Client struct {
	client   *sdk.Client
	basePath string
}

func NewClient(baseURL, token string) (*Client, error) {
	client, err := sdk.NewClient(baseURL, sdk.SetToken(token))
	if err != nil {
		return nil, fmt.Errorf("create gitea client for %s: %w", baseURL, err)
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse Gitea base URL: %w", err)
	}
	return &Client{client: client, basePath: strings.TrimRight(base.EscapedPath(), "/")}, nil
}

// RenderIssueBody uses comment mode so references are resolved in the issue's
// repository, including the instance's deployment subpath.
func (c *Client) RenderIssueBody(issue *sdk.Issue) (string, error) {
	if strings.TrimSpace(issue.Body) == "" {
		return "", nil
	}
	if issue.Repository == nil || issue.Repository.FullName == "" {
		return "", fmt.Errorf("render issue %d: missing repository name", issue.ID)
	}
	html, _, err := c.client.RenderMarkup(sdk.MarkupOption{
		Text:    issue.Body,
		Mode:    "comment",
		Context: c.basePath + "/" + issue.Repository.FullName,
	})
	if err != nil {
		return "", fmt.Errorf("render issue %d: %w", issue.ID, err)
	}
	return html, nil
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
