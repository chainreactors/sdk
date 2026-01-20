package neutron

import "github.com/chainreactors/neutron/templates"

// Templates wraps neutron templates with helper APIs.
type Templates struct {
	Items map[string]*templates.Template
}

// Templates returns template list.
func (t Templates) Templates() []*templates.Template {
	if len(t.Items) == 0 {
		return nil
	}
	out := make([]*templates.Template, 0, len(t.Items))
	for _, item := range t.Items {
		out = append(out, item)
	}
	return out
}

// Len returns item count.
func (t Templates) Len() int {
	return len(t.Items)
}

// Append adds a single template.
func (t Templates) Append(item *templates.Template) Templates {
	if item == nil {
		return t
	}
	if t.Items == nil {
		t.Items = make(map[string]*templates.Template)
	}
	key := templateKey(item)
	if key != "" {
		t.Items[key] = item
	}
	return t
}

// Merge appends templates into Templates.
func (t Templates) Merge(tpls []*templates.Template) Templates {
	if len(tpls) == 0 {
		return t
	}
	if t.Items == nil {
		t.Items = make(map[string]*templates.Template)
	}
	for _, item := range tpls {
		t = t.Append(item)
	}
	return t
}

// Filter returns a filtered copy of Templates using predicate.
func (t Templates) Filter(predicate func(*templates.Template) bool) Templates {
	if predicate == nil || len(t.Items) == 0 {
		return t
	}
	filtered := Templates{
		Items: make(map[string]*templates.Template),
	}
	for key, item := range t.Items {
		if predicate(item) {
			filtered.Items[key] = item
		}
	}
	return filtered
}

func templateKey(item *templates.Template) string {
	if item == nil {
		return ""
	}
	if item.Info.Name != "" {
		return item.Info.Name
	}
	return item.Id
}
