package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/steugen/trigger/internal"
)

var signPayload string

var signCmd = &cobra.Command{
	Use:   "sign --payload file",
	Short: "compute HMAC-SHA256 for a payload using env TRIGGER_SECRET",
	RunE: func(cmd *cobra.Command, args []string) error {
		secret := os.Getenv("TRIGGER_SECRET")
		if secret == "" {
			return fmt.Errorf("TRIGGER_SECRET env var is not set")
		}

		var b []byte
		var err error

		if signPayload == "" {
			b, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
		} else {
			b, err = ioutil.ReadFile(signPayload)
			if err != nil {
				return err
			}
		}

		fmt.Println(internal.ComputeHMAC(secret, b))
		return nil
	},
}

func init() {
	signCmd.Flags().StringVar(&signPayload, "payload", "", "path to payload file (default: stdin)")
}
