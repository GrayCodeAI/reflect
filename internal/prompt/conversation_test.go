package prompt

import "testing"

func TestIsSmallTalk(t *testing.T) {
	for _, message := range []string{"Hi", " Hello! ", "hey there", "hi reflect", "WHO ARE YOU?", "what can you do?", "thanks a lot", "good morning", "bye"} {
		if !IsSmallTalk(message) {
			t.Errorf("conversation was classified as work: %q", message)
		}
	}
	for _, message := range []string{"", "Hi fix bug", "hi run ls", "hey read x", "thanks do it", "hi, inspect this repository", "hello there, who fixes this bug?", "how are you handling retries?", "what can you do about this error?", "hi.js"} {
		if IsSmallTalk(message) {
			t.Errorf("work request was classified as conversation: %q", message)
		}
	}
}
