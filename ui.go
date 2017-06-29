package main

import (
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/maxdec/termui"
)

// Refresh represents a function that returns a fresh list of instances
type Refresh func() []*ec2.Instance

// UI wraps
type UI struct {
	api                *API
	searchBox          *termui.Par
	instancesTable     *termui.Table
	helpBox            *termui.Par
	instances          []*ec2.Instance
	displayedInstances []*ec2.Instance
	startRow           int
	selectedRow        int
	err                error
}

const (
	// UP moves the cursor up...
	UP = -1
	// DOWN moves the cursor down...
	DOWN = 1
	// TOP moves the cursor to the top
	TOP = -2
	// BOTTOM moves the cursor to the bottom
	BOTTOM = 2
)

// StartUI launches the UI (initialization)
func StartUI(api *API) {
	err := termui.Init()
	if err != nil {
		panic(err)
	}
	defer termui.Close()

	ui := &UI{
		api:            api,
		searchBox:      newSearchBox(),
		instancesTable: newInstancesTable(),
		helpBox:        newHelpTable(),
		selectedRow:    -1,
	}

	go func() {
		for instances := range api.instancesChan {
			termui.SendCustomEvt("/usr/instances", instances)
		}
	}()

	go func() {
		for errors := range api.errChan {
			termui.SendCustomEvt("/usr/errors", errors)
		}
	}()

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, ui.searchBox),
		),
		termui.NewRow(
			termui.NewCol(12, 0, ui.instancesTable),
		),
		termui.NewRow(
			termui.NewCol(12, 0, ui.helpBox),
		),
	)
	termui.Body.Align()
	termui.Render(termui.Body)

	ui.SetEvents()
	ui.triggerInstancesUpdate()

	termui.Loop()
}

// SetEvents registers custom events (keyboard shortuts, key presses, window resizes, ...)
func (ui *UI) SetEvents() {
	termui.Handle("/sys/kbd/C-c", func(termui.Event) {
		termui.StopLoop()
	})

	termui.Handle("/sys/kbd/C-r", func(termui.Event) {
		ui.triggerInstancesUpdate()
	})

	termui.Handle("/usr/instances", func(e termui.Event) {
		ui.instances = e.Data.([]*ec2.Instance)
		ui.filterInstancesToDisplay()
		ui.refreshInstancesTable()
		termui.Render(termui.Body)
	})

	termui.Handle("/usr/errors", func(e termui.Event) {
		if e.Data != nil {
			ui.err = e.Data.(error)
			ui.refreshErrorMsg(ui.err)
			termui.Render(termui.Body)
		}
	})

	termui.Handle("/sys/kbd/<enter>", func(termui.Event) {
		if ui.selectedRow > 0 && (ui.selectedRow+ui.startRow-1) < len(ui.displayedInstances) {
			exec.Command("open", "ssh://"+*ui.displayedInstances[ui.selectedRow+ui.startRow-1].PublicDnsName).Start()
		}
	})

	termui.Handle("/sys/kbd/<up>", func(termui.Event) {
		oldStartRow := ui.startRow
		ui.scroll(UP)
		newStartRow := ui.startRow
		if oldStartRow != newStartRow {
			ui.refreshInstancesTable()
		}
		termui.Render(termui.Body)
	})

	termui.Handle("/sys/kbd/<down>", func(termui.Event) {
		oldStartRow := ui.startRow
		ui.scroll(DOWN)
		newStartRow := ui.startRow
		if oldStartRow != newStartRow {
			ui.refreshInstancesTable()
		}
		termui.Render(termui.Body)
	})

	termui.Handle("/sys/kbd/<home>", func(termui.Event) {
		oldStartRow := ui.startRow
		ui.scroll(TOP)
		newStartRow := ui.startRow
		if oldStartRow != newStartRow {
			ui.refreshInstancesTable()
		}
		termui.Render(termui.Body)
	})

	termui.Handle("/sys/kbd/<previous>", func(termui.Event) {
		oldStartRow := ui.startRow
		ui.scroll(TOP)
		newStartRow := ui.startRow
		if oldStartRow != newStartRow {
			ui.refreshInstancesTable()
		}
		termui.Render(termui.Body)
	})

	termui.Handle("/sys/kbd/<end>", func(termui.Event) {
		oldStartRow := ui.startRow
		ui.scroll(BOTTOM)
		newStartRow := ui.startRow
		if oldStartRow != newStartRow {
			ui.refreshInstancesTable()
		}
		termui.Render(termui.Body)
	})

	termui.Handle("/sys/kbd/<next>", func(termui.Event) {
		oldStartRow := ui.startRow
		ui.scroll(BOTTOM)
		newStartRow := ui.startRow
		if oldStartRow != newStartRow {
			ui.refreshInstancesTable()
		}
		termui.Render(termui.Body)
	})

	termui.Handle("/sys/kbd/<left>", func(termui.Event) { /*ignore*/ })
	termui.Handle("/sys/kbd/<right>", func(termui.Event) { /*ignore*/ })

	termui.Handle("/sys/kbd/C-8", func(termui.Event) {
		if len(ui.searchBox.Text) > 0 {
			ui.searchBox.Text = ui.searchBox.Text[:len(ui.searchBox.Text)-1]
			ui.filterInstancesToDisplay()
			ui.refreshInstancesTable()
			termui.Render(termui.Body)
		}
	})

	termui.Handle("/sys/kbd/<backspace>", func(termui.Event) {
		if len(ui.searchBox.Text) > 0 {
			ui.searchBox.Text = ui.searchBox.Text[:len(ui.searchBox.Text)-1]
			ui.filterInstancesToDisplay()
			ui.refreshInstancesTable()
			termui.Render(termui.Body)
		}
	})

	termui.Handle("/sys/kbd", func(e termui.Event) {
		if len(ui.searchBox.Text) < 60 {
			key := strings.Split(e.Path, "/")[3]
			ui.searchBox.Text = ui.searchBox.Text + (key)
			ui.filterInstancesToDisplay()
			ui.refreshInstancesTable()
			termui.Render(termui.Body)
		}
	})

	termui.Handle("/sys/wnd/resize", func(termui.Event) {
		termui.Body.Width = termui.TermWidth()
		termui.Body.Align()
		termui.Clear()
		termui.Render(termui.Body)
	})
}

func newSearchBox() *termui.Par {
	searchBox := termui.NewPar("")
	searchBox.Height = 3
	searchBox.Width = 60
	searchBox.PaddingLeft = 3
	searchBox.BorderFg = termui.ColorCyan
	searchBox.BorderLabel = " Search "
	searchBox.BorderLabelFg = termui.ColorCyan

	return searchBox
}

func newInstancesTable() *termui.Table {
	instancesTable := termui.NewTable()
	instancesTable.BorderLabel = " Instances "
	instancesTable.BorderLabelFg = termui.ColorWhite
	instancesTable.FgColor = termui.ColorWhite
	instancesTable.BgColor = termui.ColorDefault
	instancesTable.Y = 0
	instancesTable.X = 0
	// instancesTable.Width = 62
	instancesTable.Separator = false
	instancesTable.Height = termui.TermHeight() - 7
	// instancesTable.Rows = [][]string{headers()}

	return instancesTable
}

func newHelpTable() *termui.Par {
	helpTable := termui.NewPar("[CTRL+c] quit    [CTRL+r] refresh    [ENTER] ssh    [⬆ ⬇ ] scroll")
	helpTable.Height = 3
	helpTable.Width = 62
	helpTable.PaddingLeft = 3
	helpTable.BorderFg = termui.ColorCyan
	helpTable.BorderLabel = " Help "
	helpTable.BorderLabelFg = termui.ColorCyan

	return helpTable
}

func (ui *UI) triggerInstancesUpdate() {
	go ui.api.List("")
	// go ui.api.ExampleList()
}

func (ui *UI) refreshInstancesTable() {
	instances := ui.getInstancesToDisplay()
	rows := [][]string{ui.headers()}
	for _, instance := range instances {
		if instance != nil {
			rows = append(rows, ToRow(instance))
		}
	}
	ui.instancesTable.SetRows(rows)
}

func (ui *UI) refreshErrorMsg(err error) {
	rows := [][]string{[]string{err.Error()}}
	ui.instancesTable.SetRows(rows)
}

func (ui *UI) getInstancesToDisplay() []*ec2.Instance {
	height := ui.instancesTable.Height - 3
	if len(ui.displayedInstances) < height {
		return ui.displayedInstances
	}
	start := ui.startRow
	end := ui.startRow + height
	return ui.displayedInstances[start:end]
}

func (ui *UI) filterInstancesToDisplay() {
	ui.displayedInstances = Filter(ui.instances, ui.searchBox.Text)
}

func (ui *UI) scroll(dir int) {
	if ui.selectedRow < 0 {
		ui.selectedRow = 0
	}

	// Prevents crashing by scrolling during initialization
	if len(ui.instancesTable.BgColors) == 0 || len(ui.instancesTable.FgColors) == 0 {
		return
	}

	ui.instancesTable.BgColors[ui.selectedRow] = termui.ColorDefault
	ui.instancesTable.FgColors[ui.selectedRow] = termui.ColorDefault
	height := ui.instancesTable.Height - 3

	if dir == UP {
		ui.selectedRow--
		if ui.selectedRow < 1 {
			ui.selectedRow = 1
			ui.startRow = max(ui.startRow-1, 0)
		} else if ui.selectedRow > len(ui.displayedInstances) {
			ui.selectedRow = len(ui.displayedInstances)
			ui.startRow = 0
		}
	} else if dir == TOP {
		ui.selectedRow = 1
		ui.startRow = 0
	} else if dir == DOWN {
		ui.selectedRow++
		if ui.selectedRow > len(ui.displayedInstances) {
			ui.selectedRow = len(ui.displayedInstances)
			ui.startRow = 0
		} else if ui.selectedRow > height {
			ui.selectedRow = height
			if len(ui.instances) < height {
				ui.startRow = 0
			} else {
				ui.startRow = min(ui.startRow+1, len(ui.instances)-height)
			}
		}
	} else if dir == BOTTOM {
		ui.selectedRow = min(height, len(ui.displayedInstances))
		if len(ui.instances) < height {
			ui.startRow = 0
		} else {
			ui.startRow = max(ui.startRow+1, len(ui.instances)-height)
		}
	}

	// ui.selectedRow = between(ui.selectedRow+dir, 1, len(ui.instancesTable.Rows)-1)
	ui.instancesTable.BgColors[ui.selectedRow] = termui.ColorWhite
	ui.instancesTable.FgColors[ui.selectedRow] = termui.ColorBlack
}

func (ui *UI) headers() []string {
	headers := []string{}
	for _, h := range []string{"InstanceId", "Name", "PublicDNS", "PrivateDns", "Type", "Zone", "LaunchedAt", "State"} {
		headers = append(headers, "["+h+"](fg-bold)")
	}
	return headers
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
func between(x, min, max int) int {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}
