package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/mturk"
)

type Workflow struct {
	Title       string
	Description string
	Tags        string
	Reward      string

	Input  sources.SourceConfig
	Output sources.SourceConfig

	Fields []Field

	Tasks map[string]*Task

	MTurk struct {
		HitTypeId string
	}

	client *mturk.MTurk
}

func (w *Workflow) Config() {
	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("File error: %v\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(file, w)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	endpoint := SandboxEndpoint
	if isLive {
		endpoint = LiveEndpoint
	}

	fmt.Println("Endpoint:", endpoint)

	sess := session.Must(session.NewSession())
	w.client = mturk.New(sess, &aws.Config{
		Credentials: credentials.NewSharedCredentials("/Users/abhi/.aws/credentials", "opszero_mturk"),
		Endpoint:    aws.String(endpoint),
		Region:      aws.String("us-east-1"),
	})

	resp, err := w.client.GetAccountBalance(nil)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp)

	w.Input.Init()

	if w.Tasks == nil {
		w.Tasks = make(map[string]*Task)
	}
}

func (w *Workflow) Save() {
	b, err := json.MarshalIndent(w, "", "    ")
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile(configFile, b, os.ModePerm)
}

func (w *Workflow) BuildHit() {
	if w.MTurk.HitTypeId != "" {
		// Update HIT here
		return
	}

	resp, err := w.client.CreateHITType(&mturk.CreateHITTypeInput{
		AssignmentDurationInSeconds: aws.Int64(3000),
		AutoApprovalDelayInSeconds:  aws.Int64(86400),
		Title:                       aws.String(w.Title),
		Description:                 aws.String(w.Description),
		Keywords:                    aws.String(w.Tags),
		Reward:                      aws.String(w.Reward),
	})

	if err != nil {
		fmt.Println(err)
	}

	w.MTurk.HitTypeId = *resp.HITTypeId
}

func (w *Workflow) BuildTasks() {
	records := w.Input.Records()

	for i := range records {
		log.Println("Me", records[i])

		t, ok := w.Tasks[records[i]["ID"]]

		if !ok {
			log.Println("Creating new task")

			t = &Task{
				Title:       w.Title,
				Description: w.Description,
				SourceID:    records[i]["ID"],
			}

			t.New(w, records[i])
		} else {
			log.Println("Updating task")
			t.Update(w, records[i])
		}

		w.Save()
	}
}

func (w *Workflow) SaveOutput() {
	var (
		header []string
		rows   []map[string]string
	)

	header = []string{"ID"}

	for i := range w.Fields {
		header = append(header, w.Fields[i].Name)
	}

	for _, t := range w.Tasks {
		if len(t.MTurk.Assignments) > 0 {
			row := make(map[string]string)
			row["ID"] = t.SourceID

			for _, field := range t.Fields {
				row[field.Name] = field.Value
			}

			rows = append(rows, row)
		}
	}

	w.Output.WriteAll(header, rows)
}

func (w *Workflow) ReviewTasks() {
	workerTasks := make(map[string][]*Task)

	for _, t := range w.Tasks {
		if len(t.MTurk.Assignments) > 0 && *t.MTurk.Assignments[0].AssignmentStatus == "Submitted" {
			workerTasks[*t.MTurk.Assignments[0].WorkerId] = append(workerTasks[*t.MTurk.Assignments[0].WorkerId], t)
		}
	}

	for workerId, t := range workerTasks {
		fmt.Println("WorkerID: ", workerId)
		fmt.Printf(`aws --region us-east-1 --profile opszero_mturk mturk create-worker-block --worker-id %s --reason "Spammy Work, Incorrect Work"`, workerId)
		fmt.Println("\n")
		for _, task := range t {
			for _, f := range task.Fields {
				fmt.Println(f)
			}
			fmt.Printf(`aws --region us-east-1 --profile opszero_mturk mturk reject-assignment --assignment-id %s --requester-feedback "Didn't do work"`, *task.MTurk.Assignments[0].AssignmentId)
			fmt.Println("\n")
		}

		fmt.Println("Approve All (a), Reject All (r), Next (n)")

		switch getchar() {
		case 'a':
			for _, workerTask := range t {
				workerTask.Approve(w)
			}
		case 'r':
			for _, workerTask := range t {
				workerTask.Reject(w)
			}
		}
	}
}

func getchar() byte {
	reader := bufio.NewReader(os.Stdin)
	char, _ := reader.ReadByte()

	return char
}

func (w *Workflow) ExpireTasks() {
	for _, t := range w.Tasks {
		t.Expire(w)
		w.Save()
	}
}
