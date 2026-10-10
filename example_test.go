package irnoti_test

import (
	"context"
	"errors"
	"fmt"

	irnoti "github.com/irnoti/go-sms-panel-api"
)

func ExampleClient_sendSMS() {
	c := irnoti.New("irn_test_xxx")
	_, err := c.SMS.Send(context.Background(), irnoti.SendParams{
		LineID: "sandbox", To: []string{"09121234567"}, Text: "Hi", IdempotencyKey: "order-1042",
	})
	var apiErr *irnoti.APIError
	if errors.As(err, &apiErr) && apiErr.IsInsufficientBalance() {
		fmt.Println("top up the wallet")
	}
}
