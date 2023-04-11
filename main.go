package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sizeResult struct {
	Input string
	Size  int
}

type model struct {
	inputs    []string
	results   []sizeResult
	renderErr error
	mu        sync.Mutex
}

func main() {
	inputs := parseFlags()

	m := model{inputs: inputs}

	p := tea.NewProgram(&m)
	if err := p.Start(); err != nil {
		fmt.Println("Failed to start the program:", err)
		os.Exit(1)
	}
}

func (m *model) Init() tea.Cmd {
	return m.fetchSizes()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string {
	if m.renderErr != nil {
		return m.renderErr.Error()
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF00"))
	rowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#DDD"))

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		headerStyle.Width(40).Render("Input"),
		headerStyle.Width(15).Render("Size (bytes)"),
	)

	var rows []string
	for _, r := range m.results {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			rowStyle.Width(40).Render(r.Input),
			rowStyle.Width(15).Render(fmt.Sprint(r.Size)),
		))
	}

	rowString := strings.Join(rows, "\n")

	tableString := lipgloss.JoinVertical(lipgloss.Top, header, rowString)

	table := lipgloss.NewStyle().
		Width(60).
		Padding(3).
		Border(lipgloss.RoundedBorder()).
		Render(tableString)

	return table
}

func (m *model) fetchSizes() tea.Cmd {
	return func() tea.Msg {
		for _, input := range m.inputs {
			var size int
			var err error

			if isURL(input) {
				size, err = getSizeFromURL(input)
			} else {
				size = len(input)
			}

			if err != nil {
				m.mu.Lock()
				m.renderErr = err
				m.mu.Unlock()
				return nil
			}

			m.mu.Lock()
			m.results = append(m.results, sizeResult{Input: input, Size: size})
			m.mu.Unlock()
		}
		// Sort the results initially
		sort.Slice(m.results, func(i, j int) bool {
			return m.results[i].Size < m.results[j].Size
		})

		return nil
	}
}

func parseFlags() []string {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		fmt.Println("Please provide at least one string or URL.")
		os.Exit(1)
	}

	return args
}

func isURL(input string) bool {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return true
	}
	return false
}

func getSizeFromURL(url string) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, errors.New("Error fetching URL: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.New("Error reading response body: " + err.Error())
	}
	return len(body), nil
}
