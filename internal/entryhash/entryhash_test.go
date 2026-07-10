package entryhash

import "testing"

func TestHash_Deterministic(t *testing.T) {
	a := Hash("secret", 42)
	b := Hash("secret", 42)
	if a != b {
		t.Fatalf("Hash not deterministic: %q != %q", a, b)
	}
	if len(a) != 8 {
		t.Fatalf("Hash length = %d, want 8", len(a))
	}
}

func TestVerify_Valid(t *testing.T) {
	h := Hash("secret", 42)
	if !Verify("secret", 42, h) {
		t.Fatal("Verify rejected a valid hash")
	}
}

func TestVerify_WrongPosterID(t *testing.T) {
	h := Hash("secret", 42)
	if Verify("secret", 43, h) {
		t.Fatal("Verify accepted a hash for the wrong poster ID")
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	h := Hash("secret", 42)
	if Verify("other-secret", 42, h) {
		t.Fatal("Verify accepted a hash generated with a different secret")
	}
}

func TestVerify_Tampered(t *testing.T) {
	h := Hash("secret", 42)
	tampered := h[:7] + "0"
	if tampered == h {
		tampered = h[:7] + "1"
	}
	if Verify("secret", 42, tampered) {
		t.Fatal("Verify accepted a tampered hash")
	}
}

func TestVerify_WrongLength(t *testing.T) {
	if Verify("secret", 42, "short") {
		t.Fatal("Verify accepted a too-short token")
	}
	if Verify("secret", 42, Hash("secret", 42)+"extra") {
		t.Fatal("Verify accepted a too-long token")
	}
}
