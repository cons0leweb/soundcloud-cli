package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cons0leweb/soundcloud-cli/internal/soundcloud"
)

type catalog interface {
	Search(context.Context, string) ([]soundcloud.Track, error)
	Expand(context.Context, soundcloud.Track) ([]soundcloud.Track, error)
	StreamURL(context.Context, string) (string, error)
	Mixes(context.Context) ([]soundcloud.Track, error)
	Likes(context.Context) ([]soundcloud.Track, error)
	History(context.Context) ([]soundcloud.Track, error)
}

type audioPlayer interface {
	Play(string) (<-chan error, error)
	TogglePause() (bool, error)
	Stop()
}

type playbackState int

const (
	playbackStopped playbackState = iota
	playbackLoading
	playbackPlaying
	playbackPaused
)

type Model struct {
	catalog catalog
	player  audioPlayer

	width  int
	height int

	query          []rune
	searchFocus    bool
	searching      bool
	tracks         []soundcloud.Track
	cursor         int
	errorText      string
	statusText     string
	spinner        int
	catalogRequest int

	playback     playbackState
	playingIndex int
	currentTrack soundcloud.Track
	hasCurrent   bool
	playRequest  int
	startedAt    time.Time
	pausedAt     time.Time
	pausedFor    time.Duration
	history      []viewSnapshot
}

type viewSnapshot struct {
	query  []rune
	tracks []soundcloud.Track
	cursor int
}

func New(catalog catalog, player audioPlayer) Model {
	return Model{
		catalog:      catalog,
		player:       player,
		searchFocus:  true,
		playingIndex: -1,
		statusText:   "Введите запрос и нажмите Enter",
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		m.spinner = (m.spinner + 1) % len(spinnerFrames)
		return m, tickCmd()
	case searchResultMsg:
		if msg.request != m.catalogRequest {
			return m, nil
		}
		m.searching = false
		if msg.err != nil {
			if msg.browse && len(m.history) > 0 {
				m.history = m.history[:len(m.history)-1]
			}
			m.errorText = msg.err.Error()
			m.statusText = "Поиск не удался"
			return m, nil
		}
		m.tracks = msg.tracks
		m.cursor = 0
		m.playingIndex = m.currentIndex(msg.tracks)
		if msg.browse || msg.section {
			m.query = []rune(msg.title)
		}
		if !msg.browse {
			m.history = nil
		}
		m.searchFocus = false
		m.errorText = ""
		m.statusText = fmt.Sprintf("Найдено треков: %d", len(msg.tracks))
		return m, nil
	case streamResolvedMsg:
		if msg.request != m.playRequest {
			return m, nil
		}
		if msg.err != nil {
			m.playback = playbackStopped
			m.errorText = msg.err.Error()
			m.statusText = "Не удалось начать воспроизведение"
			return m, nil
		}
		done, err := m.player.Play(msg.streamURL)
		if err != nil {
			m.playback = playbackStopped
			m.errorText = err.Error()
			m.statusText = "Не удалось начать воспроизведение"
			return m, nil
		}
		m.playback = playbackPlaying
		m.startedAt = time.Now()
		m.pausedFor = 0
		m.errorText = ""
		m.statusText = "Воспроизведение"
		return m, waitForPlayback(msg.request, done)
	case playbackEndedMsg:
		if msg.request != m.playRequest {
			return m, nil
		}
		m.playback = playbackStopped
		m.statusText = "Воспроизведение завершено"
		if msg.err != nil {
			m.errorText = "Воспроизведение неожиданно завершилось"
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.String() == "ctrl+c" {
		m.player.Stop()
		return m, tea.Quit
	}

	if m.searchFocus {
		switch key.Type {
		case tea.KeyEnter:
			if m.searching || strings.TrimSpace(string(m.query)) == "" {
				return m, nil
			}
			m.searching = true
			m.catalogRequest++
			m.errorText = ""
			m.statusText = "Ищу в SoundCloud"
			return m, searchCmd(m.catalog, string(m.query), false, false, "", m.catalogRequest)
		case tea.KeyEsc:
			if len(m.tracks) > 0 {
				m.searchFocus = false
			} else if len(m.query) > 0 {
				m.query = nil
			} else {
				m.searchFocus = false
			}
		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
			}
		case tea.KeyDown:
			if len(m.tracks) > 0 {
				m.searchFocus = false
			}
		case tea.KeyRunes:
			m.query = append(m.query, key.Runes...)
		}
		return m, nil
	}

	switch key.String() {
	case "q":
		m.player.Stop()
		return m, tea.Quit
	case "/":
		m.searchFocus = true
		return m, nil
	case "esc":
		m.searchFocus = true
		return m, nil
	case "j", "down":
		if m.cursor < len(m.tracks)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.tracks) && m.tracks[m.cursor].Collection {
			return m.browseCollection(m.cursor)
		}
		return m.beginPlayback(m.cursor)
	case "b":
		if len(m.history) > 0 {
			last := m.history[len(m.history)-1]
			m.history = m.history[:len(m.history)-1]
			m.query = append([]rune(nil), last.query...)
			m.tracks = last.tracks
			m.cursor = last.cursor
			m.playingIndex = m.currentIndex(last.tracks)
			m.statusText = "Возврат к предыдущему списку"
		}
	case "m":
		return m.loadSection("Персональные миксы", m.catalog.Mixes)
	case "l":
		return m.loadSection("Мои лайки", m.catalog.Likes)
	case "h":
		return m.loadSection("История прослушивания", m.catalog.History)
	case " ":
		if m.playback != playbackPlaying && m.playback != playbackPaused {
			return m, nil
		}
		paused, err := m.player.TogglePause()
		if err != nil {
			m.errorText = err.Error()
			return m, nil
		}
		if paused {
			m.playback = playbackPaused
			m.pausedAt = time.Now()
			m.statusText = "Пауза"
		} else {
			m.playback = playbackPlaying
			m.pausedFor += time.Since(m.pausedAt)
			m.statusText = "Воспроизведение"
		}
	case "s":
		m.stopPlayback("Остановлено")
	case "n":
		if len(m.tracks) > 0 {
			base := m.playingIndex
			if base < 0 {
				base = m.cursor
			}
			return m.beginPlayback((base + 1) % len(m.tracks))
		}
	case "p":
		if len(m.tracks) > 0 {
			base := m.playingIndex
			if base < 0 {
				base = m.cursor
			}
			return m.beginPlayback((base - 1 + len(m.tracks)) % len(m.tracks))
		}
	}
	return m, nil
}

func (m Model) browseCollection(index int) (tea.Model, tea.Cmd) {
	track := m.tracks[index]
	m.history = append(m.history, viewSnapshot{
		query:  append([]rune(nil), m.query...),
		tracks: m.tracks,
		cursor: m.cursor,
	})
	m.searching = true
	m.catalogRequest++
	m.errorText = ""
	m.statusText = "Открываю сет или микс"
	return m, collectionCmd(m.catalog, track, track.Title, m.catalogRequest)
}

func (m Model) loadSection(title string, load func(context.Context) ([]soundcloud.Track, error)) (tea.Model, tea.Cmd) {
	if m.searching {
		return m, nil
	}
	m.searching = true
	m.catalogRequest++
	m.errorText = ""
	m.statusText = "Загружаю: " + title
	return m, sectionCmd(load, title, m.catalogRequest)
}

func (m Model) beginPlayback(index int) (tea.Model, tea.Cmd) {
	if index < 0 || index >= len(m.tracks) {
		return m, nil
	}
	m.player.Stop()
	m.playRequest++
	m.playingIndex = index
	m.currentTrack = m.tracks[index]
	m.hasCurrent = true
	m.cursor = index
	m.playback = playbackLoading
	m.errorText = ""
	m.statusText = "Получаю аудиопоток"
	return m, startPlaybackCmd(m.catalog, m.tracks[index], m.playRequest)
}

func (m Model) currentIndex(tracks []soundcloud.Track) int {
	if !m.hasCurrent {
		return -1
	}
	for index, track := range tracks {
		if track.URL == m.currentTrack.URL {
			return index
		}
	}
	return -1
}

func (m *Model) stopPlayback(status string) {
	m.player.Stop()
	m.playRequest++
	m.playback = playbackStopped
	m.statusText = status
}

func (m Model) elapsed() time.Duration {
	if m.startedAt.IsZero() || m.playback == playbackLoading || m.playback == playbackStopped {
		return 0
	}
	end := time.Now()
	if m.playback == playbackPaused {
		end = m.pausedAt
	}
	elapsed := end.Sub(m.startedAt) - m.pausedFor
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

type tickMsg time.Time

type searchResultMsg struct {
	tracks  []soundcloud.Track
	err     error
	browse  bool
	section bool
	title   string
	request int
}

type streamResolvedMsg struct {
	request   int
	streamURL string
	err       error
}

type playbackEndedMsg struct {
	request int
	err     error
}

var spinnerFrames = []string{"·  ", "·· ", "···", " ··", "  ·", "   "}

func tickCmd() tea.Cmd {
	return tea.Tick(180*time.Millisecond, func(now time.Time) tea.Msg { return tickMsg(now) })
}

func searchCmd(c catalog, query string, browse, section bool, title string, request int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tracks, err := c.Search(ctx, query)
		return searchResultMsg{tracks: tracks, err: err, browse: browse, section: section, title: title, request: request}
	}
}

func sectionCmd(load func(context.Context) ([]soundcloud.Track, error), title string, request int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tracks, err := load(ctx)
		return searchResultMsg{tracks: tracks, err: err, section: true, title: title, request: request}
	}
}

func collectionCmd(c catalog, collection soundcloud.Track, title string, request int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tracks, err := c.Expand(ctx, collection)
		return searchResultMsg{tracks: tracks, err: err, browse: true, title: title, request: request}
	}
}

func startPlaybackCmd(c catalog, track soundcloud.Track, request int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		streamURL, err := c.StreamURL(ctx, track.URL)
		return streamResolvedMsg{request: request, streamURL: streamURL, err: err}
	}
}

func waitForPlayback(request int, done <-chan error) tea.Cmd {
	return func() tea.Msg {
		err := <-done
		return playbackEndedMsg{request: request, err: err}
	}
}
