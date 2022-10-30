package main

// A simple program demonstrating the text area component from the Bubbles
// component library.

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nats-io/nats.go"
)

type errMsg error

type model struct {
	viewport    viewport.Model
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error

	messages []string
	ec       *nats.EncodedConn

	userList tea.Model
	focused  string
}

func initialModel(ec *nats.EncodedConn) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	// ta.SetWidth(30)
	ta.SetHeight(1)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(100, 20)
	vp.SetContent(`Welcome to the chat room!
Type a message and press Enter to send.`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		textarea: ta,
		viewport: vp,
		messages: []string{},
		ec:       ec,
		userList: newUserModel(ec),

		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
	}
}

// inital I/O
func (m model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	// case tea.WindowSizeMsg:
	// 	// TODO: Add rest of models
	// 	return m, nil

	case *chat: // a published chat message
		m.messages = append(m.messages, m.senderStyle.Render(fmt.Sprintf("%s: %s", msg.Name, msg.Msg)))
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.textarea.Reset()
		m.viewport.GotoBottom()

	case *user: // user logs in or out
		var message string
		if msg.LoggedIn {
			message = fmt.Sprintf("%s logged in", msg.Name)
		} else {
			message = fmt.Sprintf("%s logged out", msg.Name)
		}

		m.messages = append(m.messages, message)
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.textarea.Reset()
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		if m.textarea.Focused() {
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// unsubscribe
			// broadcast leaving or remove from list of online users
			m.ec.Publish("user", user{Name: "meow", LoggedIn: false})
			m.ec.Drain()
			return m, tea.Quit
		case tea.KeyEnter:
			// publish message for others to see
			m.ec.Publish("chat", chat{Name: "kason", Msg: m.textarea.Value()})
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "left":
				m.focused = "chat" // move to chat window
			case "right":
				m.focused = "user" // move to user window
			}
		}
	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

// describes the UI of the entire application
func (m model) View() string {
	columnStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.NormalBorder())

	focusedStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("62"))

		// TODO: Style both the chat section and users section

	var cols []string
	switch m.focused {
	case "chat":
		cols = []string{
			columnStyle.Render(
				fmt.Sprintf("%s\n\n%s",
					focusedStyle.Render(m.viewport.View()),
					focusedStyle.Render(m.textarea.View()),
				),
			),
			columnStyle.Render(m.userList.View()),
		}
	case "user":
		cols = []string{
			columnStyle.Render(
				fmt.Sprintf("%s\n\n%s",
					m.viewport.View(),
					m.textarea.View(),
				),
			),
			focusedStyle.Render(columnStyle.Render(m.userList.View())),
		}
	default:
		cols = []string{
			columnStyle.Render(
				fmt.Sprintf("%s\n\n%s",
					focusedStyle.Render(m.viewport.View()),
					m.textarea.View(),
				),
			),
			columnStyle.Render(m.userList.View()),
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, cols...)
}

type userModel struct {
	ec       *nats.EncodedConn
	userList list.Model
}

func newUserModel(ec *nats.EncodedConn) tea.Model {
	items := []list.Item{
		user{Name: "Kason", LoggedIn: true},
		user{Name: "Bob", LoggedIn: true},
		user{Name: "Ted", LoggedIn: false},
	}

	l := list.New(items, itemDelegate{}, 10, 10)
	l.Title = "Users"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return userModel{
		ec:       ec,
		userList: l,
	}
}

func (u userModel) Init() tea.Cmd {
	// TODO: Get all users?
	return nil
}

func (u userModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// TODO: Add rest of models
		u.userList.SetWidth(msg.Width)
		return u, nil

	case *user: // user logs in or out
		// set green bubble?

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			// unsubscribe
			// broadcast leaving or remove from list of online users
			u.ec.Publish("user", user{Name: "meow", LoggedIn: false})
			u.ec.Drain()
			return u, tea.Quit
		}
	}

	return u, nil
}

// describes the UI of the entire application
func (u userModel) View() string {
	return u.userList.View()
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	u, ok := listItem.(user)
	if !ok {
		return
	}

	str := fmt.Sprintf(" %s", u.Name)

	// fn := itemStyle.Render
	// if index == m.Index() {
	// 	fn = func(s string) string {
	// 		return selectedItemStyle.Render("> " + s)
	// 	}
	// }

	fmt.Fprint(w, str)
}
