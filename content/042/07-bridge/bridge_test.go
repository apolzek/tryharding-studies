package main

import (
	"context"
	"strings"
	"testing"
)

func TestAlertDispatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		alert     Alert
		target    string
		newChan   func() Channel
		wantErr   bool
		wantMatch string
	}{
		{
			name:      "system alert via slack",
			alert:     SystemAlert{BaseAlert: BaseAlert{Title: "down", Message: "svc down", Priority: UrgencyCritical}, Service: "api"},
			target:    "#ops",
			newChan:   func() Channel { return &SlackChannel{} },
			wantMatch: "CRIT",
		},
		{
			name:      "marketing via email",
			alert:     MarketingAlert{BaseAlert: BaseAlert{Title: "sale", Message: "buy", Priority: UrgencyInfo}, Campaign: "X"},
			target:    "a@b.com",
			newChan:   func() Channel { return &EmailChannel{} },
			wantMatch: "Campanha X",
		},
		{
			name:    "email rejects invalid target",
			alert:   SystemAlert{BaseAlert: BaseAlert{Title: "x", Message: "y"}, Service: "s"},
			target:  "not-an-email",
			newChan: func() Channel { return &EmailChannel{} },
			wantErr: true,
		},
		{
			name:    "sms rejects short target",
			alert:   MarketingAlert{BaseAlert: BaseAlert{Title: "x", Message: "y"}, Campaign: "Y"},
			target:  "12",
			newChan: func() Channel { return &SMSChannel{} },
			wantErr: true,
		},
		{
			name:    "slack rejects target without #",
			alert:   SystemAlert{BaseAlert: BaseAlert{Title: "x", Message: "y"}, Service: "s"},
			target:  "ops",
			newChan: func() Channel { return &SlackChannel{} },
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ch := tc.newChan()
			err := tc.alert.Dispatch(context.Background(), tc.target, ch)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Verifica que a saída contém a marcação esperada.
			got := sentLines(ch)
			joined := strings.Join(got, " | ")
			if !strings.Contains(joined, tc.wantMatch) {
				t.Fatalf("output %q should contain %q", joined, tc.wantMatch)
			}
		})
	}
}

func TestAlertDispatch_NilChannel(t *testing.T) {
	t.Parallel()
	a := SystemAlert{BaseAlert: BaseAlert{Title: "x", Message: "y"}, Service: "s"}
	if err := a.Dispatch(context.Background(), "ops@example.com", nil); err == nil {
		t.Fatalf("expected error for nil channel")
	}
}

func TestAlertDispatch_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := MarketingAlert{BaseAlert: BaseAlert{Title: "x", Message: "y"}, Campaign: "Z"}
	err := a.Dispatch(ctx, "a@b.com", &EmailChannel{})
	if err == nil {
		t.Fatalf("expected context error")
	}
}

func TestAlertUrgency(t *testing.T) {
	t.Parallel()
	a := SystemAlert{BaseAlert: BaseAlert{Priority: UrgencyWarning}}
	if a.Urgency() != UrgencyWarning {
		t.Errorf("urgency mismatch")
	}
	m := MarketingAlert{BaseAlert: BaseAlert{Priority: UrgencyInfo}}
	if m.Urgency() != UrgencyInfo {
		t.Errorf("urgency mismatch")
	}
}

func TestSMSChannel_BodyShape(t *testing.T) {
	t.Parallel()
	c := &SMSChannel{}
	if err := c.Send(context.Background(), "+5511999990000", "subj", "body"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(c.Sent) != 1 {
		t.Fatalf("sent=%d", len(c.Sent))
	}
	if !strings.Contains(c.Sent[0], "subj: body") {
		t.Fatalf("body not composed: %s", c.Sent[0])
	}
}

func TestAllChannels_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := (&EmailChannel{}).Send(ctx, "a@b.com", "s", "b"); err == nil {
		t.Errorf("email should error on canceled ctx")
	}
	if err := (&SMSChannel{}).Send(ctx, "+5511999999999", "s", "b"); err == nil {
		t.Errorf("sms should error on canceled ctx")
	}
	if err := (&SlackChannel{}).Send(ctx, "#x", "s", "b"); err == nil {
		t.Errorf("slack should error on canceled ctx")
	}
}

func TestUrgencyString(t *testing.T) {
	t.Parallel()
	cases := map[Urgency]string{
		UrgencyInfo:     "INFO",
		UrgencyWarning:  "WARN",
		UrgencyCritical: "CRIT",
		Urgency(99):     "UNKNOWN",
	}
	for u, want := range cases {
		if got := u.String(); got != want {
			t.Errorf("Urgency(%d).String()=%s want %s", u, got, want)
		}
	}
}

func TestRunDemoOutputs(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	run(context.Background(), &buf)
	out := buf.String()
	for _, want := range []string{"EMAIL", "SLACK", "SMS"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q: %s", want, out)
		}
	}
}

func sentLines(ch Channel) []string {
	switch c := ch.(type) {
	case *EmailChannel:
		return c.Sent
	case *SMSChannel:
		return c.Sent
	case *SlackChannel:
		return c.Sent
	}
	return nil
}
