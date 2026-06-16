package association

import (
	"strings"

	fingersEngine "github.com/chainreactors/fingers/fingers"
)

// WithSearch sets a full-text substring search across all indexed term values.
func (q *Query) WithSearch(text string) *Query {
	q.Search = text
	return q
}

// empty returns true when the query has no terms and no search text.
func (q *Query) empty() bool {
	if q == nil {
		return true
	}
	return len(q.Fingers) == 0 && len(q.Aliases) == 0 && len(q.Templates) == 0 &&
		len(q.Tags) == 0 && len(q.Services) == 0 && len(q.CPEs) == 0 &&
		len(q.CVEs) == 0 && len(q.Attributes) == 0 && q.Search == ""
}

// searchRefs performs substring matching across all indexed terms.
func (idx *Index) searchRefs(text string) []entityRef {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return nil
	}
	var refs []entityRef
	seen := make(map[entityRef]struct{})
	for t, termRefs := range idx.termIndex {
		if strings.Contains(t.value, text) {
			for _, ref := range termRefs {
				if _, ok := seen[ref]; ok {
					continue
				}
				seen[ref] = struct{}{}
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

// allRefs returns references to every entity in the index.
func (idx *Index) allRefs() []entityRef {
	refs := make([]entityRef, 0, len(idx.fingers)+len(idx.aliases)+len(idx.templates))
	for i := range idx.fingers {
		refs = append(refs, entityRef{kind: entityFinger, id: i})
	}
	for i := range idx.aliases {
		refs = append(refs, entityRef{kind: entityAlias, id: i})
	}
	for i := range idx.templates {
		refs = append(refs, entityRef{kind: entityTemplate, id: i})
	}
	return refs
}

// FingerWithCount pairs a fingerprint with its associated template count.
type FingerWithCount struct {
	Finger        *fingersEngine.Finger
	TemplateCount int
}

// FingersWithTemplates returns fingerprints from the result that have
// at least one associated template, along with the count. Requires the
// index to resolve associations.
func (r *QueryResult) FingersWithTemplates(idx *Index) []FingerWithCount {
	if r == nil || idx == nil {
		return nil
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var out []FingerWithCount
	for _, f := range r.Fingers {
		if f == nil {
			continue
		}
		id, ok := idx.fingerByName[nameKey(f.Name)]
		if !ok {
			continue
		}
		count := len(idx.fingerTemplates[id])
		if count > 0 {
			out = append(out, FingerWithCount{Finger: f, TemplateCount: count})
		}
	}
	return out
}
