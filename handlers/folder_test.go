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
			name:          "root folder has no breadcrumbs",
			folderID:      "",
			folderParents: []sdk.Folder{root},
			want:          nil,
		},
		{
			name:          "single-level subfolder gets a crumb and trailing root link",
			folderID:      "docs-uuid",
			folderParents: []sdk.Folder{root, docs},
			want: []sdk.Breadcrumb{
				{Label: "/", FolderID: ""},
				{Label: "Docs", FolderID: "docs-uuid"},
				{Label: "root", FolderID: shared.ROOTFOLDERID},
			},
		},
		{
			name:          "nested subfolder crumbs are leaf-first",
			folderID:      "photos-uuid",
			folderParents: []sdk.Folder{root, docs, photos},
			want: []sdk.Breadcrumb{
				{Label: "/", FolderID: ""},
				{Label: "Photos", FolderID: "photos-uuid"},
				{Label: "Docs", FolderID: "docs-uuid"},
				{Label: "root", FolderID: shared.ROOTFOLDERID},
			},
		},
		{
			name:          "folderID equal to root UUID gets no trailing crumb",
			folderID:      shared.ROOTFOLDERID,
			folderParents: []sdk.Folder{root},
			want: []sdk.Breadcrumb{
				{Label: "root", FolderID: shared.ROOTFOLDERID},
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
