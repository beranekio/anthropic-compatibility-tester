package mockserver

import (
	"strconv"
	"sync"
)

type batchStore struct {
	mu      sync.Mutex
	nextID  int
	batches map[string]map[string]any
}

func newBatchStore() *batchStore {
	return &batchStore{batches: make(map[string]map[string]any)}
}

func (s *batchStore) create(status string) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := "msgbatch_mock_" + strconv.Itoa(s.nextID)
	payload := mockMessageBatchPayload(id, status)
	s.batches[id] = payload
	return cloneMap(payload)
}

func (s *batchStore) get(id string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, ok := s.batches[id]
	if !ok {
		return nil, false
	}
	return cloneMap(payload), true
}

func (s *batchStore) update(id, status string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, ok := s.batches[id]
	if !ok {
		return nil, false
	}
	payload["processing_status"] = status
	if status == "canceling" {
		payload["cancel_initiated_at"] = payload["created_at"]
	}
	return cloneMap(payload), true
}

func (s *batchStore) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.batches[id]; !ok {
		return false
	}
	delete(s.batches, id)
	return true
}

func (s *batchStore) listAll() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, 0, len(s.batches))
	for _, payload := range s.batches {
		out = append(out, cloneMap(payload))
	}
	return out
}

type fileStore struct {
	mu    sync.Mutex
	next  int
	files map[string]fileEntry
}

type fileEntry struct {
	metadata map[string]any
	content  []byte
}

func newFileStore() *fileStore {
	return &fileStore{files: make(map[string]fileEntry)}
}

func (s *fileStore) create(filename string, content []byte) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	id := "file_mock_" + strconv.Itoa(s.next)
	meta := mockFileMetadata(id, filename, int64(len(content)))
	s.files[id] = fileEntry{metadata: meta, content: append([]byte(nil), content...)}
	return cloneMap(meta)
}

func (s *fileStore) get(id string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.files[id]
	if !ok {
		return nil, false
	}
	return cloneMap(entry.metadata), true
}

func (s *fileStore) content(id string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.files[id]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), entry.content...), true
}

func (s *fileStore) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.files[id]; !ok {
		return false
	}
	delete(s.files, id)
	return true
}

func (s *fileStore) listAll() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, 0, len(s.files))
	for _, entry := range s.files {
		out = append(out, cloneMap(entry.metadata))
	}
	return out
}

type skillStore struct {
	mu     sync.Mutex
	next   int
	skills map[string]skillEntry
}

type skillEntry struct {
	metadata map[string]any
	versions map[string]map[string]any
}

func newSkillStore() *skillStore {
	return &skillStore{skills: make(map[string]skillEntry)}
}

func (s *skillStore) create() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	id := "skill_mock_" + strconv.Itoa(s.next)
	meta := mockSkillPayload(id)
	version := mockSkillVersionPayload(id, "1759178010641129")
	s.skills[id] = skillEntry{
		metadata: meta,
		versions: map[string]map[string]any{version["version"].(string): version},
	}
	return cloneMap(meta)
}

func (s *skillStore) get(id string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.skills[id]
	if !ok {
		return nil, false
	}
	return cloneMap(entry.metadata), true
}

func (s *skillStore) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.skills[id]; !ok {
		return false
	}
	delete(s.skills, id)
	return true
}

func (s *skillStore) addVersion(skillID string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.skills[skillID]
	if !ok {
		return nil, false
	}
	version := mockSkillVersionPayload(skillID, "1759178010641130")
	entry.versions[version["version"].(string)] = version
	s.skills[skillID] = entry
	return cloneMap(version), true
}

func (s *skillStore) getVersion(skillID, version string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.skills[skillID]
	if !ok {
		return nil, false
	}
	payload, ok := entry.versions[version]
	if !ok {
		return nil, false
	}
	return cloneMap(payload), true
}

func (s *skillStore) listAll() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, 0, len(s.skills))
	for _, entry := range s.skills {
		out = append(out, cloneMap(entry.metadata))
	}
	return out
}

func (s *skillStore) listVersions(skillID string) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.skills[skillID]
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(entry.versions))
	for _, payload := range entry.versions {
		out = append(out, cloneMap(payload))
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}