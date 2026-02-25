package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var (
	apiBase = "/"
	theme   *material.Theme
)

// Pages
const (
	pageDashboard = iota
	pageEvents
	pageTasks
	pageAuthority
	pageChat
)

type UI struct {
	currentPage int

	// Nav buttons
	navDashboard widget.Clickable
	navEvents    widget.Clickable
	navTasks     widget.Clickable
	navAuthority widget.Clickable
	navChat      widget.Clickable

	// Dashboard
	status Status

	// Events
	eventList  widget.List
	events     []Event
	refreshBtn widget.Clickable

	// Tasks
	taskList      widget.List
	tasks         []Task
	newTaskEditor widget.Editor
	createTaskBtn widget.Clickable

	// Authority
	authList     widget.List
	authRequests []AuthRequest
	approveBtn   []widget.Clickable
	rejectBtn    []widget.Clickable

	// Chat
	chatList      widget.List
	chatMessages  []ChatMessage
	chatEditor    widget.Editor
	chatSendBtn   widget.Clickable
}

type Status struct {
	Events          int `json:"events"`
	Tasks           int `json:"tasks"`
	PendingTasks    int `json:"pending_tasks"`
	PendingApprovals int `json:"pending_approvals"`
}

type Event struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Source    string         `json:"source"`
	Content   map[string]any `json:"content"`
}

type Task struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	Assignee    string `json:"assignee"`
}

type AuthRequest struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Level       string `json:"level"`
	Status      string `json:"status"`
	Source      string `json:"source"`
}

type ChatMessage struct {
	From    string
	Content string
	Time    time.Time
}

func main() {
	if base := os.Getenv("API_BASE"); base != "" {
		apiBase = base
	}

	theme = material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	theme.Palette.Bg = color.NRGBA{R: 0x12, G: 0x12, B: 0x12, A: 0xFF}
	theme.Palette.Fg = color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF}
	theme.Palette.ContrastBg = color.NRGBA{R: 0x30, G: 0x60, B: 0xA0, A: 0xFF}
	theme.Palette.ContrastFg = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

	ui := &UI{}
	ui.eventList.Axis = layout.Vertical
	ui.taskList.Axis = layout.Vertical
	ui.authList.Axis = layout.Vertical
	ui.chatList.Axis = layout.Vertical
	ui.newTaskEditor.SingleLine = true
	ui.chatEditor.SingleLine = true

	go ui.pollData()

	go func() {
		w := new(app.Window)
		w.Option(app.Title("mind-zero-five"))
		w.Option(app.Size(unit.Dp(1200), unit.Dp(800)))
		if err := ui.run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func (ui *UI) run(w *app.Window) error {
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			ui.handleClicks(gtx)
			ui.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (ui *UI) handleClicks(gtx layout.Context) {
	if ui.navDashboard.Clicked(gtx) {
		ui.currentPage = pageDashboard
	}
	if ui.navEvents.Clicked(gtx) {
		ui.currentPage = pageEvents
	}
	if ui.navTasks.Clicked(gtx) {
		ui.currentPage = pageTasks
	}
	if ui.navAuthority.Clicked(gtx) {
		ui.currentPage = pageAuthority
	}
	if ui.navChat.Clicked(gtx) {
		ui.currentPage = pageChat
	}
	if ui.refreshBtn.Clicked(gtx) {
		go ui.fetchAll()
	}
	if ui.createTaskBtn.Clicked(gtx) {
		subject := ui.newTaskEditor.Text()
		if subject != "" {
			go ui.createTask(subject)
			ui.newTaskEditor.SetText("")
		}
	}
	if ui.chatSendBtn.Clicked(gtx) {
		msg := ui.chatEditor.Text()
		if msg != "" {
			go ui.sendChat(msg)
			ui.chatEditor.SetText("")
		}
	}
	for i := range ui.approveBtn {
		if i < len(ui.authRequests) && ui.approveBtn[i].Clicked(gtx) {
			go ui.resolveAuth(ui.authRequests[i].ID, true)
		}
	}
	for i := range ui.rejectBtn {
		if i < len(ui.authRequests) && ui.rejectBtn[i].Clicked(gtx) {
			go ui.resolveAuth(ui.authRequests[i].ID, false)
		}
	}
}

func (ui *UI) layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.layoutNav(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				switch ui.currentPage {
				case pageEvents:
					return ui.layoutEvents(gtx)
				case pageTasks:
					return ui.layoutTasks(gtx)
				case pageAuthority:
					return ui.layoutAuthority(gtx)
				case pageChat:
					return ui.layoutChat(gtx)
				default:
					return ui.layoutDashboard(gtx)
				}
			})
		}),
	)
}

func (ui *UI) layoutNav(gtx layout.Context) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(180))
			gtx.Constraints.Max.X = gtx.Dp(unit.Dp(180))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.H6(theme, "mz5")
						label.Color = theme.Palette.ContrastFg
						return label.Layout(gtx)
					})
				}),
				layout.Rigid(navBtn(theme, &ui.navDashboard, "Dashboard", ui.currentPage == pageDashboard)),
				layout.Rigid(navBtn(theme, &ui.navEvents, "Events", ui.currentPage == pageEvents)),
				layout.Rigid(navBtn(theme, &ui.navTasks, "Tasks", ui.currentPage == pageTasks)),
				layout.Rigid(navBtn(theme, &ui.navAuthority, "Authority", ui.currentPage == pageAuthority)),
				layout.Rigid(navBtn(theme, &ui.navChat, "Chat", ui.currentPage == pageChat)),
			)
		},
	)
}

func navBtn(th *material.Theme, btn *widget.Clickable, label string, active bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			b := material.Button(th, btn, label)
			if active {
				b.Background = th.Palette.ContrastBg
			} else {
				b.Background = color.NRGBA{A: 0}
			}
			b.Color = th.Palette.Fg
			return b.Layout(gtx)
		})
	}
}

func (ui *UI) layoutDashboard(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceEnd}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H5(theme, "Dashboard").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(theme, fmt.Sprintf("Events: %d", ui.status.Events)).Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(theme, fmt.Sprintf("Tasks: %d", ui.status.Tasks)).Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(theme, fmt.Sprintf("Pending Tasks: %d", ui.status.PendingTasks)).Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(theme, fmt.Sprintf("Pending Approvals: %d", ui.status.PendingApprovals)).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(theme, &ui.refreshBtn, "Refresh").Layout(gtx)
		}),
	)
}

func (ui *UI) layoutEvents(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H5(theme, "Events").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(theme, &ui.eventList).Layout(gtx, len(ui.events), func(gtx layout.Context, i int) layout.Dimensions {
				e := ui.events[i]
				return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Body2(theme, fmt.Sprintf("[%s] %s ← %s", e.Timestamp.Format("15:04:05"), e.Type, e.Source))
							label.Font.Weight = font.Bold
							return label.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Caption(theme, e.ID[:8]+"...")
							label.Color = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF}
							return label.Layout(gtx)
						}),
					)
				})
			})
		}),
	)
}

func (ui *UI) layoutTasks(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H5(theme, "Tasks").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					ed := material.Editor(theme, &ui.newTaskEditor, "New task subject...")
					return ed.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Button(theme, &ui.createTaskBtn, "Create").Layout(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(theme, &ui.taskList).Layout(gtx, len(ui.tasks), func(gtx layout.Context, i int) layout.Dimensions {
				t := ui.tasks[i]
				statusColor := color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF}
				switch t.Status {
				case "pending":
					statusColor = color.NRGBA{R: 0xFF, G: 0xA0, B: 0x00, A: 0xFF}
				case "in_progress":
					statusColor = color.NRGBA{R: 0x00, G: 0xA0, B: 0xFF, A: 0xFF}
				case "completed":
					statusColor = color.NRGBA{R: 0x00, G: 0xC0, B: 0x00, A: 0xFF}
				case "blocked":
					statusColor = color.NRGBA{R: 0xFF, G: 0x40, B: 0x40, A: 0xFF}
				}
				return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Body2(theme, t.Subject)
							label.Font.Weight = font.Bold
							return label.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Caption(theme, fmt.Sprintf("[%s] %s", t.Status, t.ID[:8]+"..."))
							label.Color = statusColor
							return label.Layout(gtx)
						}),
					)
				})
			})
		}),
	)
}

func (ui *UI) layoutAuthority(gtx layout.Context) layout.Dimensions {
	// Ensure button slices match data
	for len(ui.approveBtn) < len(ui.authRequests) {
		ui.approveBtn = append(ui.approveBtn, widget.Clickable{})
		ui.rejectBtn = append(ui.rejectBtn, widget.Clickable{})
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H5(theme, "Authority").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(theme, &ui.authList).Layout(gtx, len(ui.authRequests), func(gtx layout.Context, i int) layout.Dimensions {
				r := ui.authRequests[i]
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Body2(theme, fmt.Sprintf("[%s] %s", r.Level, r.Action))
							label.Font.Weight = font.Bold
							return label.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Caption(theme, r.Description).Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if r.Status != "pending" {
								label := material.Caption(theme, "Status: "+r.Status)
								return label.Layout(gtx)
							}
							return layout.Flex{}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return material.Button(theme, &ui.approveBtn[i], "Approve").Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(theme, &ui.rejectBtn[i], "Reject")
									btn.Background = color.NRGBA{R: 0xC0, G: 0x30, B: 0x30, A: 0xFF}
									return btn.Layout(gtx)
								}),
							)
						}),
					)
				})
			})
		}),
	)
}

func (ui *UI) layoutChat(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H5(theme, "Chat").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(theme, &ui.chatList).Layout(gtx, len(ui.chatMessages), func(gtx layout.Context, i int) layout.Dimensions {
				msg := ui.chatMessages[i]
				return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					label := material.Body2(theme, fmt.Sprintf("[%s] %s: %s", msg.Time.Format("15:04"), msg.From, msg.Content))
					return label.Layout(gtx)
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					ed := material.Editor(theme, &ui.chatEditor, "Send a message...")
					return ed.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Button(theme, &ui.chatSendBtn, "Send").Layout(gtx)
				}),
			)
		}),
	)
}

// Data fetching

func (ui *UI) pollData() {
	ui.fetchAll()
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		ui.fetchAll()
	}
}

func (ui *UI) fetchAll() {
	ui.fetchStatus()
	ui.fetchEvents()
	ui.fetchTasks()
	ui.fetchAuthority()
}

func (ui *UI) fetchStatus() {
	var s Status
	if err := httpGetJSON(apiBase+"api/status", &s); err != nil {
		log.Printf("fetch status: %v", err)
		return
	}
	ui.status = s
}

func (ui *UI) fetchEvents() {
	var events []Event
	if err := httpGetJSON(apiBase+"api/events?limit=100", &events); err != nil {
		log.Printf("fetch events: %v", err)
		return
	}
	ui.events = events
}

func (ui *UI) fetchTasks() {
	var tasks []Task
	if err := httpGetJSON(apiBase+"api/tasks?limit=100", &tasks); err != nil {
		log.Printf("fetch tasks: %v", err)
		return
	}
	ui.tasks = tasks
}

func (ui *UI) fetchAuthority() {
	var reqs []AuthRequest
	if err := httpGetJSON(apiBase+"api/authority?limit=50", &reqs); err != nil {
		log.Printf("fetch authority: %v", err)
		return
	}
	ui.authRequests = reqs
}

func (ui *UI) createTask(subject string) {
	body := fmt.Sprintf(`{"subject":%q,"source":"ui"}`, subject)
	resp, err := http.Post(apiBase+"api/tasks", "application/json", strings.NewReader(body))
	if err != nil {
		log.Printf("create task: %v", err)
		return
	}
	resp.Body.Close()
	ui.fetchTasks()
}

func (ui *UI) resolveAuth(id string, approved bool) {
	url := fmt.Sprintf("%sapi/authority/%s/resolve?approved=%v", apiBase, id, approved)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		log.Printf("resolve auth: %v", err)
		return
	}
	resp.Body.Close()
	ui.fetchAuthority()
}

func (ui *UI) sendChat(msg string) {
	ui.chatMessages = append(ui.chatMessages, ChatMessage{
		From:    "human",
		Content: msg,
		Time:    time.Now(),
	})
	// Create a signal.human event
	body := fmt.Sprintf(`{"type":"signal.human","source":"ui","content":{"message":%q}}`, msg)
	resp, err := http.Post(apiBase+"api/events", "application/json", strings.NewReader(body))
	if err != nil {
		log.Printf("send chat: %v", err)
		return
	}
	resp.Body.Close()
}

func httpGetJSON(url string, v any) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
