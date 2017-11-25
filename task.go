package main

import (
	"encoding/xml"
	"fmt"
	"html"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mturk"
)

type questionFormAnswers struct {
	Answer []struct {
		QuestionIdentifier string
		FreeText           string
	}
}

type Task struct {
	// Copied from Workflow
	Title       string
	Description string

	HitID    string
	SourceID string
	Fields   []Field

	MTurk struct {
		QuestionFormAnswers questionFormAnswers
		Assignments         []*mturk.Assignment
	}
}

func (t *Task) Question() string {
	var fieldsHTML string
	for i := range t.Fields {
		fieldsHTML += t.Fields[i].HTML()
	}

	return fmt.Sprintf(`
<HTMLQuestion xmlns="http://mechanicalturk.amazonaws.com/AWSMechanicalTurkDataSchemas/2011-11-11/HTMLQuestion.xsd">
  <HTMLContent><![CDATA[
<!DOCTYPE html>
<html>
 <head>
  <meta http-equiv='Content-Type' content='text/html; charset=UTF-8'/>
  <script type='text/javascript' src='https://s3.amazonaws.com/mturk-public/externalHIT_v1.js'></script>

<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
<script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js" integrity="sha384-Tc5IQib027qvyjSMfHjOMaLkfuWVxZxUPnCJA7l2mCWNIpG9mGCD8wGNIcPD7Txa" crossorigin="anonymous"></script>

 </head>
 <body>
  <div class="container">
    <form name='mturk_form' method='post' id='mturk_form' action='https://www.mturk.com/mturk/externalSubmit'>
    <h1>%s</h1>

    <p>
    %s
    </p>

    %s

    <p>
	<input type='hidden' value='' name='assignmentId' id='assignmentId'/>
	<input type='submit' id='submitButton' value='Submit' class='btn btn-success' />
    </p>
    </form>
  </div>

  <script language='Javascript'>turkSetAssignmentID();</script>
 </body>
</html>
]]>
  </HTMLContent>
  <FrameHeight>1000</FrameHeight>
</HTMLQuestion>
`, html.EscapeString(t.Title), html.EscapeString(t.Description), fieldsHTML)
}

func (t *Task) New(w *Workflow, record map[string]string) {
	for _, workflowField := range w.Fields {
		workflowField.Value = record[workflowField.Name]
		t.Fields = append(t.Fields, workflowField)
	}

	resp, err := w.client.CreateHITWithHITType(&mturk.CreateHITWithHITTypeInput{
		HITTypeId:         aws.String(w.MTurk.HitTypeId),
		MaxAssignments:    aws.Int64(1),
		Question:          aws.String(t.Question()),
		LifetimeInSeconds: aws.Int64(86400), // 1 day
	})

	if err == nil {
		t.HitID = *resp.HIT.HITId
		w.Tasks[t.SourceID] = t
	} else {
		fmt.Println(err)
		if r := recover(); r != nil {
			fmt.Println("Recovered", r)
		}
	}
}

func (t *Task) Update(w *Workflow, records []map[string]string, i int) {
	// UpdateHITTypeOfHIT
	for field := range t.Fields {
		f := &t.Fields[field]
		f.Value = records[i][f.Name]
	}

	resp, err := w.client.ListAssignmentsForHIT(&mturk.ListAssignmentsForHITInput{
		HITId: aws.String(t.HitID),
	})

	fmt.Println(err)
	fmt.Println(resp)

	t.MTurk.Assignments = resp.Assignments

	if len(resp.Assignments) > 0 {
		var q questionFormAnswers

		xml.Unmarshal([]byte(*resp.Assignments[0].Answer), &q)
		t.MTurk.QuestionFormAnswers = q

		for field := range t.Fields {
			f := &t.Fields[field]

			for _, answer := range t.MTurk.QuestionFormAnswers.Answer {
				if f.Name == answer.QuestionIdentifier {
					f.Value = strings.TrimSpace(answer.FreeText)
				}
			}
		}

	}
}
