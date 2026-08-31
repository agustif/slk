package compose

import "testing"

func TestSwapDraft_SavesAndRestoresText(t *testing.T) {
	m := New("a")
	m.SwapDraft("C1")
	m.SetValue("hello a")

	m.SwapDraft("C2")
	if m.Value() != "" {
		t.Fatalf("C2 should start empty, got %q", m.Value())
	}
	if m.DraftKey() != "C2" {
		t.Fatalf("DraftKey = %q, want C2", m.DraftKey())
	}

	m.SwapDraft("C1")
	if m.Value() != "hello a" {
		t.Fatalf("C1 draft not restored, got %q", m.Value())
	}
}

func TestSwapDraft_UnseenKeyIsEmpty(t *testing.T) {
	m := New("a")
	m.SwapDraft("C1")
	m.SetValue("stay in c1")
	m.SwapDraft("C2")
	if m.Value() != "" {
		t.Fatalf("unseen key should restore empty, got %q", m.Value())
	}
}

func TestSwapDraft_SameKeyIsNoop(t *testing.T) {
	m := New("a")
	m.SwapDraft("C1")
	m.SetValue("live")
	m.SwapDraft("C1")
	if m.Value() != "live" {
		t.Fatalf("same-key swap should keep live text, got %q", m.Value())
	}
}

func TestSwapDraft_RestoresAttachments(t *testing.T) {
	m := New("a")
	m.SwapDraft("C1")
	m.SetValue("caption")
	m.AddAttachment(PendingAttachment{Filename: "a.png", Path: "/tmp/a.png", Size: 12})

	m.SwapDraft("C2")
	if len(m.Attachments()) != 0 {
		t.Fatalf("C2 should not inherit attachments, got %d", len(m.Attachments()))
	}

	m.SwapDraft("C1")
	atts := m.Attachments()
	if m.Value() != "caption" {
		t.Fatalf("text = %q, want caption", m.Value())
	}
	if len(atts) != 1 || atts[0].Filename != "a.png" || atts[0].Path != "/tmp/a.png" {
		t.Fatalf("attachments not restored: %+v", atts)
	}
}

func TestSwapDraft_EmptyLiveDeletesStoredDraft(t *testing.T) {
	m := New("a")
	m.SwapDraft("C1")
	m.SetValue("temp")
	m.SwapDraft("C2")
	m.SwapDraft("C1")
	m.SetValue("") // user cleared
	m.SwapDraft("C2")
	m.SwapDraft("C1")
	if m.Value() != "" {
		t.Fatalf("cleared draft should stay gone, got %q", m.Value())
	}
}

func TestReset_DiscardsStoredDraftForCurrentKey(t *testing.T) {
	m := New("a")
	m.SwapDraft("C1")
	m.SetValue("will send")
	m.Reset()
	if m.Value() != "" {
		t.Fatal("Reset should clear live text")
	}
	m.SwapDraft("C2")
	m.SwapDraft("C1")
	if m.Value() != "" {
		t.Fatalf("Reset should drop the stored draft, got %q", m.Value())
	}
}

func TestSwapDraft_ParkAndRestore(t *testing.T) {
	m := New("a")
	m.SwapDraft("C1")
	m.SetValue("park me")
	m.SwapDraft("")
	if m.Value() != "" {
		t.Fatalf("park should clear live, got %q", m.Value())
	}
	if m.DraftKey() != "" {
		t.Fatalf("park should unbind, DraftKey=%q", m.DraftKey())
	}
	m.SwapDraft("C1")
	if m.Value() != "park me" {
		t.Fatalf("parked draft not restored, got %q", m.Value())
	}
}

func TestBindDraftKey_NoopWhenEmptyLive(t *testing.T) {
	m := New("a")
	m.SwapDraft("C1")
	m.SetValue("saved")
	m.SwapDraft("")
	// Parked: live empty. Binding the old channel must not steal
	// attribution of the empty box (which would then save "" over the draft).
	m.BindDraftKey("C1")
	if m.DraftKey() != "" {
		t.Fatalf("BindDraftKey on empty live should no-op, got %q", m.DraftKey())
	}
	m.SwapDraft("C2")
	m.SwapDraft("C1")
	if m.Value() != "saved" {
		t.Fatalf("parked draft clobbered by empty bind, got %q", m.Value())
	}
}

func TestBindDraftKey_AttributesUnboundLiveText(t *testing.T) {
	m := New("a")
	m.SetValue("typed before bind")
	m.BindDraftKey("C1")
	m.SwapDraft("C2")
	m.SwapDraft("C1")
	if m.Value() != "typed before bind" {
		t.Fatalf("unbound live text was not saved under C1, got %q", m.Value())
	}
}

func TestBindDraftKey_DoesNotOverwriteExistingKey(t *testing.T) {
	m := New("a")
	m.SetValue("c1")
	m.BindDraftKey("C1")
	m.BindDraftKey("C2")
	if m.DraftKey() != "C1" {
		t.Fatalf("BindDraftKey should not replace a bound key, got %q", m.DraftKey())
	}
}
