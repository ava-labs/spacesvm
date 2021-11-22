package create

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/ava-labs/quarkvm/crypto/ed25519"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	privateKeyFile = ".quark-cli-pk"
)

func init() {
	cobra.EnablePrefixMatching = true
}

// NewCommand implements "quark-cli" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [options]",
		Short: "Creates a new key in the default location",
		Long: `
Creates a new key in the default location.

$ quark-cli create

`,
		RunE: createFunc,
	}
	return cmd
}

func getPKLocation() (string, error) {
	p, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return path.Join(p, privateKeyFile), nil
}

// TODO: run before all functions (erroring if can't load)
func LoadPK() (ed25519.PrivateKey, error) {
	pkLocation, err := getPKLocation()
	if err != nil {
		return nil, err
	}

	pk, err := os.ReadFile(pkLocation)
	if err != nil {
		return nil, err
	}
	return ed25519.LoadPrivateKey(pk)
}

func createFunc(cmd *cobra.Command, args []string) error {
	// Error if key already exists
	pkLocation, err := getPKLocation()
	if err != nil {
		return err
	}
	if _, err := os.Stat(pkLocation); err == nil {
		return fmt.Errorf("file already exists at %s", pkLocation)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Generate new key and save to disk
	// TODO: encrypt key
	pk, err := ed25519.NewPrivateKey()
	if err != nil {
		return err
	}
	if err := os.WriteFile(pkLocation, pk.Bytes(), 0644); err != nil {
		return err
	}
	color.Green("created address %s and saved to %s", pk.PublicKey().Address(), pkLocation)
	return nil
}
