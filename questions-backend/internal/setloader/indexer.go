package setloader

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var ErrSetNotFound = errors.New("set not found")

type SetMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Length      int    `json:"length"`
	SourcePath  string `json:"-"`
}

type Question struct {
	ID            string   `json:"id"`
	Difficulty    int      `json:"difficulty"`
	Categories    []string `json:"categories"`
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer int      `json:"correctAnswer"`
}

type questionSetFile struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Questions   []Question `json:"questions"`
}

type Indexer struct {
	baseDir string

	mu       sync.RWMutex
	metadata []SetMetadata
	sourceBy map[string]string
}

func NewIndexer(baseDir string) *Indexer {
	return &Indexer{
		baseDir:  baseDir,
		sourceBy: make(map[string]string),
	}
}

func (i *Indexer) LoadAllMetadata() ([]SetMetadata, error) {
	entries, err := os.ReadDir(i.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read sets directory: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	nextMetadata := make([]SetMetadata, 0, len(files))
	nextSource := make(map[string]string, len(files))

	for _, fileName := range files {
		fullPath := filepath.Join(i.baseDir, fileName)
		setFile, err := readSetFile(fullPath)
		if err != nil {
			return nil, err
		}
		if setFile.ID == "" {
			return nil, fmt.Errorf("set id missing in %s", fileName)
		}
		if _, exists := nextSource[setFile.ID]; exists {
			return nil, fmt.Errorf("duplicate set id %q", setFile.ID)
		}

		meta := SetMetadata{
			ID:          setFile.ID,
			Name:        setFile.Name,
			Description: setFile.Description,
			Length:      len(setFile.Questions),
			SourcePath:  fullPath,
		}

		nextMetadata = append(nextMetadata, meta)
		nextSource[setFile.ID] = fullPath
	}

	sort.Slice(nextMetadata, func(a, b int) bool {
		return nextMetadata[a].ID < nextMetadata[b].ID
	})

	i.mu.Lock()
	i.metadata = nextMetadata
	i.sourceBy = nextSource
	i.mu.Unlock()

	return i.ListSets(), nil
}

func (i *Indexer) ListSets() []SetMetadata {
	i.mu.RLock()
	defer i.mu.RUnlock()

	cloned := make([]SetMetadata, len(i.metadata))
	copy(cloned, i.metadata)
	return cloned
}

func (i *Indexer) LoadQuestionsByID(id string) ([]Question, error) {
	i.mu.RLock()
	source, exists := i.sourceBy[id]
	i.mu.RUnlock()

	if !exists {
		return nil, ErrSetNotFound
	}

	setFile, err := readSetFile(source)
	if err != nil {
		return nil, err
	}

	return setFile.Questions, nil
}

func readSetFile(path string) (*questionSetFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read set file %s: %w", path, err)
	}

	var parsed questionSetFile
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode set file %s: %w", path, err)
	}

	return &parsed, nil
}
