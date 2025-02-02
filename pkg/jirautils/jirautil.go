package jirautils

import (
	"bytes"
	"github.com/portworx/torpedo/pkg/log"
	"net/http"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/trivago/tgo/tcontainer"
)

var (
	client                     *jira.Client
	isJiraConnectionSuccessful bool

	//AccountID for bug assignment
	AccountID string
)

const (
	jiraURL = "https://portworx.atlassian.net/"
)

// Init function for the Jira
func Init(username, token string) {
	jiraAuth := jira.BasicAuthTransport{
		Username: username,
		Password: token,
	}
	var err error
	timeout := 15 * time.Second
	httpClient := http.Client{
		Timeout: timeout,
	}
	response, err := httpClient.Get(jiraURL)
	if err == nil {
		isJiraConnectionSuccessful = true
	}
	if response != nil && response.StatusCode != 200 {
		log.Warnf("Response code : %d", response.StatusCode)
		isJiraConnectionSuccessful = false
	}

	if isJiraConnectionSuccessful {
		client, err = jira.NewClient(jiraAuth.Client(), jiraURL)
	} else {
		log.Errorf("Jira connection not successful, Cause: %v", err)
	}

	log.Info("Jira connection is successful")

}

// CreateIssue creates issue in jira
func CreateIssue(issueDesription, issueSummary string) (string, error) {

	issueKey := ""
	var err error
	if isJiraConnectionSuccessful && client != nil {
		issueKey, err = createPTX(issueDesription, issueSummary)
	} else {
		log.Warn("Skipping issue creation as jira connection is not successful")
	}
	return issueKey, err

}

func getPTX(issueID string) {

	issue, _, err := client.Issue.Get(issueID, nil)
	log.Infof("Error: %v", err)

	log.Infof("%s: %+v\n", issue.Key, issue.Fields.Summary)

	log.Infof("%s: %s\n", issue.ID, issue.Fields.Summary)
	log.Info(issue.Fields.FixVersions[0].Name)

}

func createPTX(description, summary string) (string, error) {

	//Hardcoding the Priority to P1
	customFieldsMap := tcontainer.NewMarshalMap()
	customFieldsMap["customfield_11115"] = map[string]interface{}{
		"id":    "10936",
		"value": "P1 (High)",
	}

	i := jira.Issue{
		Fields: &jira.IssueFields{
			Assignee: &jira.User{
				AccountID: AccountID,
			},
			Description: description,
			Type: jira.IssueType{
				Name: "Bug",
			},
			Project: jira.Project{
				Key: "PTX",
			},
			FixVersions: []*jira.FixVersion{
				{
					Name: "master",
				},
			},
			AffectsVersions: []*jira.AffectsVersion{
				{
					Name: "master",
				},
			},
			Summary:  summary,
			Unknowns: customFieldsMap,
		},
	}
	issue, resp, err := client.Issue.Create(&i)

	log.Infof("Resp: %v", resp.StatusCode)
	issueKey := ""
	if resp.StatusCode == 201 {
		log.Info("Successfully created new jira issue.")
		log.Infof("Jira Issue: %+v\n", issue.Key)
		issueKey = issue.Key

	} else {
		log.Infof("Error while creating jira issue: %v", err)
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		newStr := buf.String()
		log.Infof(newStr)

	}
	return issueKey, err

}

func getProjects() {
	req, _ := client.NewRequest("GET", "rest/api/3/project/recent", nil)

	projects := new([]jira.Project)
	_, err := client.Do(req, projects)
	if err != nil {
		log.Info("Error while getting project")
		log.Error(err)
		return
	}

	for _, project := range *projects {

		log.Infof("%s: %s\n", project.Key, project.Name)
	}
}
