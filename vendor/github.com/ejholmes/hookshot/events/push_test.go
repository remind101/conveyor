package events

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestCanDecodeOldPushPayload(t *testing.T) {
	push := &Push{}
	err := json.Unmarshal([]byte(OldPushPayload), &push)
	if err != nil {
		t.Fatalf("Could not decode the payload")
	}
	expectedTime := &Timestamp{time.Unix(1370647950, 0)}
	if !reflect.DeepEqual(push.Repository.CreatedAt, expectedTime) {
		t.Fatalf("Expected %#v but got %#v", expectedTime, push.Repository.CreatedAt)
	}

	expectedTime = &Timestamp{time.Unix(1503384499, 0)}
	if !reflect.DeepEqual(push.Repository.PushedAt, expectedTime) {
		t.Fatalf("Expected %#v but got %#v", expectedTime, push.Repository.PushedAt)
	}
}

func TestCanDecodeNewPushPayload(t *testing.T) {
	push := &Push{}
	err := json.Unmarshal([]byte(NewPushPayload), &push)
	if err != nil {
		t.Fatalf("Could not decode the payload")
	}

	parsedTime, _ := time.Parse(time.RFC3339, "2013-06-07T23:32:30Z")
	expectedTime := &Timestamp{parsedTime}
	if !reflect.DeepEqual(push.Repository.CreatedAt, expectedTime) {
		t.Fatalf("Expected %#v but got %#v", expectedTime, push.Repository.CreatedAt)
	}

	parsedTime, _ = time.Parse(time.RFC3339, "2017-08-22T16:05:23.656Z")
	expectedTime = &Timestamp{parsedTime}
	if !reflect.DeepEqual(push.Repository.PushedAt, expectedTime) {
		t.Fatalf("Expected %#v but got %#v", expectedTime, push.Repository.PushedAt)
	}
}

// Non "repository" fields removed from these payloads because they contained sensitive information.
const OldPushPayload = `{
    "repository": {
        "archive_url": "https://api.github.com/repos/exampleorg/somerepo/{archive_format}{/ref}",
        "assignees_url": "https://api.github.com/repos/exampleorg/somerepo/assignees{/user}",
        "blobs_url": "https://api.github.com/repos/exampleorg/somerepo/git/blobs{/sha}",
        "branches_url": "https://api.github.com/repos/exampleorg/somerepo/branches{/branch}",
        "clone_url": "https://github.com/exampleorg/somerepo.git",
        "collaborators_url": "https://api.github.com/repos/exampleorg/somerepo/collaborators{/collaborator}",
        "comments_url": "https://api.github.com/repos/exampleorg/somerepo/comments{/number}",
        "commits_url": "https://api.github.com/repos/exampleorg/somerepo/commits{/sha}",
        "compare_url": "https://api.github.com/repos/exampleorg/somerepo/compare/{base}...{head}",
        "contents_url": "https://api.github.com/repos/exampleorg/somerepo/contents/{+path}",
        "contributors_url": "https://api.github.com/repos/exampleorg/somerepo/contributors",
        "created_at": 1370647950,
        "default_branch": "master",
        "deployments_url": "https://api.github.com/repos/exampleorg/somerepo/deployments",
        "description": "Example's heart and soul",
        "downloads_url": "https://api.github.com/repos/exampleorg/somerepo/downloads",
        "events_url": "https://api.github.com/repos/exampleorg/somerepo/events",
        "fork": false,
        "forks": 0,
        "forks_count": 0,
        "forks_url": "https://api.github.com/repos/exampleorg/somerepo/forks",
        "full_name": "exampleorg/somerepo",
        "git_commits_url": "https://api.github.com/repos/exampleorg/somerepo/git/commits{/sha}",
        "git_refs_url": "https://api.github.com/repos/exampleorg/somerepo/git/refs{/sha}",
        "git_tags_url": "https://api.github.com/repos/exampleorg/somerepo/git/tags{/sha}",
        "git_url": "git://github.com/exampleorg/somerepo.git",
        "has_downloads": true,
        "has_issues": false,
        "has_pages": false,
        "has_projects": true,
        "has_wiki": false,
        "homepage": "https://api.example.com",
        "hooks_url": "https://api.github.com/repos/exampleorg/somerepo/hooks",
        "html_url": "https://github.com/exampleorg/somerepo",
        "id": 10560697,
        "issue_comment_url": "https://api.github.com/repos/exampleorg/somerepo/issues/comments{/number}",
        "issue_events_url": "https://api.github.com/repos/exampleorg/somerepo/issues/events{/number}",
        "issues_url": "https://api.github.com/repos/exampleorg/somerepo/issues{/number}",
        "keys_url": "https://api.github.com/repos/exampleorg/somerepo/keys{/key_id}",
        "labels_url": "https://api.github.com/repos/exampleorg/somerepo/labels{/name}",
        "language": "Ruby",
        "languages_url": "https://api.github.com/repos/exampleorg/somerepo/languages",
        "master_branch": "master",
        "merges_url": "https://api.github.com/repos/exampleorg/somerepo/merges",
        "milestones_url": "https://api.github.com/repos/exampleorg/somerepo/milestones{/number}",
        "mirror_url": null,
        "name": "somerepo",
        "notifications_url": "https://api.github.com/repos/exampleorg/somerepo/notifications{?since,all,participating}",
        "open_issues": 60,
        "open_issues_count": 60,
        "organization": "exampleorg",
        "owner": {
            "avatar_url": "https://avatars1.githubusercontent.com/u/376343?v=4",
            "email": "contact@example.com",
            "events_url": "https://api.github.com/users/exampleorg/events{/privacy}",
            "followers_url": "https://api.github.com/users/exampleorg/followers",
            "following_url": "https://api.github.com/users/exampleorg/following{/other_user}",
            "gists_url": "https://api.github.com/users/exampleorg/gists{/gist_id}",
            "gravatar_id": "",
            "html_url": "https://github.com/exampleorg",
            "id": 376343,
            "login": "exampleorg",
            "name": "exampleorg",
            "organizations_url": "https://api.github.com/users/exampleorg/orgs",
            "received_events_url": "https://api.github.com/users/exampleorg/received_events",
            "repos_url": "https://api.github.com/users/exampleorg/repos",
            "site_admin": false,
            "starred_url": "https://api.github.com/users/exampleorg/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/exampleorg/subscriptions",
            "type": "Organization",
            "url": "https://api.github.com/users/exampleorg"
        },
        "private": true,
        "pulls_url": "https://api.github.com/repos/exampleorg/somerepo/pulls{/number}",
        "pushed_at": 1503384499,
        "releases_url": "https://api.github.com/repos/exampleorg/somerepo/releases{/id}",
        "size": 70573,
        "ssh_url": "git@github.com:exampleorg/somerepo.git",
        "stargazers": 5,
        "stargazers_count": 5,
        "stargazers_url": "https://api.github.com/repos/exampleorg/somerepo/stargazers",
        "statuses_url": "https://api.github.com/repos/exampleorg/somerepo/statuses/{sha}",
        "subscribers_url": "https://api.github.com/repos/exampleorg/somerepo/subscribers",
        "subscription_url": "https://api.github.com/repos/exampleorg/somerepo/subscription",
        "svn_url": "https://github.com/exampleorg/somerepo",
        "tags_url": "https://api.github.com/repos/exampleorg/somerepo/tags",
        "teams_url": "https://api.github.com/repos/exampleorg/somerepo/teams",
        "trees_url": "https://api.github.com/repos/exampleorg/somerepo/git/trees{/sha}",
        "updated_at": "2017-05-11T22:18:54Z",
        "url": "https://github.com/exampleorg/somerepo",
        "watchers": 5,
        "watchers_count": 5
    }
}`

const NewPushPayload = `{
    "repository": {
        "archive_url": "https://api.github.com/repos/exampleorg/somerepo/{archive_format}{/ref}",
        "assignees_url": "https://api.github.com/repos/exampleorg/somerepo/assignees{/user}",
        "blobs_url": "https://api.github.com/repos/exampleorg/somerepo/git/blobs{/sha}",
        "branches_url": "https://api.github.com/repos/exampleorg/somerepo/branches{/branch}",
        "clone_url": "https://github.com/exampleorg/somerepo.git",
        "collaborators_url": "https://api.github.com/repos/exampleorg/somerepo/collaborators{/collaborator}",
        "comments_url": "https://api.github.com/repos/exampleorg/somerepo/comments{/number}",
        "commits_url": "https://api.github.com/repos/exampleorg/somerepo/commits{/sha}",
        "compare_url": "https://api.github.com/repos/exampleorg/somerepo/compare/{base}...{head}",
        "contents_url": "https://api.github.com/repos/exampleorg/somerepo/contents/{+path}",
        "contributors_url": "https://api.github.com/repos/exampleorg/somerepo/contributors",
        "created_at": "2013-06-07T23:32:30Z",
        "default_branch": "master",
        "deployments_url": "https://api.github.com/repos/exampleorg/somerepo/deployments",
        "description": "Example's heart and soul",
        "downloads_url": "https://api.github.com/repos/exampleorg/somerepo/downloads",
        "events_url": "https://api.github.com/repos/exampleorg/somerepo/events",
        "fork": false,
        "forks": 0,
        "forks_count": 0,
        "forks_url": "https://api.github.com/repos/exampleorg/somerepo/forks",
        "full_name": "exampleorg/somerepo",
        "git_commits_url": "https://api.github.com/repos/exampleorg/somerepo/git/commits{/sha}",
        "git_refs_url": "https://api.github.com/repos/exampleorg/somerepo/git/refs{/sha}",
        "git_tags_url": "https://api.github.com/repos/exampleorg/somerepo/git/tags{/sha}",
        "git_url": "git://github.com/exampleorg/somerepo.git",
        "has_downloads": true,
        "has_issues": false,
        "has_pages": false,
        "has_projects": true,
        "has_wiki": false,
        "homepage": "https://api.example.com",
        "hooks_url": "https://api.github.com/repos/exampleorg/somerepo/hooks",
        "html_url": "https://github.com/exampleorg/somerepo",
        "id": 10560697,
        "issue_comment_url": "https://api.github.com/repos/exampleorg/somerepo/issues/comments{/number}",
        "issue_events_url": "https://api.github.com/repos/exampleorg/somerepo/issues/events{/number}",
        "issues_url": "https://api.github.com/repos/exampleorg/somerepo/issues{/number}",
        "keys_url": "https://api.github.com/repos/exampleorg/somerepo/keys{/key_id}",
        "labels_url": "https://api.github.com/repos/exampleorg/somerepo/labels{/name}",
        "language": "Ruby",
        "languages_url": "https://api.github.com/repos/exampleorg/somerepo/languages",
        "master_branch": "master",
        "merges_url": "https://api.github.com/repos/exampleorg/somerepo/merges",
        "milestones_url": "https://api.github.com/repos/exampleorg/somerepo/milestones{/number}",
        "mirror_url": null,
        "name": "somerepo",
        "notifications_url": "https://api.github.com/repos/exampleorg/somerepo/notifications{?since,all,participating}",
        "open_issues": 61,
        "open_issues_count": 61,
        "organization": "exampleorg",
        "owner": {
            "avatar_url": "https://avatars1.githubusercontent.com/u/376343?v=4",
            "email": "contact@example.com",
            "events_url": "https://api.github.com/users/exampleorg/events{/privacy}",
            "followers_url": "https://api.github.com/users/exampleorg/followers",
            "following_url": "https://api.github.com/users/exampleorg/following{/other_user}",
            "gists_url": "https://api.github.com/users/exampleorg/gists{/gist_id}",
            "gravatar_id": "",
            "html_url": "https://github.com/exampleorg",
            "id": 376343,
            "login": "exampleorg",
            "name": "exampleorg",
            "organizations_url": "https://api.github.com/users/exampleorg/orgs",
            "received_events_url": "https://api.github.com/users/exampleorg/received_events",
            "repos_url": "https://api.github.com/users/exampleorg/repos",
            "site_admin": false,
            "starred_url": "https://api.github.com/users/exampleorg/starred{/owner}{/repo}",
            "subscriptions_url": "https://api.github.com/users/exampleorg/subscriptions",
            "type": "Organization",
            "url": "https://api.github.com/users/exampleorg"
        },
        "private": true,
        "pulls_url": "https://api.github.com/repos/exampleorg/somerepo/pulls{/number}",
        "pushed_at": "2017-08-22T16:05:23.656Z",
        "releases_url": "https://api.github.com/repos/exampleorg/somerepo/releases{/id}",
        "size": 70575,
        "ssh_url": "git@github.com:exampleorg/somerepo.git",
        "stargazers": 5,
        "stargazers_count": 5,
        "stargazers_url": "https://api.github.com/repos/exampleorg/somerepo/stargazers",
        "statuses_url": "https://api.github.com/repos/exampleorg/somerepo/statuses/{sha}",
        "subscribers_url": "https://api.github.com/repos/exampleorg/somerepo/subscribers",
        "subscription_url": "https://api.github.com/repos/exampleorg/somerepo/subscription",
        "svn_url": "https://github.com/exampleorg/somerepo",
        "tags_url": "https://api.github.com/repos/exampleorg/somerepo/tags",
        "teams_url": "https://api.github.com/repos/exampleorg/somerepo/teams",
        "trees_url": "https://api.github.com/repos/exampleorg/somerepo/git/trees{/sha}",
        "updated_at": "2017-05-11T22:18:54Z",
        "url": "https://github.com/exampleorg/somerepo",
        "watchers": 5,
        "watchers_count": 5
    }
}`
