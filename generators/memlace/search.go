package memlace

import (
	"strings"
)

// MatchResult 搜索匹配结果。
type MatchResult struct {
	Entry  PreferenceEntry `json:"entry"`
	Score  float64         `json:"score"`  // 0.0~1.0 越高越相关
	Match  string          `json:"match"`  // 匹配的字段
}

// Searcher 提供语义化搜索能力（基于关键词 + 字段权重）。
type Searcher struct {
	store *Store
}

// NewSearcher 创建搜索器。
func NewSearcher(store *Store) *Searcher {
	return &Searcher{store: store}
}

// Search 执行搜索并评分。
// 字段权重：key > category > value。
func (s *Searcher) Search(query string, limit int) ([]MatchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return []MatchResult{}, nil
	}

	// 1. FTS 搜索
	entries, err := s.store.Search(query, limit*2)
	if err != nil {
		entries = nil
	}

	// 2. 评分 + 排序
	var results []MatchResult
	seen := make(map[string]bool)

	for _, e := range entries {
		if seen[e.ID] {
			continue
		}
		seen[e.ID] = true
		score, match := scoreEntry(e, query)
		results = append(results, MatchResult{
			Entry: e,
			Score: score,
			Match: match,
		})
	}

	// 3. 降级：精确 key 匹配优先
	sorted := results
	sortMatchResults(sorted)

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	if sorted == nil {
		sorted = []MatchResult{}
	}
	return sorted, nil
}

// ScoreEntry 给单条偏好打分（0.0~1.0）。
func scoreEntry(e PreferenceEntry, query string) (float64, string) {
	q := strings.ToLower(query)
	fields := []struct {
		text  string
		weight float64
	}{
		{e.Key, 1.0},
		{e.Category, 0.7},
		{e.Value, 0.5},
		{e.Source, 0.3},
	}
	bestScore := 0.0
	bestMatch := ""

	for _, f := range fields {
		lower := strings.ToLower(f.text)
		if lower == q {
			return f.weight, f.text // 精确匹配
		}
		if strings.Contains(lower, q) {
			score := f.weight * 0.8
			if score > bestScore {
				bestScore = score
				bestMatch = f.text
			}
		}
	}
	return bestScore, bestMatch
}

func sortMatchResults(r []MatchResult) {
	// shell sort（简单）
	for i := 1; i < len(r); i++ {
		for j := i; j > 0 && r[j].Score > r[j-1].Score; j-- {
			r[j], r[j-1] = r[j-1], r[j]
		}
	}
}
