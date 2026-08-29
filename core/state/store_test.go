package state

import "testing"

func TestSetGet(t *testing.T) {
	s := New()
	s.Set("state.form.login.email", "a@b.c")
	v, ok := s.Get("state.form.login.email")
	if !ok || v != "a@b.c" {
		t.Fatalf("get: got %v %v", v, ok)
	}
	_, ok = s.Get("state.missing")
	if ok {
		t.Fatal("expected missing")
	}
}

func TestSubscribe(t *testing.T) {
	s := New()
	called := 0
	s.Subscribe("state.form.login", func() { called++ })
	s.Set("state.form.login.email", "x")
	if called != 1 {
		t.Fatalf("expected 1 call, got %d", called)
	}
	// изменение вне префикса не должно триггерить
	s.Set("state.other", 1)
	if called != 1 {
		t.Fatalf("expected still 1 call, got %d", called)
	}
}

func TestNestedMap(t *testing.T) {
	s := New()
	s.Set("state.a.b.c", "deep")
	v, ok := s.Get("state.a.b.c")
	if !ok || v != "deep" {
		t.Fatalf("nested: %v %v", v, ok)
	}
}
