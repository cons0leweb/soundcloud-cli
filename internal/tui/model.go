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
	StreamURL(context.Context, soundcloud.Track) (string, error)
	Mixes(context.Context) ([]soundcloud.Track, error)
	Likes(context.Context) ([]soundcloud.Track, error)
	History(context.Context) ([]soundcloud.Track, error)
	Station(context.Context, soundcloud.Track) ([]soundcloud.Track, error)
}

type audioPlayer interface {
	Play(string) (<-chan error, error)
	TogglePause() (bool, error)
	AdjustVolume(int) (int, error)
	ToggleMute() (bool, error)
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
	waveFrame      uint64
	catalogRequest int
	activeView     string
	helpVisible    bool
	radioMode      bool
	radioLoading   bool
	radioResume    bool
	radioRequest   int

	playback     playbackState
	currentTrack soundcloud.Track
	hasCurrent   bool
	queue        playbackQueue
	volume       int
	muted        bool
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
	view   string
}

func New(catalog catalog, player audioPlayer) Model {
	return Model{
		catalog:     catalog,
		player:      player,
		searchFocus: true,
		queue:       newPlaybackQueue(),
		volume:      80,
		activeView:  "search",
		statusText:  "Введите запрос и нажмите Enter",
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
		if m.playback == playbackPlaying {
			m.waveFrame++
		}
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
		if msg.browse || msg.section {
			m.query = []rune(msg.title)
		}
		if msg.section {
			m.activeView = msg.view
		} else if !msg.browse {
			m.activeView = "search"
		}
		if !msg.browse {
			m.history = nil
		}
		m.searchFocus = false
		m.errorText = ""
		m.statusText = fmt.Sprintf("Найдено треков: %d", len(msg.tracks))
		return m, nil
	case stationResultMsg:
		if msg.request != m.radioRequest {
			return m, nil
		}
		m.radioLoading = false
		if msg.err != nil {
			resume := m.radioResume
			m.radioResume = false
			if msg.initial {
				m.radioMode = false
			}
			if resume {
				m.playback = playbackStopped
			}
			m.errorText = msg.err.Error()
			m.statusText = "Не удалось продолжить радио"
			return m, nil
		}
		if msg.initial {
			m.queue = newPlaybackQueue()
			m.queue.appendUnique(append([]soundcloud.Track{msg.seed}, msg.tracks...))
			m.queue.repeat = repeatOff
			m.queue.shuffle = false
			m.tracks = append([]soundcloud.Track(nil), m.queue.tracks...)
			m.query = []rune("Радио: " + msg.seed.Title)
			m.activeView = "radio"
			m.history = nil
			m.cursor = 0
			m.radioMode = true
			m.errorText = ""
			m.statusText = fmt.Sprintf("Радио готово · %d треков", len(m.tracks))
			return m.beginQueuedPlayback(0)
		}
		resume := msg.autoPlay || m.radioResume
		m.radioResume = false
		first := m.queue.appendUnique(msg.tracks)
		m.tracks = append([]soundcloud.Track(nil), m.queue.tracks...)
		if first < 0 {
			m.statusText = "SoundCloud не нашёл новых треков для радио"
			return m, nil
		}
		m.statusText = fmt.Sprintf("Радио пополнено · ещё %d треков", len(m.queue.tracks)-first)
		if resume {
			return m.beginQueuedPlayback(first)
		}
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
		m.muted = false
		m.startedAt = time.Now()
		m.pausedFor = 0
		m.errorText = ""
		m.statusText = "Воспроизведение"
		return m, waitForPlayback(msg.request, done)
	case playbackEndedMsg:
		if msg.request != m.playRequest {
			return m, nil
		}
		if msg.err != nil {
			m.playback = playbackStopped
			m.errorText = "Воспроизведение неожиданно завершилось"
			m.statusText = "Воспроизведение завершено"
			return m, nil
		}
		if next := m.queue.next(false); next >= 0 {
			return m.beginQueuedPlayback(next)
		}
		if m.radioMode {
			m.playback = playbackLoading
			if m.radioLoading {
				m.radioResume = true
				m.statusText = "Жду продолжение радио"
				return m, nil
			}
			return m.refillRadio(m.currentTrack, true)
		}
		m.playback = playbackStopped
		m.statusText = "Очередь завершена"
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
	if m.helpVisible {
		if key.String() == "?" || key.String() == "esc" || key.String() == "q" {
			m.helpVisible = false
		}
		return m, nil
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
		case tea.KeySpace:
			m.query = append(m.query, ' ')
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
	case "?":
		m.helpVisible = true
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
	case "a":
		if m.cursor < 0 || m.cursor >= len(m.tracks) || m.tracks[m.cursor].Collection {
			return m, nil
		}
		return m.startRadio(m.tracks[m.cursor])
	case "b":
		if len(m.history) > 0 {
			last := m.history[len(m.history)-1]
			m.history = m.history[:len(m.history)-1]
			m.query = append([]rune(nil), last.query...)
			m.tracks = last.tracks
			m.cursor = last.cursor
			m.activeView = last.view
			m.statusText = "Возврат к предыдущему списку"
		}
	case "m":
		return m.loadSection("Персональные миксы", "mixes", m.catalog.Mixes)
	case "l":
		return m.loadSection("Мои лайки", "likes", m.catalog.Likes)
	case "h":
		return m.loadSection("История прослушивания", "history", m.catalog.History)
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
		if m.radioMode && m.queue.remaining() == 0 {
			if m.radioLoading {
				m.radioResume = true
				m.statusText = "Жду продолжение радио"
				return m, nil
			}
			return m.refillRadio(m.currentTrack, true)
		}
		if next := m.queue.next(true); next >= 0 {
			return m.beginQueuedPlayback(next)
		}
	case "p":
		if previous := m.queue.previous(); previous >= 0 {
			return m.beginQueuedPlayback(previous)
		}
	case "z":
		m.queue.shuffle = !m.queue.shuffle
		m.statusText = map[bool]string{true: "Случайный порядок включён", false: "Случайный порядок выключен"}[m.queue.shuffle]
	case "r":
		m.queue.cycleRepeat()
		m.statusText = "Повтор: " + m.repeatLabel()
	case "+", "=":
		return m.adjustVolume(5)
	case "-":
		return m.adjustVolume(-5)
	case "x":
		if m.playback == playbackPlaying || m.playback == playbackPaused {
			muted, err := m.player.ToggleMute()
			if err != nil {
				m.errorText = err.Error()
				return m, nil
			}
			m.muted = muted
			m.statusText = map[bool]string{true: "Звук выключен", false: "Звук включён"}[muted]
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
		view:   m.activeView,
	})
	m.searching = true
	m.catalogRequest++
	m.errorText = ""
	m.statusText = "Открываю сет или микс"
	return m, collectionCmd(m.catalog, track, track.Title, m.catalogRequest)
}

func (m Model) loadSection(title, view string, load func(context.Context) ([]soundcloud.Track, error)) (tea.Model, tea.Cmd) {
	if m.searching {
		return m, nil
	}
	m.searching = true
	m.catalogRequest++
	m.errorText = ""
	m.statusText = "Загружаю: " + title
	return m, sectionCmd(load, title, view, m.catalogRequest)
}

func (m Model) beginPlayback(index int) (tea.Model, tea.Cmd) {
	if index < 0 || index >= len(m.tracks) {
		return m, nil
	}
	selected := m.tracks[index]
	m.radioMode = false
	m.radioLoading = false
	m.radioResume = false
	m.radioRequest++
	queueIndex := m.queue.replace(m.tracks, selected.URL)
	return m.beginQueuedPlayback(queueIndex)
}

func (m Model) beginQueuedPlayback(index int) (tea.Model, tea.Cmd) {
	track, ok := m.queue.selectIndex(index)
	if !ok {
		return m, nil
	}
	m.player.Stop()
	m.playRequest++
	m.currentTrack = track
	m.hasCurrent = true
	if current := m.currentIndex(m.tracks); current >= 0 {
		m.cursor = current
	}
	m.playback = playbackLoading
	m.errorText = ""
	m.statusText = "Получаю аудиопоток"
	play := startPlaybackCmd(m.catalog, track, m.playRequest)
	if m.radioMode && !m.radioLoading && m.queue.remaining() <= 3 {
		updated, refill := m.refillRadio(track, false)
		m = updated.(Model)
		return m, tea.Batch(play, refill)
	}
	return m, play
}

func (m Model) startRadio(seed soundcloud.Track) (tea.Model, tea.Cmd) {
	m.radioRequest++
	m.radioLoading = true
	m.errorText = ""
	m.statusText = "Строю радио по треку: " + seed.Title
	return m, stationCmd(m.catalog, seed, m.radioRequest, true, false)
}

func (m Model) refillRadio(seed soundcloud.Track, autoPlay bool) (tea.Model, tea.Cmd) {
	if m.radioLoading {
		return m, nil
	}
	m.radioRequest++
	m.radioLoading = true
	m.errorText = ""
	m.statusText = "Подбираю продолжение радио"
	return m, stationCmd(m.catalog, seed, m.radioRequest, false, autoPlay)
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

func (m Model) adjustVolume(delta int) (tea.Model, tea.Cmd) {
	if m.playback != playbackPlaying && m.playback != playbackPaused {
		return m, nil
	}
	volume, err := m.player.AdjustVolume(delta)
	if err != nil {
		m.errorText = err.Error()
		return m, nil
	}
	m.volume = volume
	m.muted = false
	m.statusText = fmt.Sprintf("Громкость: %d%%", volume)
	return m, nil
}

func (m Model) repeatLabel() string {
	switch m.queue.repeat {
	case repeatAll:
		return "вся очередь"
	case repeatOne:
		return "один трек"
	default:
		return "выключен"
	}
}

func (m *Model) stopPlayback(status string) {
	m.player.Stop()
	m.playRequest++
	m.radioRequest++
	m.radioLoading = false
	m.radioResume = false
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
	view    string
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

type stationResultMsg struct {
	seed     soundcloud.Track
	tracks   []soundcloud.Track
	err      error
	request  int
	initial  bool
	autoPlay bool
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

func sectionCmd(load func(context.Context) ([]soundcloud.Track, error), title, view string, request int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tracks, err := load(ctx)
		return searchResultMsg{tracks: tracks, err: err, section: true, title: title, view: view, request: request}
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

func stationCmd(c catalog, seed soundcloud.Track, request int, initial, autoPlay bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tracks, err := c.Station(ctx, seed)
		return stationResultMsg{seed: seed, tracks: tracks, err: err, request: request, initial: initial, autoPlay: autoPlay}
	}
}

func startPlaybackCmd(c catalog, track soundcloud.Track, request int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		streamURL, err := c.StreamURL(ctx, track)
		return streamResolvedMsg{request: request, streamURL: streamURL, err: err}
	}
}

func waitForPlayback(request int, done <-chan error) tea.Cmd {
	return func() tea.Msg {
		err := <-done
		return playbackEndedMsg{request: request, err: err}
	}
}
