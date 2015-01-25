package views

import (
	"fmt"
	"github.com/gophergala/humble"
	"github.com/gophergala/humble/view"
	"honnef.co/go/js/dom"
	"strings"
)

type Footer struct {
	humble.Identifier
	TodoViews *[]*Todo
}

func (f *Footer) RenderHTML() string {
	return fmt.Sprintf(`
			<span id="todo-count">
				<strong>%d</strong> items left
			</span>
			<ul id="filters">
				<li>
					<a href="#/">All</a>
				</li>
				<li>
					<a href="#/active">Active</a>
				</li>
				<li>
					<a href="#/completed">Completed</a>
				</li>
			</ul>`, f.countRemaining())
}

func (f *Footer) OuterTag() string {
	return "div"
}

func (f *Footer) OnLoad() error {
	if err := f.setSelected(); err != nil {
		return err
	}
	return nil
}

func (f *Footer) countRemaining() int {
	count := 0
	if f.TodoViews == nil {
		return 0
	}
	for _, todoView := range *f.TodoViews {
		if !todoView.Model.IsCompleted {
			count++
		}
	}
	return count
}

func (f *Footer) setSelected() error {
	hash := dom.GetWindow().Location().Hash
	links, err := view.QuerySelectorAll(f, "#filters li a")
	if err != nil {
		return err
	}
	for _, linkEl := range links {
		if linkEl.GetAttribute("href") == hash {
			linkEl.SetAttribute("class", linkEl.GetAttribute("class")+" selected")
		} else {
			oldClass := linkEl.GetAttribute("class")
			newClass := strings.Replace(oldClass, "selected", "", 1)
			linkEl.SetAttribute("class", newClass)
		}
	}
	return nil
}
