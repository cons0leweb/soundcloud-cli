package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

var (
	orange      = lipgloss.Color("#ff5500")
	muted       = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
	panel       = lipgloss.AdaptiveColor{Light: "#eeeeee", Dark: "#242424"}
	text        = lipgloss.AdaptiveColor{Light: "#171717", Dark: "#f2f2f2"}
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(orange)
	mutedStyle  = lipgloss.NewStyle().Foreground(muted)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4d4d"))
	selected    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Background(orange).Bold(true)
)

func (m Model) View() string {
	if m.width == 0 {
		return "Загрузка SoundCloud…"
	}
	contentWidth := max(24, m.width-4)
	var out strings.Builder

	out.WriteString(headerStyle.Render("SOUNDCLOUD"))
	out.WriteString(mutedStyle.Render("  / терминальный плеер"))
	out.WriteString("\n\n")
	out.WriteString(m.renderSearch(contentWidth))
	out.WriteString("\n")
	out.WriteString(mutedStyle.Render("m Миксы   l Лайки   h История   / Поиск"))
	out.WriteString("\n")

	if len(m.tracks) == 0 && !m.searching {
		out.WriteString("\n")
		out.WriteString(mutedStyle.Render("Ищите треки без открытого браузера."))
		out.WriteString("\n")
		out.WriteString(mutedStyle.Render("Запрос · @пользователь · @пользователь/sets · ссылка SoundCloud"))
	} else {
		out.WriteString(m.renderTracks(contentWidth))
	}

	footerHeight := 5
	used := strings.Count(out.String(), "\n") + footerHeight
	if m.height > used {
		out.WriteString(strings.Repeat("\n", m.height-used))
	} else {
		out.WriteString("\n")
	}
	out.WriteString(m.renderNowPlaying(contentWidth))
	out.WriteString("\n")
	out.WriteString(m.renderStatusAndHelp(contentWidth))

	return lipgloss.NewStyle().Padding(1, 2).Foreground(text).Render(out.String())
}

func (m Model) renderSearch(width int) string {
	prompt := "/ "
	query := string(m.query)
	if m.searchFocus {
		query += "█"
	}
	line := prompt + query
	style := lipgloss.NewStyle().Width(max(10, width-2)).Padding(0, 1).Background(panel)
	if m.searchFocus {
		style = style.BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(orange)
	} else {
		style = style.BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(panel)
	}
	return style.Render(truncate(line, max(8, width-5)))
}

func (m Model) renderTracks(width int) string {
	available := max(3, m.height-13)
	start := 0
	if m.cursor >= available {
		start = m.cursor - available + 1
	}
	end := min(len(m.tracks), start+available)

	var lines []string
	for index := start; index < end; index++ {
		track := m.tracks[index]
		marker := "  "
		if track.Collection {
			marker = "# "
		}
		isCurrent := m.hasCurrent && track.URL == m.currentTrack.URL
		if isCurrent && m.playback != playbackStopped {
			marker = "> "
		}
		right := fmt.Sprintf("%s  %s", formatDuration(track.Duration), formatCount(track.Plays))
		if track.Collection {
			right = "СЕТ  Enter открыть"
		}
		leftWidth := max(8, width-utf8.RuneCountInString(right)-5)
		left := truncate(track.Title+" — "+track.Artist, leftWidth)
		gap := strings.Repeat(" ", max(1, width-utf8.RuneCountInString(marker+left+right)-1))
		line := marker + left + gap + right
		if index == m.cursor && !m.searchFocus {
			line = selected.Width(max(1, width)).Render(truncate(line, width))
		} else if isCurrent && m.playback != playbackStopped {
			line = lipgloss.NewStyle().Foreground(orange).Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderNowPlaying(width int) string {
	if !m.hasCurrent {
		return mutedStyle.Render("СЕЙЧАС ИГРАЕТ  —")
	}
	track := m.currentTrack
	state := "остановлено"
	switch m.playback {
	case playbackLoading:
		state = spinnerFrames[m.spinner] + " загрузка"
	case playbackPlaying:
		state = "играет"
	case playbackPaused:
		state = "пауза"
	}
	progress := formatDuration(int(m.elapsed().Seconds())) + " / " + formatDuration(track.Duration)
	line := fmt.Sprintf("СЕЙЧАС ИГРАЕТ  %s — %s  [%s · %s]", track.Title, track.Artist, state, progress)
	return lipgloss.NewStyle().Bold(true).Width(width).Render(truncate(line, width))
}

func (m Model) renderStatusAndHelp(width int) string {
	status := m.statusText
	if m.searching {
		status = spinnerFrames[m.spinner] + " " + status
	}
	if m.errorText != "" {
		status = errorStyle.Render(truncate(m.errorText, width))
	} else {
		status = mutedStyle.Render(truncate(status, width))
	}

	help := "Enter — поиск"
	if !m.searchFocus {
		help = "↑/↓ выбор  Enter играть/открыть  Space пауза  n/p след/пред  b назад  m/l/h разделы  q выход"
	} else if len(m.tracks) > 0 {
		help = "Enter поиск  Esc результаты  Ctrl+C выход"
	} else {
		help = "Enter поиск  Esc меню разделов  Ctrl+C выход"
	}
	return status + "\n" + mutedStyle.Render(truncate(help, width))
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

func formatDuration(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := seconds / 3600
	minutes := seconds % 3600 / 60
	secs := seconds % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

func formatCount(count int64) string {
	switch {
	case count >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(count)/1_000_000_000)
	case count >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
	case count >= 1_000:
		return fmt.Sprintf("%.1fK", float64(count)/1_000)
	default:
		return fmt.Sprintf("%d", count)
	}
}
