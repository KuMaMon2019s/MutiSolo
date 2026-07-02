package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	controllayer "Mutesolo/control_layer"
	"Mutesolo/internal/coordination"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	store := coordination.NewStore(coordination.DefaultStatePath())
	if args[0] == "pipeline" {
		return pipelineCommand(args[1:])
	}
	return store.WithState(func(state *coordination.State) (bool, error) {
		switch args[0] {
		case "agents":
			return false, agentsCommand(args[1:], *state)
		case "skills":
			return false, skillsCommand(args[1:], *state)
		case "tasks":
			return tasksCommand(args[1:], state)
		case "events":
			return false, eventsCommand(args[1:], *state)
		default:
			printUsage()
			return false, fmt.Errorf("unknown command %q", args[0])
		}
	})
}

func pipelineCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: opclawctl pipeline run -prompt <prompt>")
	}
	switch args[0] {
	case "run":
		return pipelineRunCommand(args[1:])
	default:
		return fmt.Errorf("unknown pipeline command %q", args[0])
	}
}

func pipelineRunCommand(args []string) error {
	fs := flag.NewFlagSet("pipeline run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	prompt := fs.String("prompt", "", "input prompt")
	artifactDir := fs.String("artifacts", "artifacts", "artifact output directory")
	approveSystem := fs.Bool("approve-system", false, "allow system design artifacts to pass validation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *prompt == "" && len(fs.Args()) > 0 {
		*prompt = strings.Join(fs.Args(), " ")
	}

	result, err := controllayer.RunPipeline(controllayer.PipelineInput{
		Prompt:        *prompt,
		ApproveSystem: *approveSystem,
	}, *artifactDir)
	if err != nil {
		return err
	}

	fmt.Printf("artifact: %s\n", result.Path)
	fmt.Printf("class: %s\n", result.Artifact.Validation.Class)
	fmt.Printf("status: %s\n", result.Artifact.Validation.Status)
	if len(result.Artifact.Validation.Reasons) > 0 {
		fmt.Printf("reasons: %s\n", strings.Join(result.Artifact.Validation.Reasons, "; "))
	}
	return nil
}

func agentsCommand(args []string, state coordination.State) error {
	if len(args) != 1 || args[0] != "list" {
		return errors.New("usage: opclawctl agents list")
	}
	w := newTable()
	fmt.Fprintln(w, "ID\tADDRESS\tSTATUS\tSKILLS")
	for _, agent := range state.Agents {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", agent.ID, agent.Address, agent.Status, strings.Join(agent.Skills, ","))
	}
	return w.Flush()
}

func skillsCommand(args []string, state coordination.State) error {
	if len(args) != 1 || args[0] != "list" {
		return errors.New("usage: opclawctl skills list")
	}
	w := newTable()
	fmt.Fprintln(w, "ID\tVERSION\tCAPABILITIES")
	for _, skill := range state.Skills {
		fmt.Fprintf(w, "%s\t%s\t%s\n", skill.ID, skill.Version, strings.Join(skill.Capabilities, ","))
	}
	return w.Flush()
}

func tasksCommand(args []string, state *coordination.State) (bool, error) {
	if len(args) == 0 {
		return false, errors.New("usage: opclawctl tasks <create|match|assign>")
	}
	switch args[0] {
	case "create":
		return createTaskCommand(args[1:], state)
	case "match":
		return matchTaskCommand(args[1:], state)
	case "assign":
		return assignTaskCommand(args[1:], state)
	default:
		return false, fmt.Errorf("unknown tasks command %q", args[0])
	}
}

func createTaskCommand(args []string, state *coordination.State) (bool, error) {
	fs := flag.NewFlagSet("tasks create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	id := fs.String("id", "", "task id")
	caps := fs.String("caps", "", "comma-separated required capabilities")
	if err := fs.Parse(args); err != nil {
		return false, err
	}
	requiredCaps := splitCaps(*caps)
	requiredCaps = append(requiredCaps, fs.Args()...)
	task, err := coordination.CreateTask(state, *id, requiredCaps)
	if err != nil {
		return false, err
	}
	fmt.Printf("created task %s (%s)\n", task.ID, strings.Join(task.RequiredCaps, ","))
	return true, nil
}

func matchTaskCommand(args []string, state *coordination.State) (bool, error) {
	if len(args) != 1 {
		return false, errors.New("usage: opclawctl tasks match <task>")
	}
	result, err := coordination.MatchTask(state, args[0])
	if err != nil {
		if errors.Is(err, coordination.ErrNoMatch) {
			return true, err
		}
		return false, err
	}
	fmt.Printf("best agent: %s coverage: %.2f matched: %s\n", result.Agent.ID, result.Coverage, strings.Join(result.MatchedCaps, ","))
	return true, nil
}

func assignTaskCommand(args []string, state *coordination.State) (bool, error) {
	if len(args) != 2 {
		return false, errors.New("usage: opclawctl tasks assign <task> <agent>")
	}
	session, err := coordination.AssignTask(state, args[0], args[1])
	if err != nil {
		return false, err
	}
	fmt.Printf("assigned task %s to %s as %s\n", session.TaskID, session.AgentID, session.ID)
	return true, nil
}

func eventsCommand(args []string, state coordination.State) error {
	if len(args) != 1 || args[0] != "tail" {
		return errors.New("usage: opclawctl events tail")
	}
	w := newTable()
	fmt.Fprintln(w, "TIMESTAMP\tTYPE\tENTITY\tPAYLOAD")
	for _, event := range state.Events {
		payload, err := json.Marshal(event.Payload)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", event.Timestamp.Format("2006-01-02T15:04:05Z07:00"), event.Type, event.EntityID, string(payload))
	}
	return w.Flush()
}

func splitCaps(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

func newTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

func printUsage() {
	fmt.Println(`OpenClaw Coordination Layer

Usage:
  opclawctl agents list
  opclawctl skills list
  opclawctl tasks create [-id task-id] -caps cap1,cap2
  opclawctl tasks create [-id task-id] cap1 cap2
  opclawctl tasks match <task>
  opclawctl tasks assign <task> <agent>
  opclawctl events tail
  opclawctl pipeline run -prompt "write a safe helper"

State:
  OPENCLAW_STATE can override the default .openclaw/state.json path.`)
}
