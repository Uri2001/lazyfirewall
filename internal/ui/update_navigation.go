//go:build linux
// +build linux

package ui

import (
	"strings"
)

func (m *Model) clampSelections() {
	current := m.currentData()
	if current == nil {
		return
	}
	if m.serviceIndex >= len(current.Services) {
		m.serviceIndex = 0
	}
	if m.portIndex >= len(current.Ports) {
		m.portIndex = 0
	}
	if m.richIndex >= len(current.RichRules) {
		m.richIndex = 0
	}
	items := m.networkItems()
	if len(items) == 0 {
		m.networkIndex = 0
	} else if m.networkIndex >= len(items) {
		m.networkIndex = 0
	}
	if len(m.ipsets) == 0 {
		m.ipsetIndex = 0
	} else if m.ipsetIndex >= len(m.ipsets) {
		m.ipsetIndex = 0
	}
}

func (m *Model) moveMainSelection(delta int) {
	if m.searchQuery != "" {
		m.moveMatchSelection(delta > 0)
		return
	}
	if m.tab == tabIPSets {
		if len(m.ipsets) == 0 {
			return
		}
		next := m.ipsetIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(m.ipsets) {
			next = len(m.ipsets) - 1
		}
		m.ipsetIndex = next
		return
	}
	current := m.currentData()
	if current == nil {
		return
	}
	switch m.tab {
	case tabServices:
		if len(current.Services) == 0 {
			return
		}
		next := m.serviceIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(current.Services) {
			next = len(current.Services) - 1
		}
		m.serviceIndex = next
	case tabPorts:
		if len(current.Ports) == 0 {
			return
		}
		next := m.portIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(current.Ports) {
			next = len(current.Ports) - 1
		}
		m.portIndex = next
	case tabRich:
		if len(current.RichRules) == 0 {
			return
		}
		next := m.richIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(current.RichRules) {
			next = len(current.RichRules) - 1
		}
		m.richIndex = next
	case tabNetwork:
		items := m.networkItems()
		if len(items) == 0 {
			return
		}
		next := m.networkIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(items) {
			next = len(items) - 1
		}
		m.networkIndex = next
		return
	case tabInfo:
		return
	}
}

func (m *Model) moveMatchSelection(forward bool) {
	matches := m.currentMatchIndices()
	if len(matches) == 0 {
		return
	}
	current := m.currentIndex()
	pos := -1
	for i, idx := range matches {
		if idx == current {
			pos = i
			break
		}
	}
	if pos == -1 {
		m.setCurrentIndex(matches[0])
		return
	}
	if forward {
		pos++
		if pos >= len(matches) {
			pos = 0
		}
	} else {
		pos--
		if pos < 0 {
			pos = len(matches) - 1
		}
	}
	m.setCurrentIndex(matches[pos])
}

func (m *Model) currentIndex() int {
	if m.tab == tabPorts {
		return m.portIndex
	}
	if m.tab == tabRich {
		return m.richIndex
	}
	if m.tab == tabNetwork {
		return m.networkIndex
	}
	if m.tab == tabIPSets {
		return m.ipsetIndex
	}
	if m.tab == tabInfo {
		return 0
	}
	return m.serviceIndex
}

func (m *Model) setCurrentIndex(index int) {
	if m.tab == tabPorts {
		m.portIndex = index
		return
	}
	if m.tab == tabRich {
		m.richIndex = index
		return
	}
	if m.tab == tabNetwork {
		m.networkIndex = index
		return
	}
	if m.tab == tabIPSets {
		m.ipsetIndex = index
		return
	}
	if m.tab == tabInfo {
		return
	}
	m.serviceIndex = index
}

func (m *Model) currentItems() []string {
	if m.tab == tabIPSets {
		return m.ipsets
	}
	current := m.currentData()
	if current == nil {
		return nil
	}
	if m.tab == tabPorts {
		items := make([]string, 0, len(current.Ports))
		for _, p := range current.Ports {
			items = append(items, p.Port+"/"+p.Protocol)
		}
		return items
	}
	if m.tab == tabRich {
		return current.RichRules
	}
	if m.tab == tabNetwork {
		items := m.networkItems()
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, item.value)
		}
		return out
	}
	if m.tab == tabInfo {
		return nil
	}
	return current.Services
}

func (m *Model) currentMatchIndices() []int {
	return matchIndices(m.currentItems(), m.searchQuery)
}

func (m *Model) applySearchSelection() {
	if m.searchQuery == "" {
		return
	}
	matches := m.currentMatchIndices()
	if len(matches) == 0 {
		return
	}
	current := m.currentIndex()
	for _, idx := range matches {
		if idx == current {
			return
		}
	}
	m.setCurrentIndex(matches[0])
}

func matchIndices(items []string, query string) []int {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	indices := make([]int, 0, len(items))
	for i, item := range items {
		if strings.Contains(strings.ToLower(item), query) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m *Model) nextTab() {
	switch m.tab {
	case tabServices:
		m.tab = tabPorts
	case tabPorts:
		m.tab = tabRich
	case tabRich:
		m.tab = tabNetwork
	case tabNetwork:
		m.tab = tabIPSets
	case tabIPSets:
		m.tab = tabInfo
	case tabInfo:
		m.tab = tabServices
	}
}

func (m *Model) prevTab() {
	switch m.tab {
	case tabServices:
		m.tab = tabInfo
	case tabPorts:
		m.tab = tabServices
	case tabRich:
		m.tab = tabPorts
	case tabNetwork:
		m.tab = tabRich
	case tabIPSets:
		m.tab = tabNetwork
	case tabInfo:
		m.tab = tabIPSets
	}
}
