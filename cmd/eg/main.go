package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"mind-zero-five/internal/db"
	"mind-zero-five/pkg/actor"
	"mind-zero-five/pkg/authority"
	"mind-zero-five/pkg/eventgraph"
	"mind-zero-five/pkg/task"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		fatal("connect: %v", err)
	}
	defer pool.Close()

	events := eventgraph.NewPgStore(pool)
	tasks := task.NewPgStore(pool)
	auth := authority.NewPgStore(pool)
	actors := actor.NewPgStore(pool)

	switch os.Args[1] {
	case "event":
		handleEvent(ctx, events, os.Args[2:])
	case "task":
		handleTask(ctx, tasks, os.Args[2:])
	case "authority":
		handleAuthority(ctx, auth, os.Args[2:])
	case "actor":
		handleActor(ctx, actors, os.Args[2:])
	case "policy":
		handlePolicy(ctx, auth, os.Args[2:])
	case "status":
		handleStatus(ctx, events, tasks, auth)
	case "init":
		handleInit(ctx, events, tasks, auth, actors)
	default:
		usage()
		os.Exit(1)
	}
}

func handleEvent(ctx context.Context, store eventgraph.EventStore, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: eg event <create|list|get|ancestors|descendants|search|types|sources|verify> [--format=short for list/search]")
		os.Exit(1)
	}

	switch args[0] {
	case "create":
		flags := parseFlags(args[1:])
		eventType := flags["type"]
		source := flags["source"]
		if eventType == "" {
			fatal("--type is required")
		}
		if source == "" {
			source = "mind"
		}
		var content map[string]any
		if c := flags["content"]; c != "" {
			if err := json.Unmarshal([]byte(c), &content); err != nil {
				fatal("parse content JSON: %v", err)
			}
		}
		var causes []string
		if c := flags["causes"]; c != "" {
			causes = strings.Split(c, ",")
		}
		conversationID := flags["conversation"]
		e, err := store.Append(ctx, eventType, source, content, causes, conversationID)
		if err != nil {
			fatal("create event: %v", err)
		}
		printJSON(e)

	case "list":
		flags := parseFlags(args[1:])
		limit := intFlag(flags, "limit", 20)
		short := flags["format"] == "short"
		var events []eventgraph.Event
		var err error
		if t := flags["type"]; t != "" {
			events, err = store.ByType(ctx, t, limit)
		} else if s := flags["source"]; s != "" {
			events, err = store.BySource(ctx, s, limit)
		} else if c := flags["conversation"]; c != "" {
			events, err = store.ByConversation(ctx, c, limit)
		} else {
			events, err = store.Recent(ctx, limit)
		}
		if err != nil {
			fatal("list events: %v", err)
		}
		if short {
			printShortEvents(events)
		} else {
			printJSON(events)
		}

	case "get":
		if len(args) < 2 {
			fatal("Usage: eg event get <id>")
		}
		e, err := store.Get(ctx, args[1])
		if err != nil {
			fatal("get event: %v", err)
		}
		printJSON(e)

	case "ancestors":
		if len(args) < 2 {
			fatal("Usage: eg event ancestors <id> [--depth=N]")
		}
		flags := parseFlags(args[2:])
		depth := intFlag(flags, "depth", 10)
		events, err := store.Ancestors(ctx, args[1], depth)
		if err != nil {
			fatal("ancestors: %v", err)
		}
		printJSON(events)

	case "descendants":
		if len(args) < 2 {
			fatal("Usage: eg event descendants <id> [--depth=N]")
		}
		flags := parseFlags(args[2:])
		depth := intFlag(flags, "depth", 10)
		events, err := store.Descendants(ctx, args[1], depth)
		if err != nil {
			fatal("descendants: %v", err)
		}
		printJSON(events)

	case "search":
		if len(args) < 2 {
			fatal("Usage: eg event search <query> [--limit=N] [--format=short]")
		}
		flags := parseFlags(args[2:])
		limit := intFlag(flags, "limit", 20)
		events, err := store.Search(ctx, args[1], limit)
		if err != nil {
			fatal("search: %v", err)
		}
		if flags["format"] == "short" {
			printShortEvents(events)
		} else {
			printJSON(events)
		}

	case "types":
		types, err := store.DistinctTypes(ctx)
		if err != nil {
			fatal("types: %v", err)
		}
		printJSON(types)

	case "sources":
		sources, err := store.DistinctSources(ctx)
		if err != nil {
			fatal("sources: %v", err)
		}
		printJSON(sources)

	case "verify":
		err := store.VerifyChain(ctx)
		if err != nil {
			fatal("chain verification failed: %v", err)
		}
		fmt.Println(`{"status":"ok","message":"hash chain verified"}`)

	default:
		fatal("unknown event command: %s", args[0])
	}
}

func handleTask(ctx context.Context, store task.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: eg task <create|list|get|update|complete> [--format=short for list]")
		os.Exit(1)
	}

	switch args[0] {
	case "create":
		flags := parseFlags(args[1:])
		subject := flags["subject"]
		if subject == "" {
			fatal("--subject is required")
		}
		t := &task.Task{
			Subject:     subject,
			Description: flags["description"],
			Source:      flags["source"],
			ParentID:    flags["parent"],
			Priority:    intFlag(flags, "priority", 0),
		}
		if bb := flags["blocked-by"]; bb != "" {
			t.BlockedBy = strings.Split(bb, ",")
		}
		result, err := store.Create(ctx, t)
		if err != nil {
			fatal("create task: %v", err)
		}
		printJSON(result)

	case "list":
		flags := parseFlags(args[1:])
		status := flags["status"]
		limit := intFlag(flags, "limit", 20)
		tasks, err := store.List(ctx, status, limit)
		if err != nil {
			fatal("list tasks: %v", err)
		}
		if flags["format"] == "short" {
			printShortTasks(tasks)
		} else {
			printJSON(tasks)
		}

	case "get":
		if len(args) < 2 {
			fatal("Usage: eg task get <id>")
		}
		t, err := store.Get(ctx, args[1])
		if err != nil {
			fatal("get task: %v", err)
		}
		printJSON(t)

	case "update":
		if len(args) < 2 {
			fatal("Usage: eg task update <id> [--status=...] [--assignee=...] [--description=...]")
		}
		flags := parseFlags(args[2:])
		updates := make(map[string]any)
		if v, ok := flags["status"]; ok && v != "" {
			updates["status"] = v
		}
		if v, ok := flags["assignee"]; ok && v != "" {
			updates["assignee"] = v
		}
		if v, ok := flags["description"]; ok && v != "" {
			updates["description"] = v
		}
		if v, ok := flags["subject"]; ok && v != "" {
			updates["subject"] = v
		}
		if v, ok := flags["priority"]; ok && v != "" {
			p, _ := strconv.Atoi(v)
			updates["priority"] = p
		}
		if len(updates) == 0 {
			fatal("no updates specified")
		}
		t, err := store.Update(ctx, args[1], updates)
		if err != nil {
			fatal("update task: %v", err)
		}
		printJSON(t)

	case "complete":
		if len(args) < 2 {
			fatal("Usage: eg task complete <id>")
		}
		t, err := store.Complete(ctx, args[1])
		if err != nil {
			fatal("complete task: %v", err)
		}
		printJSON(t)

	default:
		fatal("unknown task command: %s", args[0])
	}
}

func handleAuthority(ctx context.Context, store authority.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: eg authority <request|list|check|resolve>")
		os.Exit(1)
	}

	switch args[0] {
	case "request":
		flags := parseFlags(args[1:])
		action := flags["action"]
		if action == "" {
			fatal("--action is required")
		}
		description := flags["description"]
		source := flags["source"]
		if source == "" {
			source = "mind"
		}
		levelStr := flags["level"]
		if levelStr == "" {
			levelStr = "required"
		}
		level := authority.Level(levelStr)
		r, err := store.Create(ctx, action, description, source, level)
		if err != nil {
			fatal("create request: %v", err)
		}
		printJSON(r)

	case "list":
		flags := parseFlags(args[1:])
		if flags["status"] == "pending" {
			reqs, err := store.Pending(ctx)
			if err != nil {
				fatal("list pending: %v", err)
			}
			printJSON(reqs)
		} else {
			limit := intFlag(flags, "limit", 20)
			reqs, err := store.Recent(ctx, limit)
			if err != nil {
				fatal("list recent: %v", err)
			}
			printJSON(reqs)
		}

	case "check":
		if len(args) < 2 {
			fatal("Usage: eg authority check <id>")
		}
		r, err := store.Get(ctx, args[1])
		if err != nil {
			fatal("check request: %v", err)
		}
		printJSON(r)

	case "resolve":
		if len(args) < 2 {
			fatal("Usage: eg authority resolve <id> --approved|--rejected")
		}
		flags := parseFlags(args[2:])
		approved := false
		if _, ok := flags["approved"]; ok {
			approved = true
		} else if _, ok := flags["rejected"]; ok {
			approved = false
		} else {
			fatal("specify --approved or --rejected")
		}
		r, err := store.Resolve(ctx, args[1], approved)
		if err != nil {
			fatal("resolve request: %v", err)
		}
		printJSON(r)

	default:
		fatal("unknown authority command: %s", args[0])
	}
}

func handleStatus(ctx context.Context, events eventgraph.EventStore, tasks task.Store, auth authority.Store) {
	eventCount, _ := events.Count(ctx)
	taskCount, _ := tasks.Count(ctx)
	pendingTasks, _ := tasks.PendingCount(ctx)
	pendingAuth, _ := auth.PendingCount(ctx)

	status := map[string]any{
		"events":            eventCount,
		"tasks":             taskCount,
		"pending_tasks":     pendingTasks,
		"pending_approvals": pendingAuth,
	}
	printJSON(status)
}

func handleActor(ctx context.Context, store actor.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: eg actor <list|register|get>")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		actors, err := store.List(ctx)
		if err != nil {
			fatal("list actors: %v", err)
		}
		printJSON(actors)

	case "register":
		flags := parseFlags(args[1:])
		actorType := flags["type"]
		name := flags["name"]
		if actorType == "" || name == "" {
			fatal("--type and --name are required")
		}
		email := flags["email"]
		a, err := store.Register(ctx, actorType, name, email)
		if err != nil {
			fatal("register actor: %v", err)
		}
		printJSON(a)

	case "get":
		if len(args) < 2 {
			fatal("Usage: eg actor get <id>")
		}
		a, err := store.Get(ctx, args[1])
		if err != nil {
			fatal("get actor: %v", err)
		}
		printJSON(a)

	default:
		fatal("unknown actor command: %s", args[0])
	}
}

func handlePolicy(ctx context.Context, store authority.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: eg policy <list|create|match>")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		policies, err := store.ListPolicies(ctx)
		if err != nil {
			fatal("list policies: %v", err)
		}
		printJSON(policies)

	case "create":
		flags := parseFlags(args[1:])
		action := flags["action"]
		approver := flags["approver"]
		if action == "" || approver == "" {
			fatal("--action and --approver (actor ID) are required")
		}
		levelStr := flags["level"]
		if levelStr == "" {
			levelStr = "required"
		}
		p, err := store.CreatePolicy(ctx, action, approver, authority.Level(levelStr))
		if err != nil {
			fatal("create policy: %v", err)
		}
		printJSON(p)

	case "match":
		if len(args) < 2 {
			fatal("Usage: eg policy match <action>")
		}
		p, err := store.MatchPolicy(ctx, args[1])
		if err != nil {
			fatal("match policy: %v", err)
		}
		printJSON(p)

	default:
		fatal("unknown policy command: %s", args[0])
	}
}

func handleInit(ctx context.Context, events eventgraph.EventStore, tasks task.Store, auth authority.Store, actors actor.Store) {
	if err := events.EnsureTable(ctx); err != nil {
		fatal("ensure events table: %v", err)
	}
	if err := tasks.EnsureTable(ctx); err != nil {
		fatal("ensure tasks table: %v", err)
	}
	if err := auth.EnsureTable(ctx); err != nil {
		fatal("ensure authority table: %v", err)
	}
	if err := actors.EnsureTable(ctx); err != nil {
		fatal("ensure actors table: %v", err)
	}
	fmt.Println(`{"status":"ok","message":"all tables initialized"}`)
}

// parseFlags parses --key=value and --flag style args into a map.
func parseFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		arg = strings.TrimPrefix(arg, "--")
		if idx := strings.Index(arg, "="); idx >= 0 {
			flags[arg[:idx]] = arg[idx+1:]
		} else {
			flags[arg] = ""
		}
	}
	return flags
}

func intFlag(flags map[string]string, key string, defaultVal int) int {
	if v, ok := flags[key]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal("encode JSON: %v", err)
	}
}

func truncStr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func printShortEvents(events []eventgraph.Event) {
	for _, e := range events {
		ts := ""
		if len(e.Timestamp.String()) > 19 {
			ts = e.Timestamp.Format("15:04:05")
		}
		content := ""
		if b, err := json.Marshal(e.Content); err == nil {
			content = string(b)
		}
		fmt.Printf("%-8s  %-35s  %s\n", ts, truncStr(e.Type, 35), truncStr(content, 80))
	}
}

func printShortTasks(tasks []task.Task) {
	for _, t := range tasks {
		id := truncStr(t.ID, 8)
		fmt.Printf("%-8s  %-12s  %s\n", id, t.Status, truncStr(t.Subject, 60))
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "eg: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: eg <command>

Commands:
  event      Event operations (create, list, get, ancestors, descendants, search, types, sources, verify)
  task       Task operations (create, list, get, update, complete)
  authority  Authority operations (request, list, check, resolve)
  actor      Actor operations (list, register, get)
  policy     Policy operations (list, create, match)
  status     Show system summary
  init       Initialize database tables`)
}
