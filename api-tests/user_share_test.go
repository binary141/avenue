package apitests

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"avenue/backend/sdk"
)

// wantStatus fails the test unless err is an *sdk.APIError with the given
// HTTP status. Sharing is mostly about what a user must *not* be able to do,
// so the tests assert on the exact refusal (403 vs 404) rather than just "an
// error".
func wantStatus(t *testing.T, what string, err error, status int) {
	t.Helper()
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("%s: want HTTP %d, got err=%v", what, status, err)
	}
	if apiErr.StatusCode != status {
		t.Fatalf("%s: want HTTP %d, got %d (%v)", what, status, apiErr.StatusCode, err)
	}
}

// testUser is one of the accounts the sharing tests act as.
type testUser struct {
	h     http.Header
	email string
}

var (
	sharedUsersOnce sync.Once
	sharedReady     bool
	sharedClient    *sdk.Client
	sharedAlice     testUser
	sharedBob       testUser
	sharedCarol     testUser
)

// sharedUsers returns the same three accounts — alice (who owns things), bob
// (who they're shared with) and carol (who they never are) — to every sharing
// test, logging them in once. Each test works in its own uniquely named
// folders and cleans up after itself, so sharing the accounts is safe, and it
// keeps the suite under the server's login rate limit.
func sharedUsers(t *testing.T) (client *sdk.Client, alice, bob, carol testUser) {
	t.Helper()
	sharedUsersOnce.Do(func() {
		var u testUser
		sharedClient, u.h, u.email = loginNewUser(t)
		sharedAlice = u
		_, u.h, u.email = loginNewUser(t)
		sharedBob = u
		_, u.h, u.email = loginNewUser(t)
		sharedCarol = u
		sharedReady = true
	})
	if !sharedReady {
		t.Skip("skipping: could not set up sharing test users")
	}
	return sharedClient, sharedAlice, sharedBob, sharedCarol
}

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedReady {
		for _, u := range []testUser{sharedAlice, sharedBob, sharedCarol} {
			_, _ = sharedClient.Logout(u.h)
		}
	}
	os.Exit(code)
}

// shareFixture is a folder owned by alice, holding one file and one
// subfolder, that has been shared with bob under a chosen role. carol is a
// third user it's never shared with.
type shareFixture struct {
	client  *sdk.Client
	alice   http.Header
	bob     http.Header
	carol   http.Header
	bobMail string

	folderName string
	folderID   string // alice's shared folder
	fileID     string // file inside it
	subID      string // subfolder inside it
	shareID    int64  // bob's grant on folderID
}

func newShareFixture(t *testing.T, role string) *shareFixture {
	t.Helper()

	client, a, b, c := sharedUsers(t)
	alice, bob, carol, bobMail := a.h, b.h, c.h, b.email

	f := &shareFixture{client: client, alice: alice, bob: bob, carol: carol, bobMail: bobMail}
	f.folderName = uniqueName("shared")

	if err := client.CreateFolder(alice, f.folderName, ""); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	folder := findItem(listRoot(t, client, alice), "folder", f.folderName)
	if folder == nil {
		t.Fatalf("shared folder %q not found", f.folderName)
	}
	f.folderID = folder.UUID

	file, err := client.UploadFileChunked(alice, uniqueName("inside")+".txt", strings.NewReader("alice's file"), int64(len("alice's file")), f.folderID)
	if err != nil {
		t.Fatalf("upload into shared folder: %v", err)
	}
	f.fileID = file.UUID

	if err := client.CreateFolder(alice, "sub", f.folderID); err != nil {
		t.Fatalf("create subfolder: %v", err)
	}
	contents, err := client.ListFolderContents(alice, f.folderID, sdk.ListOptions{Page: 1, Limit: 200})
	if err != nil {
		t.Fatalf("list shared folder: %v", err)
	}
	sub := findItem(contents.Items, "folder", "sub")
	if sub == nil {
		t.Fatalf("subfolder not found")
	}
	f.subID = sub.UUID

	share, err := client.CreateFolderUserShare(alice, f.folderID, sdk.CreateUserShareRequest{Email: bobMail, Role: role})
	if err != nil {
		t.Fatalf("CreateFolderUserShare: %v", err)
	}
	f.shareID = share.ID

	t.Cleanup(func() {
		_ = client.DeleteUserShare(alice, f.shareID)
		if _, err := client.DeleteFolder(alice, f.folderID); err == nil {
			_ = client.PurgeFolder(alice, f.folderID)
		}
	})

	return f
}

func TestUserShareViewer(t *testing.T) {
	f := newShareFixture(t, "viewer")

	// Sees the folder's contents, at every depth, with a viewer role.
	contents, err := f.client.ListFolderContents(f.bob, f.folderID, sdk.ListOptions{Page: 1, Limit: 200})
	if err != nil {
		t.Fatalf("viewer ListFolderContents: %v", err)
	}
	if contents.Access != "viewer" {
		t.Errorf("Access = %q, want viewer", contents.Access)
	}
	if len(contents.Items) != 2 {
		t.Fatalf("viewer sees %d items, want 2 (the file and the subfolder)", len(contents.Items))
	}
	if _, err := f.client.ListFolderContents(f.bob, f.subID, sdk.ListOptions{Page: 1, Limit: 200}); err != nil {
		t.Fatalf("viewer can't list subfolder (grant should inherit): %v", err)
	}

	// Can download.
	resp, err := f.client.DownloadFile(f.bob, f.fileID)
	if err != nil {
		t.Fatalf("viewer DownloadFile: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(body) != "alice's file" {
		t.Errorf("downloaded %q", body)
	}

	// Cannot change anything.
	wantStatus(t, "viewer create folder", f.client.CreateFolder(f.bob, "nope", f.folderID), http.StatusForbidden)
	wantStatus(t, "viewer rename file", f.client.UpdateFileName(f.bob, f.fileID, "renamed.txt"), http.StatusForbidden)
	wantStatus(t, "viewer rename folder", f.client.UpdateFolderName(f.bob, f.subID, "renamed"), http.StatusForbidden)
	wantStatus(t, "viewer trash file", f.client.DeleteFile(f.bob, f.fileID), http.StatusForbidden)
	_, err = f.client.DeleteFolder(f.bob, f.subID)
	wantStatus(t, "viewer trash folder", err, http.StatusForbidden)
	_, err = f.client.UploadFileChunked(f.bob, "up.txt", strings.NewReader("x"), 1, f.folderID)
	wantStatus(t, "viewer upload", err, http.StatusForbidden)

	// Cannot share it onward or see who else it's shared with.
	_, err = f.client.CreateFolderUserShare(f.bob, f.folderID, sdk.CreateUserShareRequest{Email: "x@example.com", Role: "viewer"})
	wantStatus(t, "viewer re-share", err, http.StatusForbidden)
	_, err = f.client.ListFolderUserShares(f.bob, f.folderID)
	wantStatus(t, "viewer list shares", err, http.StatusForbidden)
}

func TestUserShareStrangerSeesNothing(t *testing.T) {
	f := newShareFixture(t, "editor")

	// carol has no grant: everything reads as "not found", not "forbidden",
	// so she can't tell which ids exist.
	_, err := f.client.ListFolderContents(f.carol, f.folderID, sdk.ListOptions{Page: 1, Limit: 200})
	wantStatus(t, "stranger list folder", err, http.StatusNotFound)
	_, err = f.client.DownloadFile(f.carol, f.fileID)
	wantStatus(t, "stranger download", err, http.StatusNotFound)
	wantStatus(t, "stranger rename", f.client.UpdateFileName(f.carol, f.fileID, "x.txt"), http.StatusNotFound)
	_, err = f.client.UploadFileChunked(f.carol, "up.txt", strings.NewReader("x"), 1, f.folderID)
	wantStatus(t, "stranger upload", err, http.StatusNotFound)

	// The legacy single-request upload used to accept any parent uuid without
	// checking it belonged to the caller.
	_, err = f.client.UploadFile(f.carol, "up.txt", strings.NewReader("x"), f.folderID)
	wantStatus(t, "stranger legacy upload", err, http.StatusNotFound)
}

func TestUserShareEditor(t *testing.T) {
	f := newShareFixture(t, "editor")

	contents, err := f.client.ListFolderContents(f.bob, f.folderID, sdk.ListOptions{Page: 1, Limit: 200})
	if err != nil {
		t.Fatalf("editor ListFolderContents: %v", err)
	}
	if contents.Access != "editor" {
		t.Errorf("Access = %q, want editor", contents.Access)
	}

	aliceBefore, err := f.client.GetProfile(f.alice)
	if err != nil {
		t.Fatalf("alice profile: %v", err)
	}
	bobBefore, err := f.client.GetProfile(f.bob)
	if err != nil {
		t.Fatalf("bob profile: %v", err)
	}

	// Create + upload inside the share, including in the nested subfolder.
	if err := f.client.CreateFolder(f.bob, "bobs-folder", f.folderID); err != nil {
		t.Fatalf("editor CreateFolder: %v", err)
	}
	up, err := f.client.UploadFileChunked(f.bob, "bobs.txt", strings.NewReader("from bob"), int64(len("from bob")), f.subID)
	if err != nil {
		t.Fatalf("editor upload: %v", err)
	}

	// The bytes count against the folder owner's quota, not the uploader's.
	aliceAfter, err := f.client.GetProfile(f.alice)
	if err != nil {
		t.Fatalf("alice profile: %v", err)
	}
	bobAfter, err := f.client.GetProfile(f.bob)
	if err != nil {
		t.Fatalf("bob profile: %v", err)
	}
	if got, want := aliceAfter.SpaceUsed-aliceBefore.SpaceUsed, int64(len("from bob")); got != want {
		t.Errorf("alice's space used grew by %d, want %d", got, want)
	}
	if bobAfter.SpaceUsed != bobBefore.SpaceUsed {
		t.Errorf("bob's space used changed from %d to %d for an upload into alice's drive", bobBefore.SpaceUsed, bobAfter.SpaceUsed)
	}

	// Alice sees them, and the upload is recorded as hers: it lives in her
	// drive and counts against her quota.
	aliceSub, err := f.client.ListFolderContents(f.alice, f.subID, sdk.ListOptions{Page: 1, Limit: 200})
	if err != nil {
		t.Fatalf("alice list sub: %v", err)
	}
	if findItem(aliceSub.Items, "file", "bobs.txt") == nil {
		t.Fatalf("alice can't see bob's upload: %+v", aliceSub.Items)
	}
	aliceProfile, err := f.client.GetProfile(f.alice)
	if err != nil {
		t.Fatalf("alice profile: %v", err)
	}
	if up.CreatedBy != aliceProfile.ID {
		t.Errorf("upload CreatedBy = %d, want alice (%d)", up.CreatedBy, aliceProfile.ID)
	}

	// Rename anything inside, and trash things inside.
	if err := f.client.UpdateFileName(f.bob, f.fileID, "renamed-by-bob.txt"); err != nil {
		t.Fatalf("editor rename file: %v", err)
	}
	if err := f.client.DeleteFile(f.bob, up.UUID); err != nil {
		t.Fatalf("editor trash file inside share: %v", err)
	}

	// May not trash or move the shared folder itself: that's the owner's call.
	_, err = f.client.DeleteFolder(f.bob, f.folderID)
	wantStatus(t, "editor trash shared root", err, http.StatusForbidden)

	// ...and may not move things out of the owner's drive into their own
	// (quota and ownership are per drive).
	_, err = f.client.BulkMove(f.bob, sdk.BulkMoveRequest{FileIDs: []string{f.fileID}, Parent: ""})
	wantStatus(t, "editor move out of share", err, http.StatusForbidden)

	// Within the share, moving is fine.
	if err := f.client.MoveFile(f.bob, f.fileID, f.subID); err != nil {
		t.Fatalf("editor move within share: %v", err)
	}

	// Still can't manage sharing or purge.
	_, err = f.client.CreateFolderUserShare(f.bob, f.folderID, sdk.CreateUserShareRequest{Email: "x@example.com", Role: "viewer"})
	wantStatus(t, "editor re-share", err, http.StatusForbidden)
	wantStatus(t, "editor purge", f.client.PurgeFile(f.bob, up.UUID), http.StatusNotFound)
}

func TestUserShareBreadcrumbsStopAtShareRoot(t *testing.T) {
	client, a, b, _ := sharedUsers(t)
	alice, bob, bobMail := a.h, b.h, b.email

	outer, inner := uniqueName("private-outer"), uniqueName("shared-inner")
	if err := client.CreateFolder(alice, outer, ""); err != nil {
		t.Fatal(err)
	}
	outerF := findItem(listRoot(t, client, alice), "folder", outer)
	if err := client.CreateFolder(alice, inner, outerF.UUID); err != nil {
		t.Fatal(err)
	}
	contents, _ := client.ListFolderContents(alice, outerF.UUID, sdk.ListOptions{Page: 1, Limit: 200})
	innerF := findItem(contents.Items, "folder", inner)
	t.Cleanup(func() {
		_, _ = client.DeleteFolder(alice, outerF.UUID)
		_ = client.PurgeFolder(alice, outerF.UUID)
	})

	if _, err := client.CreateFolderUserShare(alice, innerF.UUID, sdk.CreateUserShareRequest{Email: bobMail, Role: "viewer"}); err != nil {
		t.Fatal(err)
	}

	got, err := client.ListFolderContents(bob, innerF.UUID, sdk.ListOptions{Page: 1, Limit: 200})
	if err != nil {
		t.Fatalf("bob list shared folder: %v", err)
	}
	for _, crumb := range got.BreadCrumbs {
		if crumb.Label == outer {
			t.Fatalf("breadcrumbs leak the name of a private ancestor folder: %+v", got.BreadCrumbs)
		}
	}

	// And a grant on the inner folder gives no access to its parent.
	_, err = client.ListFolderContents(bob, outerF.UUID, sdk.ListOptions{Page: 1, Limit: 200})
	wantStatus(t, "list parent of shared folder", err, http.StatusNotFound)
}

func TestUserShareRevokeUpdateAndLeave(t *testing.T) {
	f := newShareFixture(t, "editor")

	// Listed for the owner, and shows up in bob's "shared with me".
	shares, err := f.client.ListFolderUserShares(f.alice, f.folderID)
	if err != nil || len(shares) != 1 || shares[0].GranteeEmail != f.bobMail || shares[0].Role != "editor" {
		t.Fatalf("ListFolderUserShares = %+v, %v", shares, err)
	}
	mine, err := f.client.ListSharedWithMe(f.bob)
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for _, it := range mine {
		if it.UUID == f.folderID {
			seen = true
			if it.Type != "folder" || it.Role != "editor" || it.ShareID != f.shareID {
				t.Errorf("shared-with-me entry = %+v", it)
			}
		}
	}
	if !seen {
		t.Fatalf("shared folder missing from bob's shared-with-me: %+v", mine)
	}

	// Downgrade to viewer: writes stop working immediately.
	if err := f.client.UpdateUserShare(f.alice, f.shareID, sdk.UpdateUserShareRequest{Role: "viewer"}); err != nil {
		t.Fatalf("UpdateUserShare: %v", err)
	}
	wantStatus(t, "downgraded rename", f.client.UpdateFileName(f.bob, f.fileID, "x.txt"), http.StatusForbidden)
	// Only the owner may change roles.
	wantStatus(t, "grantee changes own role", f.client.UpdateUserShare(f.bob, f.shareID, sdk.UpdateUserShareRequest{Role: "editor"}), http.StatusNotFound)

	// Bob can leave; then it's gone for him but not for alice.
	if err := f.client.DeleteUserShare(f.bob, f.shareID); err != nil {
		t.Fatalf("grantee leaving share: %v", err)
	}
	_, err = f.client.ListFolderContents(f.bob, f.folderID, sdk.ListOptions{Page: 1, Limit: 200})
	wantStatus(t, "list after leaving", err, http.StatusNotFound)
	if _, err := f.client.ListFolderContents(f.alice, f.folderID, sdk.ListOptions{Page: 1, Limit: 200}); err != nil {
		t.Fatalf("owner lost access: %v", err)
	}

	// Re-share, then the owner revokes.
	share, err := f.client.CreateFolderUserShare(f.alice, f.folderID, sdk.CreateUserShareRequest{Email: f.bobMail, Role: "viewer"})
	if err != nil {
		t.Fatal(err)
	}
	f.shareID = share.ID
	// A stranger can't revoke it.
	wantStatus(t, "stranger revokes", f.client.DeleteUserShare(f.carol, share.ID), http.StatusNotFound)
	if err := f.client.DeleteUserShare(f.alice, share.ID); err != nil {
		t.Fatalf("owner revoke: %v", err)
	}
	_, err = f.client.DownloadFile(f.bob, f.fileID)
	wantStatus(t, "download after revoke", err, http.StatusNotFound)
}

func TestUserShareValidation(t *testing.T) {
	f := newShareFixture(t, "viewer")
	_, err := f.client.CreateFolderUserShare(f.alice, f.folderID, sdk.CreateUserShareRequest{Email: f.bobMail, Role: "admin"})
	wantStatus(t, "bad role", err, http.StatusBadRequest)
	_, err = f.client.CreateFolderUserShare(f.alice, f.folderID, sdk.CreateUserShareRequest{Email: "nobody-" + uniqueName("x") + "@example.com", Role: "viewer"})
	wantStatus(t, "unknown email", err, http.StatusNotFound)

	profile, err := f.client.GetProfile(f.alice)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.client.CreateFolderUserShare(f.alice, f.folderID, sdk.CreateUserShareRequest{Email: profile.Email, Role: "viewer"})
	wantStatus(t, "share with self", err, http.StatusBadRequest)

	// Sharing again with the same person updates their role instead of
	// failing or duplicating the grant.
	if _, err := f.client.CreateFolderUserShare(f.alice, f.folderID, sdk.CreateUserShareRequest{Email: f.bobMail, Role: "editor"}); err != nil {
		t.Fatalf("re-share: %v", err)
	}
	shares, err := f.client.ListFolderUserShares(f.alice, f.folderID)
	if err != nil || len(shares) != 1 || shares[0].Role != "editor" {
		t.Fatalf("after re-share: %+v, %v", shares, err)
	}

	// Files that aren't yours can't be shared.
	_, err = f.client.CreateFileUserShare(f.carol, f.fileID, sdk.CreateUserShareRequest{Email: f.bobMail, Role: "viewer"})
	wantStatus(t, "share someone else's file", err, http.StatusNotFound)
}

func TestUserShareSingleFile(t *testing.T) {
	client, a, b, _ := sharedUsers(t)
	alice, bob, bobMail := a.h, b.h, b.email

	name := uniqueName("solo") + ".txt"
	file, err := client.UploadFileChunked(alice, name, strings.NewReader("just this"), int64(len("just this")), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.DeleteFile(alice, file.UUID)
		_ = client.PurgeFile(alice, file.UUID)
	})

	// Not visible until shared.
	_, err = client.DownloadFile(bob, file.UUID)
	wantStatus(t, "download before share", err, http.StatusNotFound)

	share, err := client.CreateFileUserShare(alice, file.UUID, sdk.CreateUserShareRequest{Email: bobMail, Role: "viewer"})
	if err != nil {
		t.Fatalf("CreateFileUserShare: %v", err)
	}
	resp, err := client.DownloadFile(bob, file.UUID)
	if err != nil {
		t.Fatalf("download shared file: %v", err)
	}
	_ = resp.Body.Close()
	wantStatus(t, "viewer rename", client.UpdateFileName(bob, file.UUID, "x.txt"), http.StatusForbidden)

	// An editor on a single file may rename it, but trashing or moving the
	// file itself stays with the owner.
	if err := client.UpdateUserShare(alice, share.ID, sdk.UpdateUserShareRequest{Role: "editor"}); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateFileName(bob, file.UUID, "renamed-"+name); err != nil {
		t.Fatalf("editor rename shared file: %v", err)
	}
	wantStatus(t, "editor trash directly shared file", client.DeleteFile(bob, file.UUID), http.StatusForbidden)
}
