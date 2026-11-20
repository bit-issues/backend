package events_test

import (
	"testing"
	"time"

	"github.com/bit-issues/backend/internal/events"
	"go.uber.org/zap"
)

const (
	testTimeout = time.Second
	// overflowCount exceeds subscriberBuffer so Publish starts dropping for a
	// consumer that never reads.
	overflowCount = 200
)

func newTestHub() *events.Hub {
	return events.NewHub(zap.NewNop())
}

func receiveEvent(t *testing.T, ch <-chan events.Event) (events.Event, bool) {
	t.Helper()

	select {
	case ev, ok := <-ch:
		return ev, ok
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for event")
		return events.Event{}, false
	}
}

func TestHubPublishReachesSubscriber(t *testing.T) {
	t.Parallel()

	hub := newTestHub()
	t.Cleanup(hub.Close)

	ch, cancel := hub.Subscribe()
	t.Cleanup(cancel)

	want := events.Event{Type: events.TypeCommentCreated, TaskID: 1, CommentID: 2}
	hub.Publish(want)

	got, ok := receiveEvent(t, ch)
	if !ok {
		t.Fatal("channel closed before event arrived")
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestHubPublishReachesAllSubscribers(t *testing.T) {
	t.Parallel()

	hub := newTestHub()
	t.Cleanup(hub.Close)

	first, cancelFirst := hub.Subscribe()
	t.Cleanup(cancelFirst)
	second, cancelSecond := hub.Subscribe()
	t.Cleanup(cancelSecond)

	want := events.Event{Type: events.TypeCommentUpdated, TaskID: 5, CommentID: 6}
	hub.Publish(want)

	for _, ch := range []<-chan events.Event{first, second} {
		got, ok := receiveEvent(t, ch)
		if !ok {
			t.Fatal("channel closed before event arrived")
		}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	}
}

func TestHubPublishDoesNotBlockOnFullBuffer(t *testing.T) {
	t.Parallel()

	hub := newTestHub()
	t.Cleanup(hub.Close)

	_, cancel := hub.Subscribe()
	t.Cleanup(cancel)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range overflowCount {
			hub.Publish(events.Event{Type: events.TypeCommentCreated, TaskID: int64(i), CommentID: int64(i)})
		}
	}()

	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("Publish blocked on a subscriber with a full buffer")
	}
}

func TestHubUnsubscribeClosesChannel(t *testing.T) {
	t.Parallel()

	hub := newTestHub()
	t.Cleanup(hub.Close)

	ch, cancel := hub.Subscribe()
	cancel()
	cancel() // idempotent

	if _, ok := receiveEvent(t, ch); ok {
		t.Fatal("expected channel to be closed after cancel")
	}

	hub.Publish(events.Event{Type: events.TypeCommentCreated, TaskID: 1, CommentID: 1})

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected no delivery after cancel")
		}
	case <-time.After(testTimeout):
		t.Fatal("expected closed channel after cancel")
	}
}

func TestHubCloseClosesSubscribers(t *testing.T) {
	t.Parallel()

	hub := newTestHub()

	ch, cancel := hub.Subscribe()
	t.Cleanup(cancel)

	hub.Close()
	hub.Close() // idempotent

	if _, ok := receiveEvent(t, ch); ok {
		t.Fatal("expected channel to be closed after Close")
	}
}

func TestHubPublishAfterCloseIsNoOp(t *testing.T) {
	t.Parallel()

	hub := newTestHub()
	hub.Close()

	hub.Publish(events.Event{Type: events.TypeCommentDeleted, TaskID: 1, CommentID: 1})
}

func TestHubSubscribeAfterCloseReturnsClosedChannel(t *testing.T) {
	t.Parallel()

	hub := newTestHub()
	hub.Close()

	ch, cancel := hub.Subscribe()
	t.Cleanup(cancel)

	if _, ok := receiveEvent(t, ch); ok {
		t.Fatal("expected already-closed channel from a closed hub")
	}
}
