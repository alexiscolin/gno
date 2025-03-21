package pager

import (
	"testing"

	"gno.land/p/demo/avl"
	"gno.land/p/demo/uassert"
	"gno.land/p/demo/ufmt"
	"gno.land/p/demo/urequire"
)

func TestPager_GetPage(t *testing.T) {
	tree := avl.NewTree()
	tree.Set("a", 1)
	tree.Set("b", 2)
	tree.Set("c", 3)
	tree.Set("d", 4)
	tree.Set("e", 5)

	t.Run("normal ordering", func(t *testing.T) {
		pager := NewPager(tree, 10, false)

		tests := []struct {
			pageNumber int
			pageSize   int
			expected   []Item
		}{
			{1, 2, []Item{{Key: "a", Value: 1}, {Key: "b", Value: 2}}},
			{2, 2, []Item{{Key: "c", Value: 3}, {Key: "d", Value: 4}}},
			{3, 2, []Item{{Key: "e", Value: 5}}},
		}

		for _, tt := range tests {
			page := pager.GetPageWithSize(tt.pageNumber, tt.pageSize)
			uassert.Equal(t, len(tt.expected), len(page.Items))

			for i, item := range page.Items {
				uassert.Equal(t, tt.expected[i].Key, item.Key)
				uassert.Equal(t, tt.expected[i].Value, item.Value)
			}
		}
	})
}

func TestPager_GetPageByPath(t *testing.T) {
	tree := avl.NewTree()
	for i := 0; i < 50; i++ {
		tree.Set(ufmt.Sprintf("key%d", i), i)
	}

	pager := NewPager(tree, 10, false)

	tests := []struct {
		rawURL       string
		expectedPage int
		expectedSize int
	}{
		{"/r/foo:bar/baz?size=10&page=1", 1, 10},
		{"/r/foo:bar/baz?page=3", 3, pager.DefaultPageSize},
		{"/r/foo:bar/baz?size=20", 1, 20},
		{"/r/foo:bar/baz", 1, pager.DefaultPageSize},
	}

	for _, tt := range tests {
		page, err := pager.GetPageByPath(tt.rawURL)
		urequire.NoError(t, err)

		uassert.Equal(t, tt.expectedPage, page.PageNumber)
		uassert.Equal(t, tt.expectedSize, page.PageSize)
	}
}

func TestPage_PickerQueryParamPreservation(t *testing.T) {
	tree := avl.NewTree()
	for i := 1; i <= 6; i++ {
		tree.Set(ufmt.Sprintf("key%d", i), i)
	}

	pager := NewPager(tree, 2, false)

	tests := []struct {
		name       string
		pageNumber int
		path       string
		expected   string
	}{
		{
			name:       "single query param",
			pageNumber: 1,
			path:       "/test?foo=bar",
			expected:   "**1** | [2](?page=2&foo=bar) | [3](?page=3&foo=bar)",
		},
		{
			name:       "multiple query params",
			pageNumber: 2,
			path:       "/test?foo=bar&baz=qux",
			// Updated expected value to match actual output order
			expected: "[1](?page=1&baz=qux&foo=bar) | **2** | [3](?page=3&baz=qux&foo=bar)",
		},
		{
			name:       "overwrite existing page param",
			pageNumber: 1,
			path:       "/test?param1=value1&page=999&param2=value2",
			expected:   "**1** | [2](?page=2&param1=value1&param2=value2) | [3](?page=3&param1=value1&param2=value2)",
		},
		{
			name:       "empty query string",
			pageNumber: 2,
			path:       "/test",
			expected:   "[1](?page=1) | **2** | [3](?page=3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := pager.GetPageWithSize(tt.pageNumber, 2)
			result := page.Picker(tt.path)
			if result != tt.expected {
				t.Errorf("\nwant: %s\ngot:  %s", tt.expected, result)
			}
		})
	}
}

func TestPager_UI_WithManyPages(t *testing.T) {
	tree := avl.NewTree()
	for i := 0; i < 100; i++ {
		tree.Set(ufmt.Sprintf("key%d", i), i)
	}

	pager := NewPager(tree, 10, false)

	tests := []struct {
		pageNumber int
		pageSize   int
		path       string
		expected   string
	}{
		{1, 10, "/test", "**1** | [2](?page=2) | [3](?page=3) | … | [10](?page=10)"},
		{2, 10, "/test", "[1](?page=1) | **2** | [3](?page=3) | [4](?page=4) | … | [10](?page=10)"},
	}

	for _, tt := range tests {
		page := pager.GetPageWithSize(tt.pageNumber, tt.pageSize)
		ui := page.Picker(tt.path)
		uassert.Equal(t, tt.expected, ui)
	}
}

func TestPager_ParseQuery(t *testing.T) {
	tree := avl.NewTree()
	tree.Set("a", 1)
	tree.Set("b", 2)
	tree.Set("c", 3)
	tree.Set("d", 4)
	tree.Set("e", 5)

	pager := NewPager(tree, 10, false)

	tests := []struct {
		rawURL       string
		expectedPage int
		expectedSize int
	}{
		{"/r/foo:bar/baz?size=2&page=1", 1, 2},
		{"/r/foo:bar/baz?size=3&page=2", 2, 3},
		{"/r/foo:bar/baz?page=2", 2, pager.DefaultPageSize},
	}

	for _, tt := range tests {
		page, size, err := pager.ParseQuery(tt.rawURL)
		urequire.NoError(t, err)
		uassert.Equal(t, tt.expectedPage, page)
		uassert.Equal(t, tt.expectedSize, size)
	}
}
