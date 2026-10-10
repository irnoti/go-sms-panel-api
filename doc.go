// Package irnoti is the official Go SDK for the irnoti platform API (https://irnoti.com/docs).
//
//	c := irnoti.New(os.Getenv("IRNOTI_API_KEY"))
//	res, err := c.SMS.Send(ctx, irnoti.SendParams{LineID: "1", To: []string{"09121234567"}, Text: "Hi"})
//
// Money values are integer Rial amounts encoded as strings.
package irnoti
