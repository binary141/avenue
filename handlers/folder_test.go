package handlers

import (
	"reflect"
	"testing"

	"avenue/backend/sdk"
	"avenue/backend/shared"
)

func TestBuildBreadcrumbs(t *testing.T) {
	root := sdk.Folder{UUID: shared.ROOTFOLDERID, Name: "root"}
	docs := sdk.Folder{UUID: "docs-uuid", Name: "Docs"}
	photos := sdk.Folder{UUID: "photos-uuid", Name: "Photos"}

	tests := []struct {
		name          string
		folderID      string
		folderParents []sdk.Folder
		want          []sdk.Breadcrumb
	}{
		{
			name:          "root folder only gets the Drive crumb",
			folderID:      "",
			folderParents: []sdk.Folder{root},
			want: []sdk.Breadcrumb{
				{Label: "Drive", FolderID: ""},
			},
		},
		{
			name:          "single-level subfolder gets a leading Drive crumb",
			folderID:      "docs-uuid",
			folderParents: []sdk.Folder{docs, root},
			want: []sdk.Breadcrumb{
				{Label: "Drive", FolderID: ""},
				{Label: "Docs", FolderID: "docs-uuid"},
			},
		},
		{
			name:          "nested subfolder crumbs come back leaf-first and are reordered root-first",
			folderID:      "photos-uuid",
			folderParents: []sdk.Folder{photos, docs, root},
			want: []sdk.Breadcrumb{
				{Label: "Drive", FolderID: ""},
				{Label: "Docs", FolderID: "docs-uuid"},
				{Label: "Photos", FolderID: "photos-uuid"},
			},
		},
		{
			name:          "folderID equal to root UUID still just gets the Drive crumb",
			folderID:      shared.ROOTFOLDERID,
			folderParents: []sdk.Folder{root},
			want: []sdk.Breadcrumb{
				{Label: "Drive", FolderID: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBreadcrumbs(tt.folderID, tt.folderParents)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildBreadcrumbs() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
