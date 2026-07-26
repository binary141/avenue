package shared

import "testing"

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name       string
		pageStr    string
		limitStr   string
		wantPage   int
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "valid page and limit",
			pageStr:    "2",
			limitStr:   "10",
			wantPage:   2,
			wantLimit:  10,
			wantOffset: 10,
		},
		{
			name:       "empty page and limit default",
			pageStr:    "",
			limitStr:   "",
			wantPage:   DEFAULTPAGE,
			wantLimit:  DEFAULTPAGELIMIT,
			wantOffset: 0,
		},
		{
			name:       "non-numeric page and limit default",
			pageStr:    "abc",
			limitStr:   "xyz",
			wantPage:   DEFAULTPAGE,
			wantLimit:  DEFAULTPAGELIMIT,
			wantOffset: 0,
		},
		{
			name:       "zero or negative page defaults",
			pageStr:    "0",
			limitStr:   "10",
			wantPage:   DEFAULTPAGE,
			wantLimit:  10,
			wantOffset: 0,
		},
		{
			name:       "negative limit defaults",
			pageStr:    "1",
			limitStr:   "-5",
			wantPage:   1,
			wantLimit:  DEFAULTPAGELIMIT,
			wantOffset: 0,
		},
		{
			name:       "limit above max is clamped",
			pageStr:    "1",
			limitStr:   "1000",
			wantPage:   1,
			wantLimit:  MAXPAGELIMIT,
			wantOffset: 0,
		},
		{
			name:       "offset computed from page and limit",
			pageStr:    "3",
			limitStr:   "25",
			wantPage:   3,
			wantLimit:  25,
			wantOffset: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, limit, offset := ParsePagination(tt.pageStr, tt.limitStr)
			if page != tt.wantPage {
				t.Errorf("page = %d, want %d", page, tt.wantPage)
			}
			if limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", limit, tt.wantLimit)
			}
			if offset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}
