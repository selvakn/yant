package models_test

import (
	"testing"

	"github.com/selvakn/yant/internal/models"
)

func TestWriteAuditLog_Appends(t *testing.T) {
	db := openTestDB(t)
	if err := models.WriteAuditLog(db, "admin1", models.AuditDisableUser, "user", "3", "reason"); err != nil {
		t.Fatalf("WriteAuditLog: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE admin_username = ? AND action = ?`, "admin1", models.AuditDisableUser).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row, got %d", n)
	}
}

func TestBootstrapAdmin_PromotesExistingUser(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	ok, err := models.BootstrapAdmin(db, "alice")
	if err != nil {
		t.Fatalf("BootstrapAdmin: %v", err)
	}
	if !ok {
		t.Fatal("expected promotion")
	}
	if !models.IsUserAdmin(db, u.ID) {
		t.Error("expected admin after bootstrap")
	}
	ok, err = models.BootstrapAdmin(db, "nobody-xyz")
	if err != nil {
		t.Fatalf("BootstrapAdmin missing: %v", err)
	}
	if ok {
		t.Error("expected false for missing username")
	}
}

func TestIsUserAdmin_WhenNotAdmin(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "bob")
	if models.IsUserAdmin(db, u.ID) {
		t.Error("expected non-admin")
	}
}

func TestIsUserDisabled_ReflectsDB(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "carol")
	if models.IsUserDisabled(db, u.ID) {
		t.Error("expected active")
	}
	_, err := db.Exec(`UPDATE users SET disabled = 1 WHERE id = ?`, u.ID)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !models.IsUserDisabled(db, u.ID) {
		t.Error("expected disabled")
	}
}

func TestGetDashboardMetrics_ReturnsCounts(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "dave")
	_, _ = models.CreateNote(db, u.ID, "N", "n", 0, true)
	m, err := models.GetDashboardMetrics(db)
	if err != nil {
		t.Fatalf("GetDashboardMetrics: %v", err)
	}
	if m == nil || m.TotalUsers < 1 || m.TotalNotes < 1 {
		t.Fatalf("metrics: %+v", m)
	}
}

func TestIsUserAdmin_FalseWhenDisabled(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "erin")
	_, _ = db.Exec(`UPDATE users SET is_admin = 1, disabled = 1 WHERE id = ?`, u.ID)
	if models.IsUserAdmin(db, u.ID) {
		t.Error("expected non-admin when account disabled")
	}
}

func TestListAllUsers_Search(t *testing.T) {
	db := openTestDB(t)
	_, _ = models.GetOrCreateUser(db, "u1search")
	_, _ = models.GetOrCreateUser(db, "u2other")
	users, total, err := models.ListAllUsers(db, "search", 1)
	if err != nil {
		t.Fatalf("ListAllUsers: %v", err)
	}
	if total != 1 || len(users) != 1 || users[0].Username != "u1search" {
		t.Fatalf("unexpected: total=%d users=%+v", total, users)
	}
}

func TestListAllUsers_NoSearch(t *testing.T) {
	db := openTestDB(t)
	_, _ = models.GetOrCreateUser(db, "uAll1")
	_, _ = models.GetOrCreateUser(db, "uAll2")
	users, total, err := models.ListAllUsers(db, "", 1)
	if err != nil {
		t.Fatalf("ListAllUsers: %v", err)
	}
	if total < 2 || len(users) < 2 {
		t.Fatalf("unexpected: total=%d users=%d", total, len(users))
	}
}

func TestListAllUsers_InvalidPage(t *testing.T) {
	db := openTestDB(t)
	_, _ = models.GetOrCreateUser(db, "uPage")
	users, _, err := models.ListAllUsers(db, "", -1)
	if err != nil {
		t.Fatalf("ListAllUsers: %v", err)
	}
	if len(users) == 0 {
		t.Fatal("expected at least one user with page -1 corrected to 1")
	}
}

func TestListAllNotes_OwnerFilter(t *testing.T) {
	db := openTestDB(t)
	ua, _ := models.GetOrCreateUser(db, "ownerA")
	ub, _ := models.GetOrCreateUser(db, "ownerB")
	_, _ = models.CreateNote(db, ua.ID, "A", "a", 0, true)
	_, _ = models.CreateNote(db, ub.ID, "B", "b", 0, true)
	notes, total, err := models.ListAllNotes(db, "ownerA", "", "", 1)
	if err != nil {
		t.Fatalf("ListAllNotes: %v", err)
	}
	if total != 1 || len(notes) != 1 || notes[0].Title != "A" {
		t.Fatalf("unexpected: %v", notes)
	}
}

func TestGetAdminUserDetail_ExistingUser(t *testing.T) {
	db := openTestDB(t)
	_, _ = models.GetOrCreateUser(db, "detailUser")
	u, err := models.GetAdminUserDetail(db, "detailUser")
	if err != nil {
		t.Fatalf("GetAdminUserDetail: %v", err)
	}
	if u == nil || u.Username != "detailUser" {
		t.Fatalf("unexpected user: %+v", u)
	}
}

func TestGetAdminUserDetail_Missing(t *testing.T) {
	db := openTestDB(t)
	u, err := models.GetAdminUserDetail(db, "noone")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if u != nil {
		t.Fatalf("expected nil, got %+v", u)
	}
}

func TestDisableUser_SetsFlag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "disableMe")
	if err := models.DisableUser(db, u.ID); err != nil {
		t.Fatalf("DisableUser: %v", err)
	}
	if !models.IsUserDisabled(db, u.ID) {
		t.Error("expected disabled")
	}
}

func TestEnableUser_ClearsFlag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "enableMe")
	_ = models.DisableUser(db, u.ID)
	if err := models.EnableUser(db, u.ID); err != nil {
		t.Fatalf("EnableUser: %v", err)
	}
	if models.IsUserDisabled(db, u.ID) {
		t.Error("expected active")
	}
}

func TestPromoteAdmin_SetsFlag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "promoteMe")
	if err := models.PromoteAdmin(db, u.ID); err != nil {
		t.Fatalf("PromoteAdmin: %v", err)
	}
	if !models.IsUserAdmin(db, u.ID) {
		t.Error("expected admin")
	}
}

func TestDemoteAdmin_ClearsFlag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "demoteMe")
	_ = models.PromoteAdmin(db, u.ID)
	if err := models.DemoteAdmin(db, u.ID); err != nil {
		t.Fatalf("DemoteAdmin: %v", err)
	}
	if models.IsUserAdmin(db, u.ID) {
		t.Error("expected non-admin")
	}
}

func TestCountAdminUsers_ReturnsCount(t *testing.T) {
	db := openTestDB(t)
	u1, _ := models.GetOrCreateUser(db, "adm1")
	u2, _ := models.GetOrCreateUser(db, "adm2")
	_ = models.PromoteAdmin(db, u1.ID)
	_ = models.PromoteAdmin(db, u2.ID)
	if c := models.CountAdminUsers(db); c != 2 {
		t.Fatalf("expected 2 admins, got %d", c)
	}
}

func TestCountAdminUsers_ExcludesDisabled(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "disAdmin")
	_ = models.PromoteAdmin(db, u.ID)
	_ = models.DisableUser(db, u.ID)
	if c := models.CountAdminUsers(db); c != 0 {
		t.Fatalf("expected 0 active admins, got %d", c)
	}
}

func TestGetUserImpactSummary_ReturnsCounts(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "impactUser")
	_, _ = models.CreateNote(db, u.ID, "ImpactNote", "impact-note", 0, true)
	s, err := models.GetUserImpactSummary(db, u.ID)
	if err != nil {
		t.Fatalf("GetUserImpactSummary: %v", err)
	}
	if s.Username != "impactUser" || s.NoteCount != 1 {
		t.Fatalf("unexpected summary: %+v", s)
	}
}

func TestDeleteUserCascade_RemovesUser(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "cascadeUser")
	_, _ = models.CreateNote(db, u.ID, "CascadeNote", "cascade-note", 0, true)
	if err := models.DeleteUserCascade(db, u.ID); err != nil {
		t.Fatalf("DeleteUserCascade: %v", err)
	}
	detail, _ := models.GetAdminUserDetail(db, "cascadeUser")
	if detail != nil {
		t.Error("expected user to be gone")
	}
	notes, total, _ := models.ListAllNotes(db, "cascadeUser", "", "", 1)
	if total != 0 || len(notes) != 0 {
		t.Error("expected notes to be deleted")
	}
}

func TestGetNoteForAdmin_ExistingNote(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "noteViewer")
	n, _ := models.CreateNote(db, u.ID, "ViewNote", "view-note", 0, true)
	result, err := models.GetNoteForAdmin(db, n.ID)
	if err != nil {
		t.Fatalf("GetNoteForAdmin: %v", err)
	}
	if result == nil || result.Title != "ViewNote" || result.OwnerUsername != "noteViewer" {
		t.Fatalf("unexpected: %+v", result)
	}
}

func TestGetNoteForAdmin_Missing(t *testing.T) {
	db := openTestDB(t)
	result, err := models.GetNoteForAdmin(db, 99999)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil, got %+v", result)
	}
}

func TestGetNoteImpactSummary_ReturnsCounts(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "impactOwner")
	n, _ := models.CreateNote(db, u.ID, "ImpNote", "imp-note", 0, true)
	s, err := models.GetNoteImpactSummary(db, n.ID)
	if err != nil {
		t.Fatalf("GetNoteImpactSummary: %v", err)
	}
	if s.Title != "ImpNote" || s.Owner != "impactOwner" {
		t.Fatalf("unexpected: %+v", s)
	}
}

func TestAdminDeleteNote_RemovesNote(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "delNoteOwner")
	n, _ := models.CreateNote(db, u.ID, "DelMe", "del-me", 0, true)
	if err := models.AdminDeleteNote(db, n.ID); err != nil {
		t.Fatalf("AdminDeleteNote: %v", err)
	}
	result, _ := models.GetNoteForAdmin(db, n.ID)
	if result != nil {
		t.Error("expected note to be deleted")
	}
}

func TestListAllNotes_NoFilters(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "allNotesUser")
	_, _ = models.CreateNote(db, u.ID, "X", "x", 0, true)
	_, _ = models.CreateNote(db, u.ID, "Y", "y", 0, true)
	notes, total, err := models.ListAllNotes(db, "", "", "", 1)
	if err != nil {
		t.Fatalf("ListAllNotes: %v", err)
	}
	if total < 2 || len(notes) < 2 {
		t.Fatalf("expected >= 2: total=%d notes=%d", total, len(notes))
	}
}

func TestListAllNotes_PublicFilter(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "pubFilterUser")
	n1, _ := models.CreateNote(db, u.ID, "Pub", "pub", 0, true)
	_, _ = models.CreateNote(db, u.ID, "Priv", "priv", 0, true)
	_, _ = models.PublishNote(db, n1.ID)

	pubNotes, pubTotal, _ := models.ListAllNotes(db, "pubFilterUser", "yes", "", 1)
	if pubTotal != 1 || len(pubNotes) != 1 || pubNotes[0].Title != "Pub" {
		t.Fatalf("public filter: total=%d notes=%+v", pubTotal, pubNotes)
	}
	privNotes, privTotal, _ := models.ListAllNotes(db, "pubFilterUser", "no", "", 1)
	if privTotal != 1 || len(privNotes) != 1 || privNotes[0].Title != "Priv" {
		t.Fatalf("no-public filter: total=%d notes=%+v", privTotal, privNotes)
	}
}

func TestListAllNotes_SharedFilter(t *testing.T) {
	db := openTestDB(t)
	owner, _ := models.GetOrCreateUser(db, "shareFilterOwner")
	collab, _ := models.GetOrCreateUser(db, "shareFilterCollab")
	n1, _ := models.CreateNote(db, owner.ID, "Shared", "shared", 0, true)
	_, _ = models.CreateNote(db, owner.ID, "Unshared", "unshared", 0, true)
	_ = models.GrantShare(db, n1.ID, collab.ID, owner.ID, "read")

	sharedNotes, sharedTotal, _ := models.ListAllNotes(db, "shareFilterOwner", "", "yes", 1)
	if sharedTotal != 1 || len(sharedNotes) != 1 || sharedNotes[0].Title != "Shared" {
		t.Fatalf("shared filter: total=%d notes=%+v", sharedTotal, sharedNotes)
	}
	unsharedNotes, unsharedTotal, _ := models.ListAllNotes(db, "shareFilterOwner", "", "no", 1)
	if unsharedTotal != 1 || len(unsharedNotes) != 1 || unsharedNotes[0].Title != "Unshared" {
		t.Fatalf("no-shared filter: total=%d notes=%+v", unsharedTotal, unsharedNotes)
	}
}

func TestListAllPublicNotes_ReturnsPublished(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "pubListUser")
	n, _ := models.CreateNote(db, u.ID, "PublicNote", "public-note", 0, true)
	_, _ = models.PublishNote(db, n.ID)

	notes, total, err := models.ListAllPublicNotes(db, 1)
	if err != nil {
		t.Fatalf("ListAllPublicNotes: %v", err)
	}
	if total < 1 {
		t.Fatalf("expected at least 1 public note, got %d", total)
	}
	found := false
	for _, pn := range notes {
		if pn.NoteTitle == "PublicNote" {
			found = true
		}
	}
	if !found {
		t.Fatal("public note not found in list")
	}
}

func TestListAllPublicNotes_InvalidPage(t *testing.T) {
	db := openTestDB(t)
	_, _, err := models.ListAllPublicNotes(db, -1)
	if err != nil {
		t.Fatalf("error on invalid page: %v", err)
	}
}

func TestListAllShares_NoFilter(t *testing.T) {
	db := openTestDB(t)
	owner, _ := models.GetOrCreateUser(db, "shareListOwner")
	collab, _ := models.GetOrCreateUser(db, "shareListCollab")
	n, _ := models.CreateNote(db, owner.ID, "SharedN", "shared-n", 0, true)
	_ = models.GrantShare(db, n.ID, collab.ID, owner.ID, "edit")

	shares, total, err := models.ListAllShares(db, "", 1)
	if err != nil {
		t.Fatalf("ListAllShares: %v", err)
	}
	if total < 1 {
		t.Fatalf("expected at least 1 share, got %d", total)
	}
	found := false
	for _, s := range shares {
		if s.NoteTitle == "SharedN" && s.CollabUsername == "shareListCollab" {
			found = true
		}
	}
	if !found {
		t.Fatal("share not found")
	}
}

func TestListAllShares_UserFilter(t *testing.T) {
	db := openTestDB(t)
	owner, _ := models.GetOrCreateUser(db, "sFilterOwner")
	collab, _ := models.GetOrCreateUser(db, "sFilterCollab")
	other, _ := models.GetOrCreateUser(db, "sFilterOther")
	n1, _ := models.CreateNote(db, owner.ID, "SN1", "sn1", 0, true)
	n2, _ := models.CreateNote(db, other.ID, "SN2", "sn2", 0, true)
	_ = models.GrantShare(db, n1.ID, collab.ID, owner.ID, "read")
	_ = models.GrantShare(db, n2.ID, collab.ID, other.ID, "read")

	shares, total, _ := models.ListAllShares(db, "sFilterOwner", 1)
	if total != 1 || len(shares) != 1 {
		t.Fatalf("expected 1 share for owner filter, got %d", total)
	}
}

func TestListAllShares_InvalidPage(t *testing.T) {
	db := openTestDB(t)
	_, _, err := models.ListAllShares(db, "", -1)
	if err != nil {
		t.Fatalf("error on invalid page: %v", err)
	}
}

func TestListAuditLog_NoFilter(t *testing.T) {
	db := openTestDB(t)
	_ = models.WriteAuditLog(db, "admin1", models.AuditDisableUser, "user", "1", "")
	_ = models.WriteAuditLog(db, "admin1", models.AuditDeleteNote, "note", "2", "")
	entries, total, err := models.ListAuditLog(db, "", "", 1)
	if err != nil {
		t.Fatalf("ListAuditLog: %v", err)
	}
	if total < 2 || len(entries) < 2 {
		t.Fatalf("expected >= 2: total=%d entries=%d", total, len(entries))
	}
}

func TestListAuditLog_ActionFilter(t *testing.T) {
	db := openTestDB(t)
	_ = models.WriteAuditLog(db, "admin2", models.AuditDisableUser, "user", "1", "")
	_ = models.WriteAuditLog(db, "admin2", models.AuditDeleteNote, "note", "2", "")
	entries, total, _ := models.ListAuditLog(db, models.AuditDisableUser, "", 1)
	if total < 1 {
		t.Fatal("expected at least 1 disable-user entry")
	}
	for _, e := range entries {
		if e.Action != models.AuditDisableUser {
			t.Fatalf("unexpected action %q in filtered results", e.Action)
		}
	}
}

func TestListAuditLog_UserFilter(t *testing.T) {
	db := openTestDB(t)
	_ = models.WriteAuditLog(db, "admin3", models.AuditDeleteUser, "user", "target1", "")
	_ = models.WriteAuditLog(db, "admin3", models.AuditDeleteNote, "note", "other", "")
	entries, total, _ := models.ListAuditLog(db, "", "target1", 1)
	if total != 1 || len(entries) != 1 {
		t.Fatalf("expected 1 entry for target1, got %d", total)
	}
}

func TestListAuditLog_InvalidPage(t *testing.T) {
	db := openTestDB(t)
	_, _, err := models.ListAuditLog(db, "", "", 0)
	if err != nil {
		t.Fatalf("error on invalid page: %v", err)
	}
}
