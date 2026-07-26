package apitests

import (
	"net/http"
	"testing"

	"avenue/backend/sdk"
)

// findItem returns the item named name (of the given type) in items, or nil.
func findItem(items []sdk.FolderItem, itemType, name string) *sdk.FolderItem {
	for i := range items {
		if items[i].Type == itemType && items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}

func listRoot(t *testing.T, client *sdk.Client, h http.Header) []sdk.FolderItem {
	t.Helper()
	contents, err := client.ListFolderContents(h, "", 1, 200)
	if err != nil {
		t.Fatalf("ListFolderContents: %v", err)
	}
	return contents.Items
}

func TestFolderLifecycle(t *testing.T) {
	client, h := authedClient(t)
	name := uniqueName("folder")

	if err := client.CreateFolder(h, name, ""); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	folder := findItem(listRoot(t, client, h), "folder", name)
	if folder == nil {
		t.Fatalf("created folder %q not found in root listing", name)
	}

	// Rename.
	newName := name + "-renamed"
	if err := client.UpdateFolderName(h, folder.UUID, newName); err != nil {
		t.Fatalf("UpdateFolderName: %v", err)
	}
	if findItem(listRoot(t, client, h), "folder", newName) == nil {
		t.Fatalf("renamed folder %q not found in root listing", newName)
	}

	// Trash it.
	if _, err := client.DeleteFolder(h, folder.UUID); err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}
	if findItem(listRoot(t, client, h), "folder", newName) != nil {
		t.Fatalf("trashed folder %q still present in root listing", newName)
	}

	trash, err := client.ListTrash(h, 1, 200)
	if err != nil {
		t.Fatalf("ListTrash: %v", err)
	}
	if findItem(trash.Items, "folder", newName) == nil {
		t.Fatalf("trashed folder %q not found in trash listing", newName)
	}

	// Restore it.
	if err := client.RestoreFolder(h, folder.UUID); err != nil {
		t.Fatalf("RestoreFolder: %v", err)
	}
	if findItem(listRoot(t, client, h), "folder", newName) == nil {
		t.Fatalf("restored folder %q not found in root listing", newName)
	}

	// Clean up: trash + permanently purge.
	if _, err := client.DeleteFolder(h, folder.UUID); err != nil {
		t.Fatalf("DeleteFolder (cleanup): %v", err)
	}
	if err := client.PurgeFolder(h, folder.UUID); err != nil {
		t.Fatalf("PurgeFolder (cleanup): %v", err)
	}
}

func TestFolderListSortByName(t *testing.T) {
	client, h := authedClient(t)

	// Prefixed so alphabetical order is unambiguous regardless of whatever
	// else already exists in the account.
	first := uniqueName("aaa-sort-first")
	second := uniqueName("zzz-sort-second")

	if err := client.CreateFolder(h, first, ""); err != nil {
		t.Fatalf("CreateFolder(first): %v", err)
	}
	if err := client.CreateFolder(h, second, ""); err != nil {
		t.Fatalf("CreateFolder(second): %v", err)
	}
	t.Cleanup(func() {
		for _, name := range []string{first, second} {
			if item := findItem(listRoot(t, client, h), "folder", name); item != nil {
				_, _ = client.DeleteFolder(h, item.UUID)
				_ = client.PurgeFolder(h, item.UUID)
			}
		}
	})

	contents, err := client.ListFolderContents(h, "", 1, 200)
	if err != nil {
		t.Fatalf("ListFolderContents: %v", err)
	}

	indexOf := func(items []sdk.FolderItem, name string) int {
		for i, item := range items {
			if item.Name == name {
				return i
			}
		}
		return -1
	}

	firstIdx := indexOf(contents.Items, first)
	secondIdx := indexOf(contents.Items, second)
	if firstIdx == -1 || secondIdx == -1 {
		t.Fatalf("expected both test folders in listing, got firstIdx=%d secondIdx=%d", firstIdx, secondIdx)
	}
	if firstIdx >= secondIdx {
		t.Errorf("expected %q (idx %d) to sort before %q (idx %d) in default (name asc) order", first, firstIdx, second, secondIdx)
	}
}
