package main

import (
	"context"
	"fmt"
	"io"
	"os"
)

func main() {
	run(context.Background(), os.Stdout)
}

func run(ctx context.Context, w io.Writer) {
	email := &EmailChannel{}
	slack := &SlackChannel{}
	sms := &SMSChannel{}

	sysAlert := SystemAlert{
		BaseAlert: BaseAlert{
			Title:    "CPU acima de 90%",
			Message:  "Host worker-03 com alta carga por 5min",
			Priority: UrgencyCritical,
		},
		Service: "payments-api",
	}
	if err := sysAlert.Dispatch(ctx, "#ops-crit", slack); err != nil {
		fmt.Fprintln(w, "erro:", err)
	}
	if err := sysAlert.Dispatch(ctx, "ops@example.com", email); err != nil {
		fmt.Fprintln(w, "erro:", err)
	}

	mkt := MarketingAlert{
		BaseAlert: BaseAlert{
			Title:    "20% OFF hoje",
			Message:  "Aproveite antes das 23h",
			Priority: UrgencyInfo,
		},
		Campaign: "BlackDay",
	}
	if err := mkt.Dispatch(ctx, "+5511999990000", sms); err != nil {
		fmt.Fprintln(w, "erro:", err)
	}

	for _, line := range email.Sent {
		fmt.Fprintln(w, line)
	}
	for _, line := range slack.Sent {
		fmt.Fprintln(w, line)
	}
	for _, line := range sms.Sent {
		fmt.Fprintln(w, line)
	}
}
