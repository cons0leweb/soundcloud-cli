package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	orange     = lipgloss.Color("#ff5500")
	orangeSoft = lipgloss.AdaptiveColor{Light: "#c74400", Dark: "#ff7a33"}
	muted      = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#8a8a8a"}
	panel      = lipgloss.AdaptiveColor{Light: "#eeeeee", Dark: "#252525"}
	panelSoft  = lipgloss.AdaptiveColor{Light: "#f6f6f6", Dark: "#191919"}
	text       = lipgloss.AdaptiveColor{Light: "#171717", Dark: "#f2f2f2"}
	header     = lipgloss.NewStyle().Bold(true).Foreground(orange)
	mutedText  = lipgloss.NewStyle().Foreground(muted)
	errorText  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff4d4d"))
	selected   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Background(orange).Bold(true)
	playing    = lipgloss.NewStyle().Foreground(orangeSoft).Bold(true)
)

func (m Model) View() string {
	if m.width == 0 {
		return "Загрузка SoundCloud…"
	}
	horizontalPadding := 2
	if m.width < 56 {
		horizontalPadding = 1
	}
	contentWidth := max(24, m.width-horizontalPadding*2)
	innerHeight := max(12, m.height-2)
	var out strings.Builder

	out.WriteString(m.renderHeader(contentWidth))
	out.WriteString("\n")
	out.WriteString(m.renderNavigation(contentWidth))
	out.WriteString("\n\n")
	out.WriteString(m.renderSearch(contentWidth))
	out.WriteString("\n\n")

	if m.helpVisible {
		out.WriteString(m.renderHelp(contentWidth))
	} else if m.searching && len(m.tracks) == 0 {
		out.WriteString(m.renderLoading(contentWidth))
	} else if len(m.tracks) == 0 && !m.searching {
		out.WriteString(m.renderEmptyState(contentWidth))
	} else {
		out.WriteString(m.renderTracks(contentWidth))
	}

	footerHeight := 7
	used := strings.Count(out.String(), "\n") + footerHeight
	if innerHeight > used {
		out.WriteString(strings.Repeat("\n", innerHeight-used))
	} else {
		out.WriteString("\n")
	}
	out.WriteString(m.renderPlayer(contentWidth))
	out.WriteString("\n")
	out.WriteString(m.renderStatusAndHelp(contentWidth))

	return lipgloss.NewStyle().Padding(1, horizontalPadding).Foreground(text).Render(out.String())
}

func (m Model) renderLoading(width int) string {
	lines := []string{
		header.Render(spinnerFrames[m.spinner] + "  " + m.statusText),
		mutedText.Render("SoundCloud отвечает — текущий экран сохранён."),
	}
	for index, line := range lines {
		lines[index] = truncate(line, width)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderHeader(width int) string {
	brand := header.Render("▰ SOUNDCLOUD")
	tagline := mutedText.Render("TERMINAL PLAYER")
	gap := strings.Repeat(" ", max(1, width-lipgloss.Width(brand)-lipgloss.Width(tagline)))
	return brand + gap + tagline
}

func (m Model) renderNavigation(width int) string {
	items := []string{
		renderTab("/", "ПОИСК", m.activeView == "search"),
		renderTab("m", "МИКСЫ", m.activeView == "mixes"),
		renderTab("l", "ЛАЙКИ", m.activeView == "likes"),
		renderTab("h", "ИСТОРИЯ", m.activeView == "history"),
	}
	return truncate(strings.Join(items, "   "), width)
}

func renderTab(key, label string, active bool) string {
	value := key + " " + label
	if active {
		return lipgloss.NewStyle().Bold(true).Underline(true).Foreground(orange).Render(value)
	}
	return mutedText.Render(value)
}

func (m Model) renderSearch(width int) string {
	query := string(m.query)
	if query == "" {
		query = "трек, исполнитель, @профиль или ссылка"
		if !m.searchFocus {
			query = mutedText.Render(query)
		}
	}
	if m.searchFocus {
		query += "█"
	}
	prefix := lipgloss.NewStyle().Bold(true).Foreground(orange).Render("/") + "  "
	line := prefix + query
	style := lipgloss.NewStyle().Width(max(10, width-2)).Padding(0, 1).Background(panelSoft)
	if m.searchFocus {
		style = style.Foreground(text).Bold(true).BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(orange)
	} else {
		style = style.BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(panel)
	}
	return style.Render(truncate(line, max(8, width-4)))
}

func (m Model) renderEmptyState(width int) string {
	lines := []string{
		header.Render("Найдите музыку и нажмите Enter"),
		mutedText.Render("Обычный запрос ищет треки · @username открывает профиль"),
		mutedText.Render("Персональная медиатека: m миксы · l лайки · h история"),
		"",
		mutedText.Render("Нажмите ? в режиме списка, чтобы увидеть все клавиши."),
	}
	for index, line := range lines {
		lines[index] = truncate(line, width)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderTracks(width int) string {
	available := max(3, m.height-16)
	start := 0
	if m.cursor >= available {
		start = m.cursor - available + 1
	}
	end := min(len(m.tracks), start+available)

	title := fmt.Sprintf("%s  ·  %d", viewLabel(m.activeView), len(m.tracks))
	lines := []string{lipgloss.NewStyle().Bold(true).Render(truncate(title, width))}
	for index := start; index < end; index++ {
		track := m.tracks[index]
		isCurrent := m.hasCurrent && track.URL == m.currentTrack.URL
		marker := "  "
		switch {
		case isCurrent && m.playback != playbackStopped:
			marker = "▶ "
		case track.Collection:
			marker = "◆ "
		}

		right := fmt.Sprintf("%s  %s", formatDuration(track.Duration), formatCount(track.Plays))
		if track.Collection {
			right = "МИКС  ↵"
		}
		number := fmt.Sprintf("%2d ", index+1)
		leftWidth := max(8, width-lipgloss.Width(number+marker+right)-2)
		left := truncate(track.Title+"  ·  "+track.Artist, leftWidth)
		gap := strings.Repeat(" ", max(1, width-lipgloss.Width(number+marker+left+right)))
		line := number + marker + left + gap + right
		if index == m.cursor && !m.searchFocus {
			line = selected.Width(width).Render(truncate(line, width))
		} else if isCurrent && m.playback != playbackStopped {
			line = playing.Render(line)
		} else {
			line = truncate(line, width)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func viewLabel(view string) string {
	switch view {
	case "mixes":
		return "ПЕРСОНАЛЬНЫЕ МИКСЫ"
	case "likes":
		return "МОИ ЛАЙКИ"
	case "history":
		return "ИСТОРИЯ"
	default:
		return "РЕЗУЛЬТАТЫ"
	}
}

func (m Model) renderPlayer(width int) string {
	separator := lipgloss.NewStyle().Foreground(panel).Render(strings.Repeat("─", width))
	if !m.hasCurrent {
		return separator + "\n" + mutedText.Render("НЕТ АКТИВНОГО ТРЕКА") + "\n" + mutedText.Render("Enter — воспроизвести выбранный трек")
	}

	stateIcon, state := "■", "ОСТАНОВЛЕНО"
	switch m.playback {
	case playbackLoading:
		stateIcon, state = "◌", "ЗАГРУЗКА "+spinnerFrames[m.spinner]
	case playbackPlaying:
		stateIcon, state = "▶", "ИГРАЕТ"
	case playbackPaused:
		stateIcon, state = "Ⅱ", "ПАУЗА"
	}
	trackLine := fmt.Sprintf("%s %s  %s · %s", stateIcon, state, m.currentTrack.Title, m.currentTrack.Artist)
	elapsed := int(m.elapsed().Seconds())
	progress := m.renderProgress(max(8, min(34, width-22)), elapsed, m.currentTrack.Duration)
	timeText := formatDuration(elapsed) + " / " + formatDuration(m.currentTrack.Duration)
	position, total := m.queue.position()
	queueText := fmt.Sprintf("ОЧЕРЕДЬ %d/%d", position, total)
	progressGap := strings.Repeat(" ", max(1, width-lipgloss.Width(progress+timeText+queueText)-2))
	progressLine := progress + "  " + timeText + progressGap + queueText

	volume := fmt.Sprintf("VOL %3d%% %s", m.volume, volumeMeter(m.volume, m.muted))
	shuffle := "SHUFFLE OFF"
	if m.queue.shuffle {
		shuffle = "SHUFFLE ON"
	}
	modes := fmt.Sprintf("%s   %s   REPEAT %s", volume, shuffle, strings.ToUpper(m.repeatLabel()))
	if m.muted {
		modes = "MUTED   " + modes
	}
	return separator + "\n" + playing.Render(truncate(trackLine, width)) + "\n" + truncate(progressLine, width) + "\n" + mutedText.Render(truncate(modes, width))
}

func (m Model) renderProgress(width, elapsed, duration int) string {
	if width < 1 {
		return ""
	}
	filled := 0
	if duration > 0 {
		filled = min(width, max(0, elapsed*width/duration))
	}
	return lipgloss.NewStyle().Foreground(orange).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(panel).Render(strings.Repeat("─", width-filled))
}

func volumeMeter(volume int, muted bool) string {
	if muted {
		return "[··········]"
	}
	filled := min(10, max(0, (volume+5)/10))
	return "[" + strings.Repeat("▪", filled) + strings.Repeat("·", 10-filled) + "]"
}

func (m Model) renderStatusAndHelp(width int) string {
	status := m.statusText
	if m.searching {
		status = spinnerFrames[m.spinner] + " " + status
	}
	if m.errorText != "" {
		status = errorText.Render("ОШИБКА  " + truncate(m.errorText, max(1, width-8)))
	} else {
		status = mutedText.Render(truncate(status, width))
	}

	help := "Enter открыть/играть  Space пауза  n/p трек  +/- громкость  x mute  z shuffle  r repeat  ? помощь"
	if m.searchFocus {
		help = "Enter искать  Esc к списку  Ctrl+C выход"
	}
	return status + "\n" + mutedText.Render(truncate(help, width))
}

func (m Model) renderHelp(width int) string {
	title := header.Render("УПРАВЛЕНИЕ") + mutedText.Render("  ·  ? или Esc закрыть")
	columns := []string{
		"НАВИГАЦИЯ                 ПЛЕЕР                    ОЧЕРЕДЬ",
		"↑/↓  j/k   выбор          Space  пауза            n/p  след/пред",
		"Enter      открыть        +/-    громкость        z    shuffle",
		"/          поиск          x      mute             r    repeat",
		"b          назад          s      остановить       q    выход",
		"m/l/h      разделы",
	}
	lines := []string{title, ""}
	for _, line := range columns {
		lines = append(lines, truncate(line, width))
	}
	return strings.Join(lines, "\n")
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	return runewidth.Truncate(value, width, "…")
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
